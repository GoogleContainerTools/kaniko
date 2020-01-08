package integration

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/docker/swarmkit/node"

	"golang.org/x/net/context"

	"reflect"

	"github.com/cloudflare/cfssl/helpers"
	events "github.com/docker/go-events"
	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/ca"
	cautils "github.com/docker/swarmkit/ca/testutils"
	"github.com/docker/swarmkit/identity"
	"github.com/docker/swarmkit/manager"
	"github.com/docker/swarmkit/testutils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

var showTrace = flag.Bool("show-trace", false, "show stack trace after tests finish")

func printTrace() {
	var (
		buf       []byte
		stackSize int
	)
	bufferLen := 16384
	for stackSize == len(buf) {
		buf = make([]byte, bufferLen)
		stackSize = runtime.Stack(buf, true)
		bufferLen *= 2
	}
	buf = buf[:stackSize]
	logrus.Error("===========================STACK TRACE===========================")
	fmt.Println(string(buf))
	logrus.Error("===========================STACK TRACE END=======================")
}

func TestMain(m *testing.M) {
	ca.RenewTLSExponentialBackoff = events.ExponentialBackoffConfig{
		Factor: time.Millisecond * 500,
		Max:    time.Minute,
	}
	flag.Parse()
	res := m.Run()
	if *showTrace {
		printTrace()
	}
	os.Exit(res)
}

// pollClusterReady calls control api until all conditions are true:
// * all nodes are ready
// * all managers has membership == accepted
// * all managers has reachability == reachable
// * one node is leader
// * number of workers and managers equals to expected
func pollClusterReady(t *testing.T, c *testCluster, numWorker, numManager int) {
	pollFunc := func() error {
		res, err := c.api.ListNodes(context.Background(), &api.ListNodesRequest{})
		if err != nil {
			return err
		}
		var mCount int
		var leaderFound bool
		for _, n := range res.Nodes {
			if n.Status.State != api.NodeStatus_READY {
				return fmt.Errorf("node %s with desired role %s isn't ready, status %s, message %s", n.ID, n.Spec.DesiredRole, n.Status.State, n.Status.Message)
			}
			if n.Spec.Membership != api.NodeMembershipAccepted {
				return fmt.Errorf("node %s with desired role %s isn't accepted to cluster, membership %s", n.ID, n.Spec.DesiredRole, n.Spec.Membership)
			}
			if n.Certificate.Role != n.Spec.DesiredRole {
				return fmt.Errorf("node %s had different roles in spec and certificate, %s and %s respectively", n.ID, n.Spec.DesiredRole, n.Certificate.Role)
			}
			if n.Certificate.Status.State != api.IssuanceStateIssued {
				return fmt.Errorf("node %s with desired role %s has no issued certificate, issuance state %s", n.ID, n.Spec.DesiredRole, n.Certificate.Status.State)
			}
			if n.Role == api.NodeRoleManager {
				if n.ManagerStatus == nil {
					return fmt.Errorf("manager node %s has no ManagerStatus field", n.ID)
				}
				if n.ManagerStatus.Reachability != api.RaftMemberStatus_REACHABLE {
					return fmt.Errorf("manager node %s has reachable status: %s", n.ID, n.ManagerStatus.Reachability)
				}
				mCount++
				if n.ManagerStatus.Leader {
					leaderFound = true
				}
			} else {
				if n.ManagerStatus != nil {
					return fmt.Errorf("worker node %s should not have manager status, returned %s", n.ID, n.ManagerStatus)
				}
			}
			if n.Description.TLSInfo == nil {
				return fmt.Errorf("node %s has not reported its TLS info yet", n.ID)
			}
		}
		if !leaderFound {
			return fmt.Errorf("leader of cluster is not found")
		}
		wCount := len(res.Nodes) - mCount
		if mCount != numManager {
			return fmt.Errorf("unexpected number of managers: %d, expected %d", mCount, numManager)
		}
		if wCount != numWorker {
			return fmt.Errorf("unexpected number of workers: %d, expected %d", wCount, numWorker)
		}
		return nil
	}
	err := testutils.PollFuncWithTimeout(nil, pollFunc, opsTimeout)
	require.NoError(t, err)
}

func pollServiceReady(t *testing.T, c *testCluster, sid string, replicas int) {
	pollFunc := func() error {
		req := &api.ListTasksRequest{Filters: &api.ListTasksRequest_Filters{
			ServiceIDs: []string{sid},
		}}
		res, err := c.api.ListTasks(context.Background(), req)
		require.NoError(t, err)

		if len(res.Tasks) == 0 {
			return fmt.Errorf("tasks list is empty")
		}
		var running int
		var states []string
		for _, task := range res.Tasks {
			if task.Status.State == api.TaskStateRunning {
				running++
			}
			states = append(states, fmt.Sprintf("[task %s: %s]", task.ID, task.Status.State))
		}
		if running != replicas {
			return fmt.Errorf("only %d running tasks, but expecting %d replicas: %s", running, replicas, strings.Join(states, ", "))
		}

		return nil
	}
	require.NoError(t, testutils.PollFuncWithTimeout(nil, pollFunc, opsTimeout))
}

func newCluster(t *testing.T, numWorker, numManager int) *testCluster {
	cl := newTestCluster(t.Name(), false)
	for i := 0; i < numManager; i++ {
		require.NoError(t, cl.AddManager(false, nil), "manager number %d", i+1)
	}
	for i := 0; i < numWorker; i++ {
		require.NoError(t, cl.AddAgent(), "agent number %d", i+1)
	}

	pollClusterReady(t, cl, numWorker, numManager)
	return cl
}

func newClusterWithRootCA(t *testing.T, numWorker, numManager int, rootCA *ca.RootCA, fips bool) *testCluster {
	cl := newTestCluster(t.Name(), fips)
	for i := 0; i < numManager; i++ {
		require.NoError(t, cl.AddManager(false, rootCA), "manager number %d", i+1)
	}
	for i := 0; i < numWorker; i++ {
		require.NoError(t, cl.AddAgent(), "agent number %d", i+1)
	}

	pollClusterReady(t, cl, numWorker, numManager)
	return cl
}

func TestClusterCreate(t *testing.T) {
	t.Parallel()

	numWorker, numManager := 0, 2
	cl := newCluster(t, numWorker, numManager)
	defer func() {
		require.NoError(t, cl.Stop())
	}()
}

func TestServiceCreateLateBind(t *testing.T) {
	t.Parallel()

	numWorker, numManager := 3, 3

	cl := newTestCluster(t.Name(), false)
	for i := 0; i < numManager; i++ {
		require.NoError(t, cl.AddManager(true, nil), "manager number %d", i+1)
	}
	for i := 0; i < numWorker; i++ {
		require.NoError(t, cl.AddAgent(), "agent number %d", i+1)
	}

	defer func() {
		require.NoError(t, cl.Stop())
	}()

	sid, err := cl.CreateService("test_service", 60)
	require.NoError(t, err)
	pollServiceReady(t, cl, sid, 60)
}

func TestServiceCreate(t *testing.T) {
	t.Parallel()

	numWorker, numManager := 3, 3
	cl := newCluster(t, numWorker, numManager)
	defer func() {
		require.NoError(t, cl.Stop())
	}()

	sid, err := cl.CreateService("test_service", 60)
	require.NoError(t, err)
	pollServiceReady(t, cl, sid, 60)
}

func TestNodeOps(t *testing.T) {
	t.Parallel()

	numWorker, numManager := 1, 3
	cl := newCluster(t, numWorker, numManager)
	defer func() {
		require.NoError(t, cl.Stop())
	}()

	// demote leader
	leader, err := cl.Leader()
	require.NoError(t, err)
	require.NoError(t, cl.SetNodeRole(leader.node.NodeID(), api.NodeRoleWorker))
	// agents 2, managers 2
	numWorker++
	numManager--
	pollClusterReady(t, cl, numWorker, numManager)

	// remove node
	var worker *testNode
	for _, n := range cl.nodes {
		if !n.IsManager() && n.node.NodeID() != leader.node.NodeID() {
			worker = n
			break
		}
	}
	require.NoError(t, cl.RemoveNode(worker.node.NodeID(), false))
	// agents 1, managers 2
	numWorker--
	// long wait for heartbeat expiration
	pollClusterReady(t, cl, numWorker, numManager)

	// promote old leader back
	require.NoError(t, cl.SetNodeRole(leader.node.NodeID(), api.NodeRoleManager))
	numWorker--
	numManager++
	// agents 0, managers 3
	pollClusterReady(t, cl, numWorker, numManager)
}

func TestAutolockManagers(t *testing.T) {
	t.Parallel()

	// run this twice, once with FIPS set and once without FIPS set
	for _, fips := range []bool{true, false} {
		rootCA, err := ca.CreateRootCA("rootCN")
		require.NoError(t, err)
		numWorker, numManager := 1, 1
		cl := newClusterWithRootCA(t, numWorker, numManager, &rootCA, fips)
		defer func() {
			require.NoError(t, cl.Stop())
		}()

		// check that the cluster is not locked initially
		unlockKey, err := cl.GetUnlockKey()
		require.NoError(t, err)
		require.Equal(t, "SWMKEY-1-", unlockKey)

		// lock the cluster and make sure the unlock key is not empty
		require.NoError(t, cl.AutolockManagers(true))
		unlockKey, err = cl.GetUnlockKey()
		require.NoError(t, err)
		require.NotEqual(t, "SWMKEY-1-", unlockKey)

		// rotate unlock key
		require.NoError(t, cl.RotateUnlockKey())
		newUnlockKey, err := cl.GetUnlockKey()
		require.NoError(t, err)
		require.NotEqual(t, "SWMKEY-1-", newUnlockKey)
		require.NotEqual(t, unlockKey, newUnlockKey)

		// unlock the cluster
		require.NoError(t, cl.AutolockManagers(false))
		unlockKey, err = cl.GetUnlockKey()
		require.NoError(t, err)
		require.Equal(t, "SWMKEY-1-", unlockKey)
	}
}

func TestDemotePromote(t *testing.T) {
	t.Parallel()

	numWorker, numManager := 1, 3
	cl := newCluster(t, numWorker, numManager)
	defer func() {
		require.NoError(t, cl.Stop())
	}()

	leader, err := cl.Leader()
	require.NoError(t, err)
	var manager *testNode
	for _, n := range cl.nodes {
		if n.IsManager() && n.node.NodeID() != leader.node.NodeID() {
			manager = n
			break
		}
	}
	require.NoError(t, cl.SetNodeRole(manager.node.NodeID(), api.NodeRoleWorker))
	// agents 2, managers 2
	numWorker++
	numManager--
	pollClusterReady(t, cl, numWorker, numManager)

	// promote same node
	require.NoError(t, cl.SetNodeRole(manager.node.NodeID(), api.NodeRoleManager))
	// agents 1, managers 3
	numWorker--
	numManager++
	pollClusterReady(t, cl, numWorker, numManager)
}

func TestPromoteDemote(t *testing.T) {
	t.Parallel()

	numWorker, numManager := 1, 3
	cl := newCluster(t, numWorker, numManager)
	defer func() {
		require.NoError(t, cl.Stop())
	}()

	var worker *testNode
	for _, n := range cl.nodes {
		if !n.IsManager() {
			worker = n
			break
		}
	}
	require.NoError(t, cl.SetNodeRole(worker.node.NodeID(), api.NodeRoleManager))
	// agents 0, managers 4
	numWorker--
	numManager++
	pollClusterReady(t, cl, numWorker, numManager)

	// demote same node
	require.NoError(t, cl.SetNodeRole(worker.node.NodeID(), api.NodeRoleWorker))
	// agents 1, managers 3
	numWorker++
	numManager--
	pollClusterReady(t, cl, numWorker, numManager)
}

func TestDemotePromoteLeader(t *testing.T) {
	t.Parallel()

	numWorker, numManager := 1, 3
	cl := newCluster(t, numWorker, numManager)
	defer func() {
		require.NoError(t, cl.Stop())
	}()

	leader, err := cl.Leader()
	require.NoError(t, err)
	require.NoError(t, cl.SetNodeRole(leader.node.NodeID(), api.NodeRoleWorker))
	// agents 2, managers 2
	numWorker++
	numManager--
	pollClusterReady(t, cl, numWorker, numManager)

	//promote former leader back
	require.NoError(t, cl.SetNodeRole(leader.node.NodeID(), api.NodeRoleManager))
	// agents 1, managers 3
	numWorker--
	numManager++
	pollClusterReady(t, cl, numWorker, numManager)
}

func TestDemoteToSingleManager(t *testing.T) {
	t.Parallel()

	numWorker, numManager := 1, 3
	cl := newCluster(t, numWorker, numManager)
	defer func() {
		require.NoError(t, cl.Stop())
	}()

	leader, err := cl.Leader()
	require.NoError(t, err)
	require.NoError(t, cl.SetNodeRole(leader.node.NodeID(), api.NodeRoleWorker))
	// agents 2, managers 2
	numWorker++
	numManager--
	pollClusterReady(t, cl, numWorker, numManager)

	leader, err = cl.Leader()
	require.NoError(t, err)
	require.NoError(t, cl.SetNodeRole(leader.node.NodeID(), api.NodeRoleWorker))
	// agents 3, managers 1
	numWorker++
	numManager--
	pollClusterReady(t, cl, numWorker, numManager)
}

func TestDemoteLeader(t *testing.T) {
	t.Parallel()

	numWorker, numManager := 1, 3
	cl := newCluster(t, numWorker, numManager)
	defer func() {
		require.NoError(t, cl.Stop())
	}()

	leader, err := cl.Leader()
	require.NoError(t, err)
	require.NoError(t, cl.SetNodeRole(leader.node.NodeID(), api.NodeRoleWorker))
	// agents 2, managers 2
	numWorker++
	numManager--
	pollClusterReady(t, cl, numWorker, numManager)
}

func TestDemoteDownedManager(t *testing.T) {
	t.Parallel()

	numWorker, numManager := 0, 3
	cl := newCluster(t, numWorker, numManager)
	defer func() {
		require.NoError(t, cl.Stop())
	}()

	leader, err := cl.Leader()
	require.NoError(t, err)

	// Find a manager (not the leader) to demote. It must not be the third
	// manager we added, because there may not have been enough time for
	// that one to write anything to its WAL.
	var demotee *testNode
	for _, n := range cl.nodes {
		nodeID := n.node.NodeID()
		if n.IsManager() && nodeID != leader.node.NodeID() && cl.nodesOrder[nodeID] != 3 {
			demotee = n
			break
		}
	}

	nodeID := demotee.node.NodeID()

	resp, err := cl.api.GetNode(context.Background(), &api.GetNodeRequest{NodeID: nodeID})
	require.NoError(t, err)
	spec := resp.Node.Spec.Copy()
	spec.DesiredRole = api.NodeRoleWorker

	// stop the node, then demote it, and start it back up again so when it comes back up it has to realize
	// it's not running anymore
	require.NoError(t, demotee.Pause(false))

	// demote node, but don't use SetNodeRole, which waits until it successfully becomes a worker, since
	// the node is currently down
	require.NoError(t, testutils.PollFuncWithTimeout(nil, func() error {
		_, err := cl.api.UpdateNode(context.Background(), &api.UpdateNodeRequest{
			NodeID:      nodeID,
			Spec:        spec,
			NodeVersion: &resp.Node.Meta.Version,
		})
		return err
	}, opsTimeout))

	// start it back up again
	require.NoError(t, cl.StartNode(nodeID))

	// wait to become worker
	require.NoError(t, testutils.PollFuncWithTimeout(nil, func() error {
		if demotee.IsManager() {
			return fmt.Errorf("node is still not a worker")
		}
		return nil
	}, opsTimeout))

	// agents 1, managers 2
	numWorker++
	numManager--
	pollClusterReady(t, cl, numWorker, numManager)
}

func TestRestartLeader(t *testing.T) {
	t.Parallel()

	numWorker, numManager := 5, 3
	cl := newCluster(t, numWorker, numManager)
	defer func() {
		require.NoError(t, cl.Stop())
	}()
	leader, err := cl.Leader()
	require.NoError(t, err)

	origLeaderID := leader.node.NodeID()

	require.NoError(t, leader.Pause(false))

	require.NoError(t, testutils.PollFuncWithTimeout(nil, func() error {
		resp, err := cl.api.ListNodes(context.Background(), &api.ListNodesRequest{})
		if err != nil {
			return err
		}
		for _, node := range resp.Nodes {
			if node.ID == origLeaderID {
				continue
			}
			require.False(t, node.Status.State == api.NodeStatus_DOWN, "nodes shouldn't go to down")
			if node.Status.State != api.NodeStatus_READY {
				return errors.Errorf("node %s is still not ready", node.ID)
			}
		}
		return nil
	}, opsTimeout))

	require.NoError(t, cl.StartNode(origLeaderID))

	pollClusterReady(t, cl, numWorker, numManager)
}

func TestForceNewCluster(t *testing.T) {
	t.Parallel()

	// create an external CA so that we can use it to generate expired certificates
	rootCA, err := ca.CreateRootCA("externalRoot")
	require.NoError(t, err)

	// start a new cluster with the external CA bootstrapped
	numWorker, numManager := 0, 1
	cl := newTestCluster(t.Name(), false)
	defer func() {
		require.NoError(t, cl.Stop())
	}()
	require.NoError(t, cl.AddManager(false, &rootCA), "manager number 1")
	pollClusterReady(t, cl, numWorker, numManager)

	leader, err := cl.Leader()
	require.NoError(t, err)

	sid, err := cl.CreateService("test_service", 2)
	require.NoError(t, err)
	pollServiceReady(t, cl, sid, 2)

	// generate an expired certificate
	managerCertFile := filepath.Join(leader.stateDir, "certificates", "swarm-node.crt")
	certBytes, err := ioutil.ReadFile(managerCertFile)
	require.NoError(t, err)
	now := time.Now()
	// we don't want it too expired, because it can't have expired before the root CA cert is valid
	rootSigner, err := rootCA.Signer()
	require.NoError(t, err)
	expiredCertPEM := cautils.ReDateCert(t, certBytes, rootSigner.Cert, rootSigner.Key, now.Add(-1*time.Hour), now.Add(-1*time.Second))

	// restart node with an expired certificate while forcing a new cluster - it should start without error and the certificate should be renewed
	nodeID := leader.node.NodeID()
	require.NoError(t, leader.Pause(true))
	require.NoError(t, ioutil.WriteFile(managerCertFile, expiredCertPEM, 0644))
	require.NoError(t, cl.StartNode(nodeID))
	pollClusterReady(t, cl, numWorker, numManager)
	pollServiceReady(t, cl, sid, 2)

	err = testutils.PollFuncWithTimeout(nil, func() error {
		certBytes, err := ioutil.ReadFile(managerCertFile)
		if err != nil {
			return err
		}
		managerCerts, err := helpers.ParseCertificatesPEM(certBytes)
		if err != nil {
			return err
		}
		if managerCerts[0].NotAfter.Before(time.Now()) {
			return errors.New("certificate hasn't been renewed yet")
		}
		return nil
	}, opsTimeout)
	require.NoError(t, err)

	// restart node with an expired certificate without forcing a new cluster - it should error on start
	require.NoError(t, leader.Pause(true))
	require.NoError(t, ioutil.WriteFile(managerCertFile, expiredCertPEM, 0644))
	require.Error(t, cl.StartNode(nodeID))
}

func pollRootRotationDone(t *testing.T, cl *testCluster) {
	require.NoError(t, testutils.PollFuncWithTimeout(nil, func() error {
		clusterInfo, err := cl.GetClusterInfo()
		if err != nil {
			return err
		}
		if clusterInfo.RootCA.RootRotation != nil {
			return errors.New("root rotation not done")
		}
		return nil
	}, opsTimeout))
}

func TestSuccessfulRootRotation(t *testing.T) {
	t.Parallel()

	// run this twice, once with FIPS set and once without
	for _, fips := range []bool{true, false} {
		rootCA, err := ca.CreateRootCA("rootCN")
		require.NoError(t, err)

		numWorker, numManager := 2, 3
		cl := newClusterWithRootCA(t, numWorker, numManager, &rootCA, fips)
		defer func() {
			require.NoError(t, cl.Stop())
		}()
		pollClusterReady(t, cl, numWorker, numManager)

		// Take down one of managers and both workers, so we can't actually ever finish root rotation.
		resp, err := cl.api.ListNodes(context.Background(), &api.ListNodesRequest{})
		require.NoError(t, err)
		var (
			downManagerID string
			downWorkerIDs []string
			oldTLSInfo    *api.NodeTLSInfo
		)
		for _, n := range resp.Nodes {
			if oldTLSInfo != nil {
				require.Equal(t, oldTLSInfo, n.Description.TLSInfo)
			} else {
				oldTLSInfo = n.Description.TLSInfo
			}
			if n.Role == api.NodeRoleManager {
				if !n.ManagerStatus.Leader && downManagerID == "" {
					downManagerID = n.ID
					require.NoError(t, cl.nodes[n.ID].Pause(false))
				}
				continue
			}
			downWorkerIDs = append(downWorkerIDs, n.ID)
			require.NoError(t, cl.nodes[n.ID].Pause(false))
		}

		// perform a root rotation, and wait until all the nodes that are up have newly issued certs
		newRootCert, newRootKey, err := cautils.CreateRootCertAndKey("newRootCN")
		require.NoError(t, err)
		require.NoError(t, cl.RotateRootCA(newRootCert, newRootKey))

		require.NoError(t, testutils.PollFuncWithTimeout(nil, func() error {
			resp, err := cl.api.ListNodes(context.Background(), &api.ListNodesRequest{})
			if err != nil {
				return err
			}
			for _, n := range resp.Nodes {
				isDown := n.ID == downManagerID || n.ID == downWorkerIDs[0] || n.ID == downWorkerIDs[1]
				if reflect.DeepEqual(n.Description.TLSInfo, oldTLSInfo) != isDown {
					return fmt.Errorf("expected TLS info to have changed: %v", !isDown)
				}
			}

			// root rotation isn't done
			clusterInfo, err := cl.GetClusterInfo()
			if err != nil {
				return err
			}
			require.NotNil(t, clusterInfo.RootCA.RootRotation) // if root rotation is already done, fail and finish the test here
			return nil
		}, opsTimeout))

		// Bring the other manager back.  Also bring one worker back, kill the other worker,
		// and add a new worker - show that we can converge on a root rotation.
		require.NoError(t, cl.StartNode(downManagerID))
		require.NoError(t, cl.StartNode(downWorkerIDs[0]))
		require.NoError(t, cl.RemoveNode(downWorkerIDs[1], false))
		require.NoError(t, cl.AddAgent())

		// we can finish root rotation even though the previous leader was down because it had
		// already rotated its cert
		pollRootRotationDone(t, cl)

		// wait until all the nodes have gotten their new certs and trust roots
		require.NoError(t, testutils.PollFuncWithTimeout(nil, func() error {
			resp, err = cl.api.ListNodes(context.Background(), &api.ListNodesRequest{})
			if err != nil {
				return err
			}
			var newTLSInfo *api.NodeTLSInfo
			for _, n := range resp.Nodes {
				if newTLSInfo == nil {
					newTLSInfo = n.Description.TLSInfo
					if bytes.Equal(newTLSInfo.CertIssuerPublicKey, oldTLSInfo.CertIssuerPublicKey) ||
						bytes.Equal(newTLSInfo.CertIssuerSubject, oldTLSInfo.CertIssuerSubject) {
						return errors.New("expecting the issuer to have changed")
					}
					if !bytes.Equal(newTLSInfo.TrustRoot, newRootCert) {
						return errors.New("expecting the the root certificate to have changed")
					}
				} else if !reflect.DeepEqual(newTLSInfo, n.Description.TLSInfo) {
					return fmt.Errorf("the nodes have not converged yet, particularly %s", n.ID)
				}

				if n.Certificate.Status.State != api.IssuanceStateIssued {
					return errors.New("nodes have yet to finish renewing their TLS certificates")
				}
			}
			return nil
		}, opsTimeout))
	}
}

func TestRepeatedRootRotation(t *testing.T) {
	t.Parallel()
	numWorker, numManager := 3, 1
	cl := newCluster(t, numWorker, numManager)
	defer func() {
		require.NoError(t, cl.Stop())
	}()
	pollClusterReady(t, cl, numWorker, numManager)

	resp, err := cl.api.ListNodes(context.Background(), &api.ListNodesRequest{})
	require.NoError(t, err)
	var oldTLSInfo *api.NodeTLSInfo
	for _, n := range resp.Nodes {
		if oldTLSInfo != nil {
			require.Equal(t, oldTLSInfo, n.Description.TLSInfo)
		} else {
			oldTLSInfo = n.Description.TLSInfo
		}
	}

	// perform multiple root rotations, wait a second between each
	var newRootCert, newRootKey []byte
	for i := 0; i < 3; i++ {
		newRootCert, newRootKey, err = cautils.CreateRootCertAndKey("newRootCN")
		require.NoError(t, err)
		require.NoError(t, cl.RotateRootCA(newRootCert, newRootKey))
		time.Sleep(time.Second)
	}

	pollRootRotationDone(t, cl)

	// wait until all the nodes are stabilized back to the latest issuer
	require.NoError(t, testutils.PollFuncWithTimeout(nil, func() error {
		resp, err = cl.api.ListNodes(context.Background(), &api.ListNodesRequest{})
		if err != nil {
			return nil
		}
		for _, n := range resp.Nodes {
			if reflect.DeepEqual(n.Description.TLSInfo, oldTLSInfo) {
				return errors.New("nodes have not changed TLS info")
			}
			if n.Certificate.Status.State != api.IssuanceStateIssued {
				return errors.New("nodes have yet to finish renewing their TLS certificates")
			}
			if !bytes.Equal(n.Description.TLSInfo.TrustRoot, newRootCert) {
				return errors.New("nodes do not all trust the new root yet")
			}
		}
		return nil
	}, opsTimeout))
}

func TestNodeRejoins(t *testing.T) {
	t.Parallel()
	numWorker, numManager := 1, 1
	cl := newCluster(t, numWorker, numManager)
	defer func() {
		require.NoError(t, cl.Stop())
	}()
	pollClusterReady(t, cl, numWorker, numManager)

	clusterInfo, err := cl.GetClusterInfo()
	require.NoError(t, err)

	// find the worker
	var worker *testNode
	for _, n := range cl.nodes {
		if !n.IsManager() {
			worker = n
		}
	}

	// rejoining succeeds - (both because the certs are correct, and because node.Pause sets the JoinAddr to "")
	nodeID := worker.node.NodeID()
	require.NoError(t, worker.Pause(false))
	require.NoError(t, cl.StartNode(nodeID))
	pollClusterReady(t, cl, numWorker, numManager)

	// rejoining if the certs are wrong will fail fast so long as the join address is passed, but will keep retrying
	// forever if the join address is not passed
	leader, err := cl.Leader()
	require.NoError(t, err)
	require.NoError(t, worker.Pause(false))

	// generate new certs with the same node ID, role, and cluster ID, but with the wrong CA
	paths := ca.NewConfigPaths(filepath.Join(worker.config.StateDir, "certificates"))
	newRootCA, err := ca.CreateRootCA("bad root CA")
	require.NoError(t, err)
	ca.SaveRootCA(newRootCA, paths.RootCA)
	krw := ca.NewKeyReadWriter(paths.Node, nil, &manager.RaftDEKData{}) // make sure the key headers are preserved
	_, _, err = krw.Read()
	require.NoError(t, err)
	_, _, err = newRootCA.IssueAndSaveNewCertificates(krw, nodeID, ca.WorkerRole, clusterInfo.ID)
	require.NoError(t, err)

	worker.config.JoinAddr, err = leader.node.RemoteAPIAddr()
	require.NoError(t, err)
	err = cl.StartNode(nodeID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "certificate signed by unknown authority")
}

func TestNodeJoinWithWrongCerts(t *testing.T) {
	t.Parallel()
	numWorker, numManager := 1, 1
	cl := newCluster(t, numWorker, numManager)
	defer func() {
		require.NoError(t, cl.Stop())
	}()
	pollClusterReady(t, cl, numWorker, numManager)

	clusterInfo, err := cl.GetClusterInfo()
	require.NoError(t, err)

	joinAddr, err := cl.RandomManager().node.RemoteAPIAddr()
	require.NoError(t, err)

	tokens := map[string]string{
		ca.WorkerRole:  clusterInfo.RootCA.JoinTokens.Worker,
		ca.ManagerRole: clusterInfo.RootCA.JoinTokens.Manager,
	}

	rootCA, err := ca.CreateRootCA("rootCA")
	require.NoError(t, err)

	for role, token := range tokens {
		node, err := newTestNode(joinAddr, token, false, false)
		require.NoError(t, err)
		nodeID := identity.NewID()
		require.NoError(t,
			generateCerts(node.stateDir, &rootCA, nodeID, role, clusterInfo.ID, false))
		cl.counter++
		cl.nodes[nodeID] = node
		cl.nodesOrder[nodeID] = cl.counter

		err = cl.StartNode(nodeID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "certificate signed by unknown authority")
	}
}

// If the cluster does not require FIPS, then any node can join and re-join
// regardless of FIPS mode.
func TestMixedFIPSClusterNonMandatoryFIPS(t *testing.T) {
	t.Parallel()

	cl := newTestCluster(t.Name(), false) // no fips
	defer func() {
		require.NoError(t, cl.Stop())
	}()
	// create cluster with a non-FIPS manager, add another non-FIPS manager and a non-FIPs worker
	for i := 0; i < 2; i++ {
		require.NoError(t, cl.AddManager(false, nil))
	}
	require.NoError(t, cl.AddAgent())

	// add a FIPS manager and FIPS worker
	joinAddr, err := cl.RandomManager().node.RemoteAPIAddr()
	require.NoError(t, err)
	clusterInfo, err := cl.GetClusterInfo()
	require.NoError(t, err)
	for _, token := range []string{clusterInfo.RootCA.JoinTokens.Worker, clusterInfo.RootCA.JoinTokens.Manager} {
		node, err := newTestNode(joinAddr, token, false, true)
		require.NoError(t, err)
		require.NoError(t, cl.AddNode(node))
	}

	pollClusterReady(t, cl, 2, 3)

	// switch which worker nodes are fips and which are not - all should start up just fine
	// on managers, if we enable fips on a previously non-fips node, it won't be able to read
	// non-fernet raft logs
	for nodeID, n := range cl.nodes {
		if n.IsManager() {
			n.config.FIPS = false
		} else {
			n.config.FIPS = !n.config.FIPS
		}
		require.NoError(t, n.Pause(false))
		require.NoError(t, cl.StartNode(nodeID))
	}

	pollClusterReady(t, cl, 2, 3)
}

// If the cluster require FIPS, then only FIPS nodes can join and re-join.
func TestMixedFIPSClusterMandatoryFIPS(t *testing.T) {
	t.Parallel()

	cl := newTestCluster(t.Name(), true)
	defer func() {
		require.NoError(t, cl.Stop())
	}()
	for i := 0; i < 3; i++ {
		require.NoError(t, cl.AddManager(false, nil))
	}
	require.NoError(t, cl.AddAgent())

	pollClusterReady(t, cl, 1, 3)

	// restart a manager and restart the worker in non-FIPS mode - both will fail, but restarting it in FIPS mode
	// will succeed
	leader, err := cl.Leader()
	require.NoError(t, err)
	var nonLeader, worker *testNode
	for _, n := range cl.nodes {
		if n == leader {
			continue
		}
		if nonLeader == nil && n.IsManager() {
			nonLeader = n
		}
		if worker == nil && !n.IsManager() {
			worker = n
		}
	}
	for _, n := range []*testNode{nonLeader, worker} {
		nodeID := n.node.NodeID()
		rAddr := ""
		if n.IsManager() {
			// make sure to save the old address because if a node is stopped, we can't get the node address, and it gets set to
			// a completely new address, which will break raft in the case of a manager
			rAddr, err = n.node.RemoteAPIAddr()
			require.NoError(t, err)
		}
		require.NoError(t, n.Pause(false))
		n.config.FIPS = false
		require.Equal(t, node.ErrMandatoryFIPS, cl.StartNode(nodeID))

		require.NoError(t, n.Pause(false))
		n.config.FIPS = true
		n.config.ListenRemoteAPI = rAddr
		require.NoError(t, cl.StartNode(nodeID))
	}

	pollClusterReady(t, cl, 1, 3)

	// try to add a non-FIPS manager and non-FIPS worker - it won't work
	joinAddr, err := cl.RandomManager().node.RemoteAPIAddr()
	require.NoError(t, err)
	clusterInfo, err := cl.GetClusterInfo()
	require.NoError(t, err)
	for _, token := range []string{clusterInfo.RootCA.JoinTokens.Worker, clusterInfo.RootCA.JoinTokens.Manager} {
		n, err := newTestNode(joinAddr, token, false, false)
		require.NoError(t, err)
		require.Equal(t, node.ErrMandatoryFIPS, cl.AddNode(n))
	}
}
