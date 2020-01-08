/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package fifo

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestFifoCancel(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "fifos")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	leakCheckWg = &sync.WaitGroup{}
	defer func() {
		leakCheckWg = nil
	}()

	_, err = OpenFifo(context.Background(), filepath.Join(tmpdir, "f0"), syscall.O_RDONLY|syscall.O_NONBLOCK, 0600)
	assert.NotNil(t, err)

	assert.NoError(t, checkWgDone(leakCheckWg))

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	f, err := OpenFifo(ctx, filepath.Join(tmpdir, "f0"), syscall.O_RDONLY|syscall.O_CREAT|syscall.O_NONBLOCK, 0600)
	assert.NoError(t, err)

	b := make([]byte, 32)
	n, err := f.Read(b)
	assert.Equal(t, n, 0)
	assert.EqualError(t, err, "reading from a closed fifo")

	select {
	case <-ctx.Done():
	default:
		t.Fatal("context should have been done")
	}
	assert.NoError(t, checkWgDone(leakCheckWg))
}

func TestFifoReadWrite(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "fifos")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	leakCheckWg = &sync.WaitGroup{}
	defer func() {
		leakCheckWg = nil
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	r, err := OpenFifo(ctx, filepath.Join(tmpdir, "f0"), syscall.O_RDONLY|syscall.O_CREAT|syscall.O_NONBLOCK, 0600)
	assert.NoError(t, err)

	w, err := OpenFifo(ctx, filepath.Join(tmpdir, "f0"), syscall.O_WRONLY|syscall.O_NONBLOCK, 0)
	assert.NoError(t, err)

	_, err = w.Write([]byte("foo"))
	assert.NoError(t, err)

	b := make([]byte, 32)
	n, err := r.Read(b)
	assert.NoError(t, err)
	assert.Equal(t, string(b[:n]), "foo")

	err = r.Close()
	assert.NoError(t, err)

	_, err = w.Write([]byte("bar"))
	assert.NotNil(t, err)

	assert.NoError(t, checkWgDone(leakCheckWg))

	cancel()
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	w, err = OpenFifo(ctx, filepath.Join(tmpdir, "f1"), syscall.O_CREAT|syscall.O_WRONLY|syscall.O_NONBLOCK, 0600)
	assert.NoError(t, err)

	written := make(chan struct{})
	go func() {
		w.Write([]byte("baz"))
		close(written)
	}()

	time.Sleep(200 * time.Millisecond)

	r, err = OpenFifo(ctx, filepath.Join(tmpdir, "f1"), syscall.O_RDONLY|syscall.O_NONBLOCK, 0)
	assert.NoError(t, err)
	n, err = r.Read(b)
	assert.NoError(t, err)
	assert.Equal(t, string(b[:n]), "baz")
	select {
	case <-written:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("content should have been written")
	}

	_, err = w.Write([]byte("barbar")) // kernel-buffer
	assert.NoError(t, err)
	err = w.Close()
	assert.NoError(t, err)
	n, err = r.Read(b)
	assert.NoError(t, err)
	assert.Equal(t, string(b[:n]), "barbar")
	n, err = r.Read(b)
	assert.Equal(t, n, 0)
	assert.Equal(t, err, io.EOF)
	n, err = r.Read(b)
	assert.Equal(t, n, 0)
	assert.Equal(t, err, io.EOF)

	assert.NoError(t, checkWgDone(leakCheckWg))
}

func TestFifoCancelOneSide(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "fifos")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	leakCheckWg = &sync.WaitGroup{}
	defer func() {
		leakCheckWg = nil
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	f, err := OpenFifo(ctx, filepath.Join(tmpdir, "f0"), syscall.O_RDONLY|syscall.O_CREAT|syscall.O_NONBLOCK, 0600)
	assert.NoError(t, err)

	read := make(chan struct{})
	b := make([]byte, 32)
	go func() {
		_, err = f.Read(b)
		close(read)
	}()

	select {
	case <-read:
		t.Fatal("read should have blocked")
	case <-time.After(time.Second):
	}

	cerr := f.Close()
	assert.NoError(t, cerr)

	select {
	case <-read:
	case <-time.After(time.Second):
		t.Fatal("read should have unblocked")
	}

	assert.EqualError(t, err, "reading from a closed fifo")

	assert.NoError(t, checkWgDone(leakCheckWg))
}

func TestFifoBlocking(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "fifos")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	leakCheckWg = &sync.WaitGroup{}
	defer func() {
		leakCheckWg = nil
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err = OpenFifo(ctx, filepath.Join(tmpdir, "f0"), syscall.O_RDONLY|syscall.O_CREAT, 0600)
	assert.EqualError(t, err, "context deadline exceeded")

	select {
	case <-ctx.Done():
	default:
		t.Fatal("context should have been completed")
	}

	assert.NoError(t, checkWgDone(leakCheckWg))

	cancel()
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var rerr error
	var r io.ReadCloser
	readerOpen := make(chan struct{})
	go func() {
		r, rerr = OpenFifo(ctx, filepath.Join(tmpdir, "f1"), syscall.O_RDONLY|syscall.O_CREAT, 0600)
		close(readerOpen)
	}()

	time.Sleep(500 * time.Millisecond)
	w, err := OpenFifo(ctx, filepath.Join(tmpdir, "f1"), syscall.O_WRONLY, 0)
	assert.NoError(t, err)

	select {
	case <-readerOpen:
	case <-time.After(time.Second):
		t.Fatal("writer should have unblocke reader")
	}

	assert.NoError(t, rerr)

	_, err = w.Write([]byte("foobar"))
	assert.NoError(t, err)

	b := make([]byte, 32)
	n, err := r.Read(b)
	assert.NoError(t, err)
	assert.Equal(t, string(b[:n]), "foobar")

	assert.NoError(t, checkWgDone(leakCheckWg))

	err = w.Close()
	assert.NoError(t, err)
	n, err = r.Read(b)
	assert.Equal(t, n, 0)
	assert.Equal(t, err, io.EOF)

	assert.NoError(t, checkWgDone(leakCheckWg))
}

func TestFifoORDWR(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "fifos")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	leakCheckWg = &sync.WaitGroup{}
	defer func() {
		leakCheckWg = nil
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	f, err := OpenFifo(ctx, filepath.Join(tmpdir, "f0"), syscall.O_RDWR|syscall.O_CREAT, 0600)
	assert.NoError(t, err)

	_, err = f.Write([]byte("foobar"))
	assert.NoError(t, err)

	b := make([]byte, 32)
	n, err := f.Read(b)
	assert.NoError(t, err)
	assert.Equal(t, string(b[:n]), "foobar")

	r1, err := OpenFifo(ctx, filepath.Join(tmpdir, "f0"), syscall.O_RDONLY|syscall.O_NONBLOCK, 0)
	assert.NoError(t, err)

	_, err = f.Write([]byte("barbar"))
	assert.NoError(t, err)

	n, err = r1.Read(b)
	assert.NoError(t, err)
	assert.Equal(t, string(b[:n]), "barbar")

	r2, err := OpenFifo(ctx, filepath.Join(tmpdir, "f0"), syscall.O_RDONLY, 0)
	assert.NoError(t, err)

	_, err = f.Write([]byte("barbaz"))
	assert.NoError(t, err)

	n, err = r2.Read(b)
	assert.NoError(t, err)
	assert.Equal(t, string(b[:n]), "barbaz")

	err = r2.Close()
	assert.NoError(t, err)

	_, err = f.Write([]byte("bar123"))
	assert.NoError(t, err)

	n, err = r1.Read(b)
	assert.NoError(t, err)
	assert.Equal(t, string(b[:n]), "bar123")

	err = r1.Close()
	assert.NoError(t, err)

	_, err = f.Write([]byte("bar456"))
	assert.NoError(t, err)

	r2, err = OpenFifo(ctx, filepath.Join(tmpdir, "f0"), syscall.O_RDONLY, 0)
	assert.NoError(t, err)

	n, err = r2.Read(b)
	assert.NoError(t, err)
	assert.Equal(t, string(b[:n]), "bar456")

	err = f.Close()
	assert.NoError(t, err)

	n, err = r2.Read(b)
	assert.EqualError(t, err, io.EOF.Error())

	assert.NoError(t, checkWgDone(leakCheckWg))
}

func checkWgDone(wg *sync.WaitGroup) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done := make(chan struct{})
	go func() {
		wg.Wait() // No way to cancel
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
