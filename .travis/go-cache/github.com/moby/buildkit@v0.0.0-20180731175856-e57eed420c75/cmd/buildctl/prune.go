package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/moby/buildkit/client"
	"github.com/tonistiigi/units"
	"github.com/urfave/cli"
)

var pruneCommand = cli.Command{
	Name:   "prune",
	Usage:  "clean up build cache",
	Action: prune,
	Flags: []cli.Flag{
		cli.StringSliceFlag{
			Name:  "filter, f",
			Usage: "Filter records",
		},
		cli.BoolFlag{
			Name:  "all",
			Usage: "Include internal/frontend references",
		},
		cli.BoolFlag{
			Name:  "verbose, v",
			Usage: "Verbose output",
		},
	},
}

func prune(clicontext *cli.Context) error {
	c, err := resolveClient(clicontext)
	if err != nil {
		return err
	}

	ch := make(chan client.UsageInfo)
	printed := make(chan struct{})

	tw := tabwriter.NewWriter(os.Stdout, 1, 8, 1, '\t', 0)
	first := true
	total := int64(0)

	go func() {
		defer close(printed)
		for du := range ch {
			total += du.Size
			if clicontext.Bool("verbose") {
				printVerbose(tw, []*client.UsageInfo{&du})
			} else {
				if first {
					printTableHeader(tw)
					first = false
				}
				printTableRow(tw, &du)
				tw.Flush()
			}
		}
	}()

	opts := []client.PruneOption{client.WithFilter(clicontext.StringSlice("filter"))}

	if clicontext.Bool("all") {
		opts = append(opts, client.PruneAll)
	}

	err = c.Prune(commandContext(clicontext), ch, opts...)
	close(ch)
	<-printed
	if err != nil {
		return err
	}

	tw = tabwriter.NewWriter(os.Stdout, 1, 8, 1, '\t', 0)
	fmt.Fprintf(tw, "Total:\t%.2f\n", units.Bytes(total))
	tw.Flush()

	return nil
}
