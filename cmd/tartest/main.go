package main

import (
	"archive/tar"
	"fmt"
	"io"
	"path/filepath"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/sirupsen/logrus"
)

func main() {
	opts := new(config.KanikoOptions)
	opts.DockerfilePath = "/usr/local/google/home/cgwippern/kaniko-dev/issues/830/Dockerfile.base"
	stages, err := dockerfile.Stages(opts)
	if err != nil {
		panic(err)
	}

	fmt.Printf("There are %d stages\n", len(stages))
	for i, stage := range stages {
		fmt.Printf("working on stage %d\n", i)

		img, err := util.RetrieveSourceImage(stage, opts)
		if err != nil {
			panic(err)
		}

		fmt.Println(img)

		layers, err := img.Layers()
		if err != nil {
			panic(err)
		}

		root := "/"

		for _, l := range layers {
			r, err := l.Uncompressed()
			if err != nil {
				panic(err)
			}
			defer r.Close()

			tr := tar.NewReader(r)
			for {
				hdr, err := tr.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					panic(err)
				}
				path := filepath.Join(root, filepath.Clean(hdr.Name))

				if path == "/usr/share/bug/systemd" {
					logrus.Infof("First found it %s FileInfo %v Mode %v IsDir %t", path, hdr.FileInfo(), hdr.FileInfo().Mode(), hdr.FileInfo().IsDir())
					if !hdr.FileInfo().IsDir() {
						panic("/usr/share/bug/systemd in tar not recognized as a directory")
					}
				}
			}
		}
	}
}
