package main

import (
	"context"
	"flag"
	"os"

	"github.com/tonistiigi/fsutil"
	"github.com/tonistiigi/fsutil/util"
)

func main() {
	flag.Parse()
	if len(flag.Args()) == 0 {
		panic("source path not set")
	}

	ctx := context.Background()
	s := util.NewProtoStream(ctx, os.Stdin, os.Stdout)

	if err := fsutil.Send(ctx, s, fsutil.NewFS(flag.Args()[0], nil), nil); err != nil {
		panic(err)
	}
}
