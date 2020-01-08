package main

import (
	dockerfile "github.com/moby/buildkit/frontend/dockerfile/builder"
	"github.com/moby/buildkit/frontend/gateway/grpcclient"
	"github.com/moby/buildkit/util/appcontext"
	"github.com/sirupsen/logrus"
)

func main() {
	if err := grpcclient.Run(appcontext.Context(), dockerfile.Build); err != nil {
		logrus.Errorf("fatal error: %+v", err)
		panic(err)
	}
}
