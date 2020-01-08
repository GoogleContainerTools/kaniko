package integration

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/ca"
	"github.com/docker/swarmkit/identity"
	"github.com/docker/swarmkit/log"
	"github.com/docker/swarmkit/manager/encryption"
	"github.com/docker/swarmkit/node"
	"github.com/docker/swarmkit/testutils"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

const opsTimeout = 64 * time.Second

// Cluster is representation of cluster - connected nodes.
type testCluster struct {
	ctx        context.Context
	cancel     context.CancelFunc
	api        *dummyAPI
	nodes      map[string]*testNode
	nodesOrder map[string]int
	errs       chan error
	wg         sync.WaitGroup
	counter    int
	fips       bool
}

var testnameKey struct{}

// NewCluster creates new cluster to which nodes can be added.
// AcceptancePolicy is set to most permissive mode on first manager node added.
func newTestCluster(testname string, fips bool) *testCluster {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, testnameKey, testname)
	c := &testCluster{
		ctx:        ctx,
		cancel:     cancel,
		nodes:      make(map[string]*testNode),
		nodesOrder: make(map[string]int),
		errs:       make(chan error, 1024),
		fips:       fips,
	}
	c.api = &dummyAPI{c: c}
	return c
}

// Stop makes best effort to stop all nodes and close connections to them.
func (c *testCluster) Stop() error {
	c.cancel()
	for _, n := range c.nodes {
		if err := n.Stop(); err != nil {
			return err
		}
	}
	c.wg.Wait()
	close(c.errs)
	for err := range c.errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// RandomManager chooses random manager from cluster.
func (c *testCluster) RandomManager() *testNode {
	var managers []*testNode
	for _, n := range c.nodes {
		if n.IsManager() {
			managers = append(managers, n)
		}
	}
	idx := rand.Intn(len(managers))
	return managers[idx]
}

// AddManager adds a node with the Manager role. The node will function as both
// an agent and a manager. If lateBind is set, the manager is started before a
// remote API port is bound. If rootCA is set, the manager is bootstrapped using
// said root CA.  These settings only apply to the first manager.
func (c *testCluster) AddManager(lateBind bool, rootCA *ca.RootCA) error {
	// first node
	var n *testNode
	if len(c.nodes) == 0 {
		node, err := newTestNode("", "", lateBind, c.fips)
		if err != nil {
			return err
		}
		// generate TLS certs for this manager for bootstrapping, else the node will generate its own CA
		if rootCA != nil {
			if err := generateCerts(node.stateDir, rootCA, identity.NewID(), ca.ManagerRole, identity.NewID(), true); err != nil {
				return err
			}
		}
		n = node
	} else {
		lateBind = false
		joinAddr, err := c.RandomManager().node.RemoteAPIAddr()
		if err != nil {
			return err
		}
		clusterInfo, err := c.GetClusterInfo()
		if err != nil {
			return err
		}
		node, err := newTestNode(joinAddr, clusterInfo.RootCA.JoinTokens.Manager, false, c.fips)
		if err != nil {
			return err
		}
		n = node
	}

	if err := c.AddNode(n); err != nil {
		return err
	}

	if lateBind {
		// Verify that the control API works
		if _, err := c.GetClusterInfo(); err != nil {
			return err
		}
		return n.node.BindRemote(context.Background(), "127.0.0.1:0", "")
	}

	return nil
}

// AddAgent adds node with Agent role(doesn't participate in raft cluster).
func (c *testCluster) AddAgent() error {
	// first node
	if len(c.nodes) == 0 {
		return fmt.Errorf("there is no manager nodes")
	}
	joinAddr, err := c.RandomManager().node.RemoteAPIAddr()
	if err != nil {
		return err
	}
	clusterInfo, err := c.GetClusterInfo()
	if err != nil {
		return err
	}
	node, err := newTestNode(joinAddr, clusterInfo.RootCA.JoinTokens.Worker, false, c.fips)
	if err != nil {
		return err
	}
	return c.AddNode(node)
}

// AddNode adds a new node to the cluster
func (c *testCluster) AddNode(n *testNode) error {
	c.counter++
	if err := c.runNode(n, c.counter); err != nil {
		c.counter--
		return err
	}
	c.nodes[n.node.NodeID()] = n
	c.nodesOrder[n.node.NodeID()] = c.counter
	return nil
}

func (c *testCluster) runNode(n *testNode, nodeOrder int) error {
	ctx := log.WithLogger(c.ctx, log.L.WithFields(
		logrus.Fields{
			"testnode": nodeOrder,
			"testname": c.ctx.Value(testnameKey),
		},
	))

	errCtx, cancel := context.WithCancel(context.Background())
	done := make(chan error)
	defer cancel()
	defer close(done)

	c.wg.Add(2)
	go func() {
		c.errs <- n.node.Start(ctx)
		c.wg.Done()
	}()
	go func(n *node.Node) {
		err := n.Err(errCtx)
		select {
		case <-errCtx.Done():
		default:
			done <- err
		}
		c.wg.Done()
	}(n.node)

	select {
	case <-n.node.Ready():
	case err := <-done:
		return err
	case <-time.After(opsTimeout):
		return fmt.Errorf("node did not ready in time")
	}

	return nil
}

// CreateService creates dummy service.
func (c *testCluster) CreateService(name string, instances int) (string, error) {
	spec := &api.ServiceSpec{
		Annotations: api.Annotations{Name: name},
		Mode: &api.ServiceSpec_Replicated{
			Replicated: &api.ReplicatedService{
				Replicas: uint64(instances),
			},
		},
		Task: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{Image: "alpine", Command: []string{"sh"}},
			},
		},
	}

	resp, err := c.api.CreateService(context.Background(), &api.CreateServiceRequest{Spec: spec})
	if err != nil {
		return "", err
	}
	return resp.Service.ID, nil
}

// Leader returns TestNode for cluster leader.
func (c *testCluster) Leader() (*testNode, error) {
	resp, err := c.api.ListNodes(context.Background(), &api.ListNodesRequest{
		Filters: &api.ListNodesRequest_Filters{
			Roles: []api.NodeRole{api.NodeRoleManager},
		},
	})
	if err != nil {
		return nil, err
	}
	for _, n := range resp.Nodes {
		if n.ManagerStatus.Leader {
			tn, ok := c.nodes[n.ID]
			if !ok {
				return nil, fmt.Errorf("leader id is %s, but it isn't found in test cluster object", n.ID)
			}
			return tn, nil
		}
	}
	return nil, fmt.Errorf("cluster leader is not found in api response")
}

// RemoveNode removes node entirely. It tries to demote managers.
func (c *testCluster) RemoveNode(id string, graceful bool) error {
	node, ok := c.nodes[id]
	if !ok {
		return fmt.Errorf("remove node: node %s not found", id)
	}
	// demote before removal
	if node.IsManager() {
		if err := c.SetNodeRole(id, api.NodeRoleWorker); err != nil {
			return fmt.Errorf("demote manager: %v", err)
		}

	}
	if err := node.Stop(); err != nil {
		return err
	}
	delete(c.nodes, id)
	if graceful {
		if err := testutils.PollFuncWithTimeout(nil, func() error {
			resp, err := c.api.GetNode(context.Background(), &api.GetNodeRequest{NodeID: id})
			if err != nil {
				return fmt.Errorf("get node: %v", err)
			}
			if resp.Node.Status.State != api.NodeStatus_DOWN {
				return fmt.Errorf("node %s is still not down", id)
			}
			return nil
		}, opsTimeout); err != nil {
			return err
		}
	}
	if _, err := c.api.RemoveNode(context.Background(), &api.RemoveNodeRequest{NodeID: id, Force: !graceful}); err != nil {
		return fmt.Errorf("remove node: %v", err)
	}
	return nil
}

// SetNodeRole sets role for node through control api.
func (c *testCluster) SetNodeRole(id string, role api.NodeRole) error {
	node, ok := c.nodes[id]
	if !ok {
		return fmt.Errorf("set node role: node %s not found", id)
	}
	if node.IsManager() && role == api.NodeRoleManager {
		return fmt.Errorf("node is already manager")
	}
	if !node.IsManager() && role == api.NodeRoleWorker {
		return fmt.Errorf("node is already worker")
	}

	var initialTimeout time.Duration
	// version might change between get and update, so retry
	for i := 0; i < 5; i++ {
		time.Sleep(initialTimeout)
		initialTimeout += 500 * time.Millisecond
		resp, err := c.api.GetNode(context.Background(), &api.GetNodeRequest{NodeID: id})
		if err != nil {
			return err
		}
		spec := resp.Node.Spec.Copy()
		spec.DesiredRole = role
		if _, err := c.api.UpdateNode(context.Background(), &api.UpdateNodeRequest{
			NodeID:      id,
			Spec:        spec,
			NodeVersion: &resp.Node.Meta.Version,
		}); err != nil {
			// there possible problems on calling update node because redirecting
			// node or leader might want to shut down
			if grpc.ErrorDesc(err) == "update out of sequence" {
				continue
			}
			return err
		}
		if role == api.NodeRoleManager {
			// wait to become manager
			return testutils.PollFuncWithTimeout(nil, func() error {
				if !node.IsManager() {
					return fmt.Errorf("node is still not a manager")
				}
				return nil
			}, opsTimeout)
		}
		// wait to become worker
		return testutils.PollFuncWithTimeout(nil, func() error {
			if node.IsManager() {
				return fmt.Errorf("node is still not a worker")
			}
			return nil
		}, opsTimeout)
	}
	return fmt.Errorf("set role %s for node %s, got sequence error 5 times", role, id)
}

// Starts a node from a stopped state
func (c *testCluster) StartNode(id string) error {
	n, ok := c.nodes[id]
	if !ok {
		return fmt.Errorf("set node role: node %s not found", id)
	}
	if err := c.runNode(n, c.nodesOrder[id]); err != nil {
		return err
	}
	if n.node.NodeID() != id {
		return fmt.Errorf("restarted node does not have have the same ID")
	}
	return nil
}

func (c *testCluster) GetClusterInfo() (*api.Cluster, error) {
	clusterInfo, err := c.api.ListClusters(context.Background(), &api.ListClustersRequest{})
	if err != nil {
		return nil, err
	}
	if len(clusterInfo.Clusters) != 1 {
		return nil, fmt.Errorf("number of clusters in storage: %d; expected 1", len(clusterInfo.Clusters))
	}
	return clusterInfo.Clusters[0], nil
}

func (c *testCluster) RotateRootCA(cert, key []byte) error {
	// poll in case something else changes the cluster before we can update it
	return testutils.PollFuncWithTimeout(nil, func() error {
		clusterInfo, err := c.GetClusterInfo()
		if err != nil {
			return err
		}
		newSpec := clusterInfo.Spec.Copy()
		newSpec.CAConfig.SigningCACert = cert
		newSpec.CAConfig.SigningCAKey = key
		_, err = c.api.UpdateCluster(context.Background(), &api.UpdateClusterRequest{
			ClusterID:      clusterInfo.ID,
			Spec:           newSpec,
			ClusterVersion: &clusterInfo.Meta.Version,
		})
		return err
	}, opsTimeout)
}

func (c *testCluster) RotateUnlockKey() error {
	// poll in case something else changes the cluster before we can update it
	return testutils.PollFuncWithTimeout(nil, func() error {
		clusterInfo, err := c.GetClusterInfo()
		if err != nil {
			return err
		}
		_, err = c.api.UpdateCluster(context.Background(), &api.UpdateClusterRequest{
			ClusterID:      clusterInfo.ID,
			Spec:           &clusterInfo.Spec,
			ClusterVersion: &clusterInfo.Meta.Version,
			Rotation: api.KeyRotation{
				ManagerUnlockKey: true,
			},
		})
		return err
	}, opsTimeout)
}

func (c *testCluster) AutolockManagers(autolock bool) error {
	// poll in case something else changes the cluster before we can update it
	return testutils.PollFuncWithTimeout(nil, func() error {
		clusterInfo, err := c.GetClusterInfo()
		if err != nil {
			return err
		}
		newSpec := clusterInfo.Spec.Copy()
		newSpec.EncryptionConfig.AutoLockManagers = autolock
		_, err = c.api.UpdateCluster(context.Background(), &api.UpdateClusterRequest{
			ClusterID:      clusterInfo.ID,
			Spec:           newSpec,
			ClusterVersion: &clusterInfo.Meta.Version,
		})
		return err
	}, opsTimeout)
}

func (c *testCluster) GetUnlockKey() (string, error) {
	opts := []grpc.DialOption{}
	insecureCreds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
	opts = append(opts, grpc.WithTransportCredentials(insecureCreds))
	opts = append(opts, grpc.WithDialer(
		func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}))
	conn, err := grpc.Dial(c.RandomManager().config.ListenControlAPI, opts...)
	if err != nil {
		return "", err
	}

	resp, err := api.NewCAClient(conn).GetUnlockKey(context.Background(), &api.GetUnlockKeyRequest{})
	if err != nil {
		return "", err
	}

	return encryption.HumanReadableKey(resp.UnlockKey), nil
}
