package main

import (
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/vbatts/tar-split/tar/asm"
	"github.com/vbatts/tar-split/tar/storage"
)

func CommandDisasm(c *cli.Context) {
	if len(c.Args()) != 1 {
		logrus.Fatalf("please specify tar to be disabled <NAME|->")
	}
	if len(c.String("output")) == 0 {
		logrus.Fatalf("--output filename must be set")
	}

	// Set up the tar input stream
	var inputStream io.Reader
	if c.Args()[0] == "-" {
		inputStream = os.Stdin
	} else {
		fh, err := os.Open(c.Args()[0])
		if err != nil {
			logrus.Fatal(err)
		}
		defer fh.Close()
		inputStream = fh
	}

	// Set up the metadata storage
	mf, err := os.OpenFile(c.String("output"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(0600))
	if err != nil {
		logrus.Fatal(err)
	}
	defer mf.Close()
	mfz := gzip.NewWriter(mf)
	defer mfz.Close()
	metaPacker := storage.NewJSONPacker(mfz)

	// we're passing nil here for the file putter, because the ApplyDiff will
	// handle the extraction of the archive
	its, err := asm.NewInputTarStream(inputStream, metaPacker, nil)
	if err != nil {
		logrus.Fatal(err)
	}
	var out io.Writer
	if c.Bool("no-stdout") {
		out = ioutil.Discard
	} else {
		out = os.Stdout
	}
	i, err := io.Copy(out, its)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Infof("created %s from %s (read %d bytes)", c.String("output"), c.Args()[0], i)
}
