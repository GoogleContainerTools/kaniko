package main

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/moby/buildkit/client"
	"github.com/tonistiigi/units"
	"github.com/urfave/cli"
)

var diskUsageCommand = cli.Command{
	Name:   "du",
	Usage:  "disk usage",
	Action: diskUsage,
	Flags: []cli.Flag{
		cli.StringSliceFlag{
			Name:  "filter, f",
			Usage: "Filter records",
		},
		cli.BoolFlag{
			Name:  "verbose, v",
			Usage: "Verbose output",
		},
	},
}

func diskUsage(clicontext *cli.Context) error {
	c, err := resolveClient(clicontext)
	if err != nil {
		return err
	}

	du, err := c.DiskUsage(commandContext(clicontext), client.WithFilter(clicontext.StringSlice("filter")))
	if err != nil {
		return err
	}

	tw := tabwriter.NewWriter(os.Stdout, 1, 8, 1, '\t', 0)

	if clicontext.Bool("verbose") {
		printVerbose(tw, du)
	} else {
		printTable(tw, du)
	}

	if len(clicontext.StringSlice("filter")) == 0 {
		printSummary(tw, du)
	}

	return nil
}

func printKV(w io.Writer, k string, v interface{}) {
	fmt.Fprintf(w, "%s:\t%v\n", k, v)
}

func printVerbose(tw *tabwriter.Writer, du []*client.UsageInfo) {
	for _, di := range du {
		printKV(tw, "ID", di.ID)
		if di.Parent != "" {
			printKV(tw, "Parent", di.Parent)
		}
		printKV(tw, "Created at", di.CreatedAt)
		printKV(tw, "Mutable", di.Mutable)
		printKV(tw, "Reclaimable", !di.InUse)
		printKV(tw, "Shared", di.Shared)
		printKV(tw, "Size", fmt.Sprintf("%.2f", units.Bytes(di.Size)))
		if di.Description != "" {
			printKV(tw, "Description", di.Description)
		}
		printKV(tw, "Usage count", di.UsageCount)
		if di.LastUsedAt != nil {
			printKV(tw, "Last used", di.LastUsedAt)
		}
		if di.RecordType != "" {
			printKV(tw, "Type", di.RecordType)
		}

		fmt.Fprintf(tw, "\n")
	}

	tw.Flush()
}

func printTable(tw *tabwriter.Writer, du []*client.UsageInfo) {
	printTableHeader(tw)

	for _, di := range du {
		printTableRow(tw, di)
	}

	tw.Flush()
}

func printTableHeader(tw *tabwriter.Writer) {
	fmt.Fprintln(tw, "ID\tRECLAIMABLE\tSIZE\tLAST ACCESSED")
}

func printTableRow(tw *tabwriter.Writer, di *client.UsageInfo) {
	id := di.ID
	if di.Mutable {
		id += "*"
	}
	size := fmt.Sprintf("%.2f", units.Bytes(di.Size))
	if di.Shared {
		size += "*"
	}
	fmt.Fprintf(tw, "%-71s\t%-11v\t%s\t\n", id, !di.InUse, size)
}

func printSummary(tw *tabwriter.Writer, du []*client.UsageInfo) {
	total := int64(0)
	reclaimable := int64(0)
	shared := int64(0)

	for _, di := range du {
		if di.Size > 0 {
			total += di.Size
			if !di.InUse {
				reclaimable += di.Size
			}
		}
		if di.Shared {
			shared += di.Size
		}
	}

	tw = tabwriter.NewWriter(os.Stdout, 1, 8, 1, '\t', 0)

	if shared > 0 {
		fmt.Fprintf(tw, "Shared:\t%.2f\n", units.Bytes(shared))
		fmt.Fprintf(tw, "Private:\t%.2f\n", units.Bytes(total-shared))
	}

	fmt.Fprintf(tw, "Reclaimable:\t%.2f\n", units.Bytes(reclaimable))
	fmt.Fprintf(tw, "Total:\t%.2f\n", units.Bytes(total))
	tw.Flush()
}
