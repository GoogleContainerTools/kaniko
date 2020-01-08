package main

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/docker/swarmkit/log"
	"github.com/rcrowley/go-metrics"
	"golang.org/x/net/context"
)

// Collector waits for tasks to phone home while collecting statistics.
type Collector struct {
	t  metrics.Timer
	ln net.Listener
}

// Listen starts listening on a TCP port. Tasks have to connect to this address
// once they come online.
func (c *Collector) Listen(port int) error {
	var err error
	c.ln, err = net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return err
	}
	return nil
}

// Collect blocks until `count` tasks phoned home.
func (c *Collector) Collect(ctx context.Context, count uint64) {
	start := time.Now()
	for i := uint64(0); i < count; i++ {
		conn, err := c.ln.Accept()
		if err != nil {
			log.G(ctx).WithError(err).Error("failure accepting connection")
			continue
		}
		c.t.UpdateSince(start)
		conn.Close()
	}
}

// Stats prints various statistics related to the collection.
func (c *Collector) Stats(w io.Writer, unit time.Duration) {
	du := float64(unit)
	duSuffix := unit.String()[1:]

	t := c.t.Snapshot()
	ps := t.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})

	fmt.Fprintln(w, "stats:")
	fmt.Fprintf(w, "  count:       %9d\n", t.Count())
	fmt.Fprintf(w, "  min:         %12.2f%s\n", float64(t.Min())/du, duSuffix)
	fmt.Fprintf(w, "  max:         %12.2f%s\n", float64(t.Max())/du, duSuffix)
	fmt.Fprintf(w, "  mean:        %12.2f%s\n", t.Mean()/du, duSuffix)
	fmt.Fprintf(w, "  stddev:      %12.2f%s\n", t.StdDev()/du, duSuffix)
	fmt.Fprintf(w, "  median:      %12.2f%s\n", ps[0]/du, duSuffix)
	fmt.Fprintf(w, "  75%%:         %12.2f%s\n", ps[1]/du, duSuffix)
	fmt.Fprintf(w, "  95%%:         %12.2f%s\n", ps[2]/du, duSuffix)
	fmt.Fprintf(w, "  99%%:         %12.2f%s\n", ps[3]/du, duSuffix)
	fmt.Fprintf(w, "  99.9%%:       %12.2f%s\n", ps[4]/du, duSuffix)
	fmt.Fprintf(w, "  1-min rate:  %12.2f\n", t.Rate1())
	fmt.Fprintf(w, "  5-min rate:  %12.2f\n", t.Rate5())
	fmt.Fprintf(w, "  15-min rate: %12.2f\n", t.Rate15())
	fmt.Fprintf(w, "  mean rate:   %12.2f\n", t.RateMean())
}

// NewCollector creates and returns a collector.
func NewCollector() *Collector {
	return &Collector{
		t: metrics.NewTimer(),
	}
}
