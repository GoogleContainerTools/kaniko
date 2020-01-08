package logbroker

import (
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/ca"
	"github.com/docker/swarmkit/ca/testutils"
	"github.com/docker/swarmkit/manager/state/store"
	"github.com/docker/swarmkit/protobuf/ptypes"
	gogotypes "github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/require"
)

func TestLogBrokerLogs(t *testing.T) {
	ctx, ca, broker, serverAddr, brokerAddr, done := testLogBrokerEnv(t)
	defer done()

	client, clientDone := testLogClient(t, serverAddr)
	defer clientDone()
	brokerClient, agentSecurity, brokerClientDone := testBrokerClient(t, ca, brokerAddr)
	defer brokerClientDone()

	var (
		wg               sync.WaitGroup
		hold             = make(chan struct{}) // coordinates pubsub start
		messagesExpected int
	)

	subStream, err := brokerClient.ListenSubscriptions(ctx, &api.ListenSubscriptionsRequest{})
	if err != nil {
		t.Fatal(err)
	}

	stream, err := client.SubscribeLogs(ctx, &api.SubscribeLogsRequest{
		Options: &api.LogSubscriptionOptions{
			Follow: true,
		},
		Selector: &api.LogSelector{
			NodeIDs: []string{agentSecurity.ServerTLSCreds.NodeID()},
		},
	})
	if err != nil {
		t.Fatalf("error subscribing: %v", err)
	}

	subscription, err := subStream.Recv()
	if err != nil {
		t.Fatal(err)
	}

	// spread some services across nodes with a bunch of tasks.
	const (
		nNodes              = 5
		nServices           = 20
		nTasksPerService    = 20
		nLogMessagesPerTask = 5
	)

	for service := 0; service < nServices; service++ {
		serviceID := fmt.Sprintf("service-%v", service)

		for task := 0; task < nTasksPerService; task++ {
			taskID := fmt.Sprintf("%v.task-%v", serviceID, task)

			for node := 0; node < nNodes; node++ {
				nodeID := fmt.Sprintf("node-%v", node)

				if (task+1)%(node+1) != 0 {
					continue
				}
				messagesExpected += nLogMessagesPerTask

				wg.Add(1)
				go func(nodeID, serviceID, taskID string) {
					<-hold

					// Each goroutine gets its own publisher
					publisher, err := brokerClient.PublishLogs(ctx)
					require.NoError(t, err)

					defer func() {
						_, err := publisher.CloseAndRecv()
						require.NoError(t, err)
						wg.Done()
					}()

					msgctx := api.LogContext{
						NodeID:    agentSecurity.ClientTLSCreds.NodeID(),
						ServiceID: serviceID,
						TaskID:    taskID,
					}
					for i := 0; i < nLogMessagesPerTask; i++ {
						require.NoError(t, publisher.Send(&api.PublishLogsMessage{
							SubscriptionID: subscription.ID,
							Messages:       []api.LogMessage{newLogMessage(msgctx, "log message number %d", i)},
						}))
					}
				}(nodeID, serviceID, taskID)
			}
		}
	}

	t.Logf("expected %v messages", messagesExpected)
	close(hold)
	var messages int
	for messages < messagesExpected {
		msgs, err := stream.Recv()
		require.NoError(t, err)
		for range msgs.Messages {
			messages++
			if messages%100 == 0 {
				fmt.Println(messages, "received")
			}
		}
	}
	t.Logf("received %v messages", messages)

	wg.Wait()

	// Make sure double Start throws an error
	require.EqualError(t, broker.Start(ctx), errAlreadyRunning.Error())
	// Stop should work
	require.NoError(t, broker.Stop())
	// Double stopping should fail
	require.EqualError(t, broker.Stop(), errNotRunning.Error())
}

func listenSubscriptions(ctx context.Context, t *testing.T, client api.LogBrokerClient) <-chan *api.SubscriptionMessage {
	subscriptions, err := client.ListenSubscriptions(ctx, &api.ListenSubscriptionsRequest{})
	require.NoError(t, err)

	ch := make(chan *api.SubscriptionMessage)
	go func() {
		defer close(ch)

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			sub, err := subscriptions.Recv()
			if err != nil {
				return
			}
			ch <- sub
		}
	}()

	return ch
}

func ensureSubscription(t *testing.T, subscriptions <-chan *api.SubscriptionMessage) *api.SubscriptionMessage {
	select {
	case s := <-subscriptions:
		require.NotNil(t, s)
		return s
	case <-time.After(5 * time.Second):
		require.FailNow(t, "subscription expected")
	}
	return nil
}

func ensureNoSubscription(t *testing.T, subscriptions <-chan *api.SubscriptionMessage) {
	select {
	case s := <-subscriptions:
		require.FailNow(t, fmt.Sprintf("unexpected subscription: %v", s))
	case <-time.After(10 * time.Millisecond):
		return
	}
}

func TestLogBrokerSubscriptions(t *testing.T) {
	ctx, ca, _, serverAddr, brokerAddr, done := testLogBrokerEnv(t)
	defer done()

	client, clientDone := testLogClient(t, serverAddr)
	defer clientDone()

	agent1, agent1Security, agent1Done := testBrokerClient(t, ca, brokerAddr)
	defer agent1Done()

	agent2, agent2Security, agent2Done := testBrokerClient(t, ca, brokerAddr)
	defer agent2Done()

	// Have an agent listen to subscriptions before anyone has subscribed.
	subscriptions1 := listenSubscriptions(ctx, t, agent1)

	// Send two subscriptions - one will match both agent1 and agent2 while
	// the other only agent1
	_, err := client.SubscribeLogs(ctx, &api.SubscribeLogsRequest{
		Options: &api.LogSubscriptionOptions{
			Follow: true,
		},
		Selector: &api.LogSelector{
			NodeIDs: []string{
				agent1Security.ServerTLSCreds.NodeID(),
			},
		},
	})
	require.NoError(t, err)
	_, err = client.SubscribeLogs(ctx, &api.SubscribeLogsRequest{
		Options: &api.LogSubscriptionOptions{
			Follow: true,
		},
		Selector: &api.LogSelector{
			NodeIDs: []string{
				agent1Security.ServerTLSCreds.NodeID(),
				agent2Security.ServerTLSCreds.NodeID(),
			},
		},
	})
	require.NoError(t, err)

	// Make sure we received two subscriptions on agent 1 (already joined).
	{
		s1 := ensureSubscription(t, subscriptions1)
		require.False(t, s1.Close)
		require.Contains(t, s1.Selector.NodeIDs, agent1Security.ServerTLSCreds.NodeID())

		s2 := ensureSubscription(t, subscriptions1)
		require.False(t, s2.Close)
		require.Contains(t, s2.Selector.NodeIDs, agent1Security.ServerTLSCreds.NodeID())

		// Ensure we received two different subscriptions.
		require.NotEqual(t, s1.ID, s2.ID)
	}

	// Join a second agent.
	subscriptions2 := listenSubscriptions(ctx, t, agent2)

	// Make sure we receive past subscriptions.
	// Make sure we receive *only* the right one.
	{
		s := ensureSubscription(t, subscriptions2)
		require.False(t, s.Close)
		require.Equal(t, []string{agent1Security.ServerTLSCreds.NodeID(), agent2Security.ServerTLSCreds.NodeID()}, s.Selector.NodeIDs)

		ensureNoSubscription(t, subscriptions2)
	}
}

func TestLogBrokerSelector(t *testing.T) {
	ctx, ca, _, serverAddr, brokerAddr, done := testLogBrokerEnv(t)
	defer done()

	client, clientDone := testLogClient(t, serverAddr)
	defer clientDone()

	agent1, agent1Security, agent1Done := testBrokerClient(t, ca, brokerAddr)
	defer agent1Done()
	agent1subscriptions := listenSubscriptions(ctx, t, agent1)

	agent2, agent2Security, agent2Done := testBrokerClient(t, ca, brokerAddr)
	defer agent2Done()

	agent2subscriptions := listenSubscriptions(ctx, t, agent2)

	// Subscribe to a task.
	require.NoError(t, ca.MemoryStore.Update(func(tx store.Tx) error {
		return store.CreateTask(tx, &api.Task{
			ID: "task",
		})
	}))
	_, err := client.SubscribeLogs(ctx, &api.SubscribeLogsRequest{
		Options: &api.LogSubscriptionOptions{
			Follow: true,
		},
		Selector: &api.LogSelector{
			TaskIDs: []string{"task"},
		},
	})
	require.NoError(t, err)

	// Since it's not assigned to any agent, nobody should receive it.
	ensureNoSubscription(t, agent1subscriptions)
	ensureNoSubscription(t, agent2subscriptions)

	// Assign the task to agent-1. Make sure it's received by agent-1 but *not*
	// agent-2.
	require.NoError(t, ca.MemoryStore.Update(func(tx store.Tx) error {
		task := store.GetTask(tx, "task")
		require.NotNil(t, task)
		task.NodeID = agent1Security.ServerTLSCreds.NodeID()
		return store.UpdateTask(tx, task)
	}))

	ensureSubscription(t, agent1subscriptions)
	ensureNoSubscription(t, agent2subscriptions)

	// Subscribe to a service.
	require.NoError(t, ca.MemoryStore.Update(func(tx store.Tx) error {
		return store.CreateService(tx, &api.Service{
			ID: "service",
		})
	}))
	_, err = client.SubscribeLogs(ctx, &api.SubscribeLogsRequest{
		Options: &api.LogSubscriptionOptions{
			Follow: true,
		},
		Selector: &api.LogSelector{
			ServiceIDs: []string{"service"},
		},
	})
	require.NoError(t, err)

	// Since there are no corresponding tasks, nobody should receive it.
	ensureNoSubscription(t, agent1subscriptions)
	ensureNoSubscription(t, agent2subscriptions)

	// Create a task that does *NOT* belong to our service and assign it to node-1.
	require.NoError(t, ca.MemoryStore.Update(func(tx store.Tx) error {
		return store.CreateTask(tx, &api.Task{
			ID:        "wrong-task",
			ServiceID: "wrong-service",
			NodeID:    agent1Security.ServerTLSCreds.NodeID(),
		})
	}))

	// Ensure agent-1 doesn't receive it.
	ensureNoSubscription(t, agent1subscriptions)

	// Now create another task that does belong to our service and assign it to node-1.
	require.NoError(t, ca.MemoryStore.Update(func(tx store.Tx) error {
		return store.CreateTask(tx, &api.Task{
			ID:        "service-task-1",
			ServiceID: "service",
			NodeID:    agent1Security.ServerTLSCreds.NodeID(),
		})
	}))

	// Make sure agent-1 receives it...
	ensureSubscription(t, agent1subscriptions)
	// ...and agent-2 does not.
	ensureNoSubscription(t, agent2subscriptions)

	// Create another task, same as above.
	require.NoError(t, ca.MemoryStore.Update(func(tx store.Tx) error {
		return store.CreateTask(tx, &api.Task{
			ID:        "service-task-2",
			ServiceID: "service",
			NodeID:    agent1Security.ServerTLSCreds.NodeID(),
		})
	}))

	// agent-1 should *not* receive it anymore since the subscription was already delivered.
	// agent-2 should still not get it.
	ensureNoSubscription(t, agent1subscriptions)
	ensureNoSubscription(t, agent2subscriptions)

	// Now, create another one and assign it to agent-2.
	require.NoError(t, ca.MemoryStore.Update(func(tx store.Tx) error {
		return store.CreateTask(tx, &api.Task{
			ID:        "service-task-3",
			ServiceID: "service",
			NodeID:    agent2Security.ServerTLSCreds.NodeID(),
		})
	}))

	// Make sure it's delivered to agent-2.
	ensureSubscription(t, agent2subscriptions)
	// it shouldn't do anything for agent-1.
	ensureNoSubscription(t, agent1subscriptions)
}

func TestLogBrokerNoFollow(t *testing.T) {
	t.Parallel()

	ctx, ca, _, serverAddr, brokerAddr, done := testLogBrokerEnv(t)
	defer done()

	client, clientDone := testLogClient(t, serverAddr)
	defer clientDone()

	agent1, agent1Security, agent1Done := testBrokerClient(t, ca, brokerAddr)
	defer agent1Done()
	agent1subscriptions := listenSubscriptions(ctx, t, agent1)

	agent2, agent2Security, agent2Done := testBrokerClient(t, ca, brokerAddr)
	defer agent2Done()
	agent2subscriptions := listenSubscriptions(ctx, t, agent2)

	// Create fake environment.
	require.NoError(t, ca.MemoryStore.Update(func(tx store.Tx) error {
		if err := store.CreateTask(tx, &api.Task{
			ID:        "task1",
			ServiceID: "service",
			Status: api.TaskStatus{
				State: api.TaskStateRunning,
			},
			NodeID: agent1Security.ServerTLSCreds.NodeID(),
		}); err != nil {
			return err
		}

		return store.CreateTask(tx, &api.Task{
			ID:        "task2",
			ServiceID: "service",
			Status: api.TaskStatus{
				State: api.TaskStateRunning,
			},
			NodeID: agent2Security.ServerTLSCreds.NodeID(),
		})
	}))

	// We need to sleep here to give ListenSubscriptions time to call
	// registerSubscription before SubscribeLogs concludes that one or both
	// of the agents are not connected, and prematurely calls Done for one
	// or both nodes. Think of these stream RPC calls as goroutines which
	// don't have synchronization around anything that happens in the RPC
	// handler before a send or receive. It would be nice if we had a way
	// of confirming that a node was listening for subscriptions before
	// calling SubscribeLogs, but the current API doesn't provide this.
	time.Sleep(time.Second)

	// Subscribe to logs in no follow mode
	logs, err := client.SubscribeLogs(ctx, &api.SubscribeLogsRequest{
		Options: &api.LogSubscriptionOptions{
			Follow: false,
		},
		Selector: &api.LogSelector{
			ServiceIDs: []string{"service"},
		},
	})
	require.NoError(t, err)

	// Get the subscriptions from the agents.
	subscription1 := ensureSubscription(t, agent1subscriptions)
	require.Equal(t, subscription1.Selector.ServiceIDs[0], "service")
	subscription2 := ensureSubscription(t, agent2subscriptions)
	require.Equal(t, subscription2.Selector.ServiceIDs[0], "service")

	require.Equal(t, subscription1.ID, subscription2.ID)

	// Publish a log message from agent-1 and close the publisher
	publisher, err := agent1.PublishLogs(ctx)
	require.NoError(t, err)
	require.NoError(t,
		publisher.Send(&api.PublishLogsMessage{
			SubscriptionID: subscription1.ID,
			Messages: []api.LogMessage{
				newLogMessage(api.LogContext{
					NodeID:    agent1Security.ServerTLSCreds.NodeID(),
					ServiceID: "service",
					TaskID:    "task1",
				}, "log message"),
			},
		}))
	_, err = publisher.CloseAndRecv()
	require.NoError(t, err)

	// Ensure we get it from the other end
	log, err := logs.Recv()
	require.NoError(t, err)
	require.Len(t, log.Messages, 1)
	require.Equal(t, log.Messages[0].Context.NodeID, agent1Security.ServerTLSCreds.NodeID())

	// Now publish a message from the other agent and close the subscription
	publisher, err = agent2.PublishLogs(ctx)
	require.NoError(t, err)
	require.NoError(t,
		publisher.Send(&api.PublishLogsMessage{
			SubscriptionID: subscription2.ID,
			Messages: []api.LogMessage{
				newLogMessage(api.LogContext{
					NodeID:    agent2Security.ServerTLSCreds.NodeID(),
					ServiceID: "service",
					TaskID:    "task2",
				}, "log message"),
			},
		}))
	_, err = publisher.CloseAndRecv()
	require.NoError(t, err)

	// Ensure we get it from the other end
	log, err = logs.Recv()
	require.NoError(t, err)
	require.Len(t, log.Messages, 1)
	require.Equal(t, log.Messages[0].Context.NodeID, agent2Security.ServerTLSCreds.NodeID())

	// Since we receive both messages the log stream should end
	_, err = logs.Recv()
	require.Equal(t, err, io.EOF)
}

func TestLogBrokerNoFollowMissingNode(t *testing.T) {
	t.Parallel()

	ctx, ca, _, serverAddr, brokerAddr, done := testLogBrokerEnv(t)
	defer done()

	client, clientDone := testLogClient(t, serverAddr)
	defer clientDone()

	agent, agentSecurity, agentDone := testBrokerClient(t, ca, brokerAddr)
	defer agentDone()
	agentSubscriptions := listenSubscriptions(ctx, t, agent)

	// Create fake environment.
	// A service with one instance on a genuine node and another instance
	// and a node that didn't connect to the broker.
	require.NoError(t, ca.MemoryStore.Update(func(tx store.Tx) error {
		if err := store.CreateTask(tx, &api.Task{
			ID:        "task1",
			ServiceID: "service",
			Status: api.TaskStatus{
				State: api.TaskStateRunning,
			},
			NodeID: agentSecurity.ServerTLSCreds.NodeID(),
		}); err != nil {
			return err
		}

		return store.CreateTask(tx, &api.Task{
			ID:        "task2",
			ServiceID: "service",
			NodeID:    "node-2",
			Status: api.TaskStatus{
				State: api.TaskStateRunning,
			},
		})
	}))

	// We need to sleep here to give ListenSubscriptions time to call
	// registerSubscription before SubscribeLogs concludes that the actual
	// agent is not connected, and prematurely calls Done for it. Think of
	// these stream RPC calls as goroutines which don't have synchronization
	// around anything that happens in the RPC handler before a send or
	// receive. It would be nice if we had a way of confirming that a node
	// was listening for subscriptions before calling SubscribeLogs, but
	// the current API doesn't provide this.
	time.Sleep(time.Second)

	// Subscribe to logs in no follow mode
	logs, err := client.SubscribeLogs(ctx, &api.SubscribeLogsRequest{
		Options: &api.LogSubscriptionOptions{
			Follow: false,
		},
		Selector: &api.LogSelector{
			ServiceIDs: []string{"service"},
		},
	})
	require.NoError(t, err)

	// Grab the subscription and publish a log message from the connected agent.
	subscription := ensureSubscription(t, agentSubscriptions)
	require.Equal(t, subscription.Selector.ServiceIDs[0], "service")
	publisher, err := agent.PublishLogs(ctx)
	require.NoError(t, err)
	require.NoError(t,
		publisher.Send(&api.PublishLogsMessage{
			SubscriptionID: subscription.ID,
			Messages: []api.LogMessage{
				newLogMessage(api.LogContext{
					NodeID:    agentSecurity.ServerTLSCreds.NodeID(),
					ServiceID: "service",
					TaskID:    "task1",
				}, "log message"),
			},
		}))
	_, err = publisher.CloseAndRecv()
	require.NoError(t, err)

	// Ensure we receive the message that we could grab
	log, err := logs.Recv()
	require.NoError(t, err)
	require.Len(t, log.Messages, 1)
	require.Equal(t, log.Messages[0].Context.NodeID, agentSecurity.ServerTLSCreds.NodeID())

	// Ensure the log stream ends with an error complaining about the missing node
	_, err = logs.Recv()
	require.Error(t, err)
	require.Contains(t, err.Error(), "node-2 is not available")
}

func TestLogBrokerNoFollowNotYetRunningTask(t *testing.T) {
	ctx, ca, _, serverAddr, _, done := testLogBrokerEnv(t)
	defer done()

	client, clientDone := testLogClient(t, serverAddr)
	defer clientDone()

	// Create fake environment.
	require.NoError(t, ca.MemoryStore.Update(func(tx store.Tx) error {
		return store.CreateTask(tx, &api.Task{
			ID:        "task1",
			ServiceID: "service",
			Status: api.TaskStatus{
				State: api.TaskStateNew,
			},
		})
	}))

	// Subscribe to logs in no follow mode
	logs, err := client.SubscribeLogs(ctx, &api.SubscribeLogsRequest{
		Options: &api.LogSubscriptionOptions{
			Follow: false,
		},
		Selector: &api.LogSelector{
			ServiceIDs: []string{"service"},
		},
	})
	require.NoError(t, err)

	// The log stream should be empty, because the task was not yet running
	_, err = logs.Recv()
	require.Error(t, err)
	require.Equal(t, err, io.EOF)
}

func TestLogBrokerNoFollowDisconnect(t *testing.T) {
	t.Parallel()

	ctx, ca, _, serverAddr, brokerAddr, done := testLogBrokerEnv(t)
	defer done()

	client, clientDone := testLogClient(t, serverAddr)
	defer clientDone()

	agent1, agent1Security, agent1Done := testBrokerClient(t, ca, brokerAddr)
	defer agent1Done()
	agent1subscriptions := listenSubscriptions(ctx, t, agent1)

	agent2, agent2Security, agent2Done := testBrokerClient(t, ca, brokerAddr)
	defer agent2Done()
	agent2subscriptions := listenSubscriptions(ctx, t, agent2)

	// Create fake environment.
	require.NoError(t, ca.MemoryStore.Update(func(tx store.Tx) error {
		if err := store.CreateTask(tx, &api.Task{
			ID:        "task1",
			ServiceID: "service",
			Status: api.TaskStatus{
				State: api.TaskStateRunning,
			},
			NodeID: agent1Security.ServerTLSCreds.NodeID(),
		}); err != nil {
			return err
		}

		return store.CreateTask(tx, &api.Task{
			ID:        "task2",
			ServiceID: "service",
			Status: api.TaskStatus{
				State: api.TaskStateRunning,
			},
			NodeID: agent2Security.ServerTLSCreds.NodeID(),
		})
	}))

	// We need to sleep here to give ListenSubscriptions time to call
	// registerSubscription before SubscribeLogs concludes that one or both
	// of the agents are not connected, and prematurely calls Done for one
	// or both nodes. Think of these stream RPC calls as goroutines which
	// don't have synchronization around anything that happens in the RPC
	// handler before a send or receive. It would be nice if we had a way
	// of confirming that a node was listening for subscriptions before
	// calling SubscribeLogs, but the current API doesn't provide this.
	time.Sleep(time.Second)

	// Subscribe to logs in no follow mode
	logs, err := client.SubscribeLogs(ctx, &api.SubscribeLogsRequest{
		Options: &api.LogSubscriptionOptions{
			Follow: false,
		},
		Selector: &api.LogSelector{
			ServiceIDs: []string{"service"},
		},
	})
	require.NoError(t, err)

	// Get the subscriptions from the agents.
	subscription1 := ensureSubscription(t, agent1subscriptions)
	require.Equal(t, subscription1.Selector.ServiceIDs[0], "service")
	subscription2 := ensureSubscription(t, agent2subscriptions)
	require.Equal(t, subscription2.Selector.ServiceIDs[0], "service")

	require.Equal(t, subscription1.ID, subscription2.ID)

	// Publish a log message from agent-1 and close the publisher
	publisher, err := agent1.PublishLogs(ctx)
	require.NoError(t, err)
	require.NoError(t,
		publisher.Send(&api.PublishLogsMessage{
			SubscriptionID: subscription1.ID,
			Messages: []api.LogMessage{
				newLogMessage(api.LogContext{
					NodeID:    agent1Security.ServerTLSCreds.NodeID(),
					ServiceID: "service",
					TaskID:    "task1",
				}, "log message"),
			},
		}))
	_, err = publisher.CloseAndRecv()
	require.NoError(t, err)

	// Now suddenly disconnect agent2...
	agent2Done()

	// Ensure we get the first message
	log, err := logs.Recv()
	require.NoError(t, err)
	require.Len(t, log.Messages, 1)
	require.Equal(t, log.Messages[0].Context.NodeID, agent1Security.ServerTLSCreds.NodeID())

	// ...and then an error
	_, err = logs.Recv()
	require.Error(t, err)
	require.Contains(t, err.Error(), "disconnected unexpectedly")
}

func testLogBrokerEnv(t *testing.T) (context.Context, *testutils.TestCA, *LogBroker, string, string, func()) {
	ctx, cancel := context.WithCancel(context.Background())

	tca := testutils.NewTestCA(nil)
	broker := New(tca.MemoryStore)

	// Log Server
	logListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("error setting up listener: %v", err)
	}
	logServer := grpc.NewServer()
	api.RegisterLogsServer(logServer, broker)

	go func() {
		if err := logServer.Serve(logListener); err != nil {
			// SIGH(stevvooe): GRPC won't really shutdown gracefully.
			// This should be fatal.
			t.Logf("error serving grpc service: %v", err)
		}
	}()

	// Log Broker
	brokerListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("error setting up listener: %v", err)
	}

	securityConfig, err := tca.NewNodeConfig(ca.ManagerRole)
	if err != nil {
		t.Fatal(err)
	}
	serverOpts := []grpc.ServerOption{grpc.Creds(securityConfig.ServerTLSCreds)}
	brokerServer := grpc.NewServer(serverOpts...)

	authorize := func(ctx context.Context, roles []string) error {
		_, err := ca.AuthorizeForwardedRoleAndOrg(ctx, roles, []string{ca.ManagerRole}, tca.Organization, nil)
		return err
	}
	authenticatedLogBrokerAPI := api.NewAuthenticatedWrapperLogBrokerServer(broker, authorize)

	api.RegisterLogBrokerServer(brokerServer, authenticatedLogBrokerAPI)
	go func() {
		if err := brokerServer.Serve(brokerListener); err != nil {
			// SIGH(stevvooe): GRPC won't really shutdown gracefully.
			// This should be fatal.
			t.Logf("error serving grpc service: %v", err)
		}
	}()

	require.NoError(t, broker.Start(ctx))

	return ctx, tca, broker, logListener.Addr().String(), brokerListener.Addr().String(), func() {
		broker.Stop()

		logServer.Stop()
		brokerServer.Stop()

		logListener.Close()
		brokerListener.Close()

		cancel()
	}
}

func testLogClient(t *testing.T, addr string) (api.LogsClient, func()) {
	// Log client
	logCc, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("error dialing local server: %v", err)
	}
	return api.NewLogsClient(logCc), func() {
		logCc.Close()
	}
}

func testBrokerClient(t *testing.T, tca *testutils.TestCA, addr string) (api.LogBrokerClient, *ca.SecurityConfig, func()) {
	securityConfig, err := tca.NewNodeConfig(ca.WorkerRole)
	if err != nil {
		t.Fatal(err)
	}

	opts := []grpc.DialOption{grpc.WithTimeout(10 * time.Second), grpc.WithTransportCredentials(securityConfig.ClientTLSCreds)}
	cc, err := grpc.Dial(addr, opts...)
	if err != nil {
		t.Fatalf("error dialing local server: %v", err)
	}

	return api.NewLogBrokerClient(cc), securityConfig, func() {
		cc.Close()
	}
}

func printLogMessages(msgs ...api.LogMessage) {
	for _, msg := range msgs {
		ts, _ := gogotypes.TimestampFromProto(msg.Timestamp)
		fmt.Printf("%v %v %s\n", msg.Context, ts, string(msg.Data))
	}
}

// newLogMessage is just a helper to build a new log message.
func newLogMessage(msgctx api.LogContext, format string, vs ...interface{}) api.LogMessage {
	return api.LogMessage{
		Context:   msgctx,
		Timestamp: ptypes.MustTimestampProto(time.Now()),
		Data:      []byte(fmt.Sprintf(format, vs...)),
	}
}
