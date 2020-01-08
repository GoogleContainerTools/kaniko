// +build linux,!no_oci_worker

package main

import (
	"os/exec"

	ctdsnapshot "github.com/containerd/containerd/snapshots"
	"github.com/containerd/containerd/snapshots/native"
	"github.com/containerd/containerd/snapshots/overlay"
	"github.com/moby/buildkit/worker"
	"github.com/moby/buildkit/worker/base"
	"github.com/moby/buildkit/worker/runc"
	"github.com/opencontainers/runc/libcontainer/system"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func init() {
	flags := []cli.Flag{
		cli.StringFlag{
			Name:  "oci-worker",
			Usage: "enable oci workers (true/false/auto)",
			Value: "auto",
		},
		cli.StringSliceFlag{
			Name:  "oci-worker-labels",
			Usage: "user-specific annotation labels (com.example.foo=bar)",
		},
		cli.StringFlag{
			Name:  "oci-worker-snapshotter",
			Usage: "name of snapshotter (overlayfs or native)",
			Value: "auto",
		},
		cli.StringSliceFlag{
			Name:  "oci-worker-platform",
			Usage: "override supported platforms for worker",
		},
	}
	n := "oci-worker-rootless"
	u := "enable rootless mode"
	if system.RunningInUserNS() {
		flags = append(flags, cli.BoolTFlag{
			Name:  n,
			Usage: u,
		})
	} else {
		flags = append(flags, cli.BoolFlag{
			Name:  n,
			Usage: u,
		})
	}
	registerWorkerInitializer(
		workerInitializer{
			fn:       ociWorkerInitializer,
			priority: 0,
		},
		flags...,
	)
	// TODO: allow multiple oci runtimes
}

func ociWorkerInitializer(c *cli.Context, common workerInitializerOpt) ([]worker.Worker, error) {
	boolOrAuto, err := parseBoolOrAuto(c.GlobalString("oci-worker"))
	if err != nil {
		return nil, err
	}
	if (boolOrAuto == nil && !validOCIBinary()) || (boolOrAuto != nil && !*boolOrAuto) {
		return nil, nil
	}
	labels, err := attrMap(c.GlobalStringSlice("oci-worker-labels"))
	if err != nil {
		return nil, err
	}
	snFactory, err := snapshotterFactory(common.root, c.GlobalString("oci-worker-snapshotter"))
	if err != nil {
		return nil, err
	}
	// GlobalBool works for BoolT as well
	rootless := c.GlobalBool("oci-worker-rootless") || c.GlobalBool("rootless")
	if rootless {
		logrus.Debugf("running in rootless mode")
	}
	opt, err := runc.NewWorkerOpt(common.root, snFactory, rootless, labels)
	if err != nil {
		return nil, err
	}
	opt.SessionManager = common.sessionManager
	platformsStr := c.GlobalStringSlice("oci-worker-platform")
	if len(platformsStr) != 0 {
		platforms, err := parsePlatforms(platformsStr)
		if err != nil {
			return nil, errors.Wrap(err, "invalid platforms")
		}
		opt.Platforms = platforms
	}
	w, err := base.NewWorker(opt)
	if err != nil {
		return nil, err
	}
	return []worker.Worker{w}, nil
}

func snapshotterFactory(commonRoot, name string) (runc.SnapshotterFactory, error) {
	if name == "auto" {
		if err := overlay.Supported(commonRoot); err == nil {
			logrus.Debug("auto snapshotter: using overlayfs")
			name = "overlayfs"
		} else {
			logrus.Debugf("auto snapshotter: using native, because overlayfs is not available for %s: %v", commonRoot, err)
			name = "native"
		}
	}
	snFactory := runc.SnapshotterFactory{
		Name: name,
	}
	switch name {
	case "native":
		snFactory.New = native.NewSnapshotter
	case "overlayfs": // not "overlay", for consistency with containerd snapshotter plugin ID.
		snFactory.New = func(root string) (ctdsnapshot.Snapshotter, error) {
			return overlay.NewSnapshotter(root)
		}
	default:
		return snFactory, errors.Errorf("unknown snapshotter name: %q", name)
	}
	return snFactory, nil
}

func validOCIBinary() bool {
	_, err := exec.LookPath("runc")
	if err != nil {
		logrus.Warnf("skipping oci worker, as runc does not exist")
		return false
	}
	return true
}
