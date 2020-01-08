package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/docker/swarmkit/api"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// Config holds the benchmarking configuration.
type Config struct {
	Count   uint64
	Manager string
	IP      string
	Port    int
	Unit    time.Duration
}

// Benchmark represents a benchmark session.
type Benchmark struct {
	cfg       *Config
	collector *Collector
}

// NewBenchmark creates a new benchmark session with the given configuration.
func NewBenchmark(cfg *Config) *Benchmark {
	return &Benchmark{
		cfg:       cfg,
		collector: NewCollector(),
	}
}

// Run starts the benchmark session and waits for it to be completed.
func (b *Benchmark) Run(ctx context.Context) error {
	fmt.Printf("Listening for incoming connections at %s:%d\n", b.cfg.IP, b.cfg.Port)
	if err := b.collector.Listen(b.cfg.Port); err != nil {
		return err
	}
	j, err := b.launch(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Service %s launched (%d instances)\n", j.ID, b.cfg.Count)

	// Periodically print stats.
	doneCh := make(chan struct{})
	go func() {
		for {
			select {
			case <-time.After(5 * time.Second):
				fmt.Printf("\n%s: Progression report\n", time.Now())
				b.collector.Stats(os.Stdout, time.Second)
			case <-doneCh:
				return
			}
		}
	}()

	fmt.Println("Collecting metrics...")
	b.collector.Collect(ctx, b.cfg.Count)
	doneCh <- struct{}{}

	fmt.Printf("\n%s: Benchmark completed\n", time.Now())
	b.collector.Stats(os.Stdout, time.Second)

	return nil
}

func (b *Benchmark) spec() *api.ServiceSpec {
	return &api.ServiceSpec{
		Annotations: api.Annotations{
			Name: "benchmark",
		},
		Task: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{
					Image:   "alpine:latest",
					Command: []string{"nc", b.cfg.IP, strconv.Itoa(b.cfg.Port)},
				},
			},
		},
		Mode: &api.ServiceSpec_Replicated{
			Replicated: &api.ReplicatedService{
				Replicas: b.cfg.Count,
			},
		},
	}
}

func (b *Benchmark) launch(ctx context.Context) (*api.Service, error) {
	conn, err := grpc.Dial(b.cfg.Manager, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	client := api.NewControlClient(conn)
	r, err := client.CreateService(ctx, &api.CreateServiceRequest{
		Spec: b.spec(),
	})
	if err != nil {
		return nil, err
	}
	return r.Service, nil
}
