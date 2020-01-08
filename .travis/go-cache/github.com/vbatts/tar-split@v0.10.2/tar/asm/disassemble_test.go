package asm

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/vbatts/tar-split/tar/storage"
)

// This test failing causes the binary to crash due to memory overcommitment.
func TestLargeJunkPadding(t *testing.T) {
	pR, pW := io.Pipe()

	// Write a normal tar file into the pipe and then load it full of junk
	// bytes as padding. We have to do this in a goroutine because we can't
	// store 20GB of junk in-memory.
	go func() {
		// Empty archive.
		tw := tar.NewWriter(pW)
		if err := tw.Close(); err != nil {
			pW.CloseWithError(err)
			t.Fatal(err)
			return
		}

		// Write junk.
		const (
			junkChunkSize = 64 * 1024 * 1024
			junkChunkNum  = 20 * 16
		)
		devZero, err := os.Open("/dev/zero")
		if err != nil {
			pW.CloseWithError(err)
			t.Fatal(err)
			return
		}
		defer devZero.Close()
		for i := 0; i < junkChunkNum; i++ {
			if i%32 == 0 {
				fmt.Fprintf(os.Stderr, "[TestLargeJunkPadding] junk chunk #%d/#%d\n", i, junkChunkNum)
			}
			if _, err := io.CopyN(pW, devZero, junkChunkSize); err != nil {
				pW.CloseWithError(err)
				t.Fatal(err)
				return
			}
		}

		fmt.Fprintln(os.Stderr, "[TestLargeJunkPadding] junk chunk finished")
		pW.Close()
	}()

	// Disassemble our junk file.
	nilPacker := storage.NewJSONPacker(ioutil.Discard)
	rdr, err := NewInputTarStream(pR, nilPacker, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Copy the entire rdr.
	_, err = io.Copy(ioutil.Discard, rdr)
	if err != nil {
		t.Fatal(err)
	}

	// At this point, if we haven't crashed then we are not vulnerable to
	// CVE-2017-14992.
}
