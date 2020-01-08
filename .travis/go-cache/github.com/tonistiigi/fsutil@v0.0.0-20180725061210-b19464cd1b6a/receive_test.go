package fsutil

import (
	"bytes"
	"context"
	"crypto/sha256"
	"hash"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestInvalidExcludePatterns(t *testing.T) {
	d, err := tmpDir(changeStream([]string{
		"ADD foo file data1",
	}))
	assert.NoError(t, err)
	defer os.RemoveAll(d)

	dest, err := ioutil.TempDir("", "dest")
	assert.NoError(t, err)
	defer os.RemoveAll(dest)

	ts := newNotificationBuffer()
	chs := &changes{fn: ts.HandleChange}

	eg, ctx := errgroup.WithContext(context.Background())
	s1, s2 := sockPairProto(ctx)

	eg.Go(func() error {
		defer s1.(*fakeConnProto).closeSend()
		return Send(ctx, s1, NewFS(d, &WalkOpt{ExcludePatterns: []string{"!"}}), nil)
	})
	eg.Go(func() error {
		return Receive(ctx, s2, dest, ReceiveOpt{
			NotifyHashed:  chs.HandleChange,
			ContentHasher: simpleSHA256Hasher,
		})
	})

	errCh := make(chan error)
	go func() {
		errCh <- eg.Wait()
	}()
	select {
	case <-time.After(15 * time.Second):
		t.Fatal("timeout")
	case err = <-errCh:
		assert.Contains(t, err.Error(), "error from sender:")
	}
}

func TestCopyWithSubDir(t *testing.T) {
	d, err := tmpDir(changeStream([]string{
		"ADD foo dir",
		"ADD foo/bar file data1",
	}))
	assert.NoError(t, err)
	defer os.RemoveAll(d)

	dest, err := ioutil.TempDir("", "dest")
	assert.NoError(t, err)
	defer os.RemoveAll(dest)

	eg, ctx := errgroup.WithContext(context.Background())
	s1, s2 := sockPairProto(ctx)

	eg.Go(func() error {
		defer s1.(*fakeConnProto).closeSend()
		return Send(ctx, s1, SubDirFS(NewFS(d, &WalkOpt{}), Stat{Path: "sub", Mode: uint32(os.ModeDir | 0755)}), nil)
	})
	eg.Go(func() error {
		return Receive(ctx, s2, dest, ReceiveOpt{})
	})

	err = eg.Wait()
	assert.NoError(t, err)

	dt, err := ioutil.ReadFile(filepath.Join(dest, "sub/foo/bar"))
	assert.NoError(t, err)
	assert.Equal(t, "data1", string(dt))
}

func TestCopySwitchDirToFile(t *testing.T) {
	d, err := tmpDir(changeStream([]string{
		"ADD foo file data1",
	}))
	assert.NoError(t, err)
	defer os.RemoveAll(d)

	dest, err := tmpDir(changeStream([]string{
		"ADD foo dir",
		"ADD foo/bar dile data2",
	}))
	assert.NoError(t, err)
	defer os.RemoveAll(d)

	copy := func(src, dest string) (*changes, error) {
		ts := newNotificationBuffer()
		chs := &changes{fn: ts.HandleChange}

		eg, ctx := errgroup.WithContext(context.Background())
		s1, s2 := sockPairProto(ctx)

		eg.Go(func() error {
			defer s1.(*fakeConnProto).closeSend()
			return Send(ctx, s1, NewFS(src, &WalkOpt{
				Map: func(s *Stat) bool {
					s.Uid = 0
					s.Gid = 0
					return true
				},
			}), nil)
		})
		eg.Go(func() error {
			return Receive(ctx, s2, dest, ReceiveOpt{
				NotifyHashed:  chs.HandleChange,
				ContentHasher: simpleSHA256Hasher,
				Filter: func(s *Stat) bool {
					s.Uid = uint32(os.Getuid())
					s.Gid = uint32(os.Getgid())
					return true
				},
			})
		})

		if err := eg.Wait(); err != nil {
			return nil, err
		}

		return chs, nil
	}

	chs, err := copy(d, dest)
	require.NoError(t, err)

	k, ok := chs.c["foo"]
	require.True(t, ok)
	require.Equal(t, k, ChangeKindAdd)
	require.Equal(t, len(chs.c), 1)

	b := &bytes.Buffer{}
	err = Walk(context.Background(), dest, nil, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, string(b.Bytes()), `file foo
`)
}

func TestCopySimple(t *testing.T) {
	d, err := tmpDir(changeStream([]string{
		"ADD foo file data1",
		"ADD foo2 file dat2",
		"ADD zzz dir",
		"ADD zzz/aa file data3",
		"ADD zzz/bb dir",
		"ADD zzz/bb/cc dir",
		"ADD zzz/bb/cc/dd symlink ../../",
		"ADD zzz.aa zzdata",
	}))
	assert.NoError(t, err)
	defer os.RemoveAll(d)

	dest, err := ioutil.TempDir("", "dest")
	assert.NoError(t, err)
	defer os.RemoveAll(dest)

	ts := newNotificationBuffer()
	chs := &changes{fn: ts.HandleChange}

	eg, ctx := errgroup.WithContext(context.Background())
	s1, s2 := sockPairProto(ctx)

	eg.Go(func() error {
		defer s1.(*fakeConnProto).closeSend()
		return Send(ctx, s1, NewFS(d, &WalkOpt{
			Map: func(s *Stat) bool {
				s.Uid = 0
				s.Gid = 0
				return true
			},
		}), nil)
	})
	eg.Go(func() error {
		return Receive(ctx, s2, dest, ReceiveOpt{
			NotifyHashed:  chs.HandleChange,
			ContentHasher: simpleSHA256Hasher,
			Filter: func(s *Stat) bool {
				s.Uid = uint32(os.Getuid())
				s.Gid = uint32(os.Getgid())
				return true
			},
		})
	})

	assert.NoError(t, eg.Wait())

	b := &bytes.Buffer{}
	err = Walk(context.Background(), dest, nil, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, string(b.Bytes()), `file foo
file foo2
dir zzz
file zzz/aa
dir zzz/bb
dir zzz/bb/cc
symlink:../../ zzz/bb/cc/dd
file zzz.aa
`)

	dt, err := ioutil.ReadFile(filepath.Join(dest, "zzz/aa"))
	assert.NoError(t, err)
	assert.Equal(t, "data3", string(dt))

	dt, err = ioutil.ReadFile(filepath.Join(dest, "foo2"))
	assert.NoError(t, err)
	assert.Equal(t, "dat2", string(dt))

	h, ok := ts.Hash("zzz/aa")
	assert.True(t, ok)
	assert.Equal(t, digest.Digest("sha256:99b6ef96ee0572b5b3a4eb28f00b715d820bfd73836e59cc1565e241f4d1bb2f"), h)

	h, ok = ts.Hash("foo2")
	assert.True(t, ok)
	assert.Equal(t, digest.Digest("sha256:dd2529f7749ba45ea55de3b2e10086d6494cc45a94e57650c2882a6a14b4ff32"), h)

	h, ok = ts.Hash("zzz/bb/cc/dd")
	assert.True(t, ok)
	assert.Equal(t, digest.Digest("sha256:eca07e8f2d09bd574ea2496312e6de1685ef15b8e6a49a534ed9e722bcac8adc"), h)

	k, ok := chs.c["zzz/aa"]
	assert.Equal(t, ok, true)
	assert.Equal(t, k, ChangeKindAdd)

	err = ioutil.WriteFile(filepath.Join(d, "zzz/bb/cc/foo"), []byte("data5"), 0600)
	assert.NoError(t, err)

	err = os.RemoveAll(filepath.Join(d, "foo2"))
	assert.NoError(t, err)

	chs = &changes{fn: ts.HandleChange}

	eg, ctx = errgroup.WithContext(context.Background())
	s1, s2 = sockPairProto(ctx)

	eg.Go(func() error {
		defer s1.(*fakeConnProto).closeSend()
		return Send(ctx, s1, NewFS(d, &WalkOpt{
			Map: func(s *Stat) bool {
				s.Uid = 0
				s.Gid = 0
				return true
			},
		}), nil)
	})
	eg.Go(func() error {
		return Receive(ctx, s2, dest, ReceiveOpt{
			NotifyHashed:  chs.HandleChange,
			ContentHasher: simpleSHA256Hasher,
			Filter: func(s *Stat) bool {
				s.Uid = uint32(os.Getuid())
				s.Gid = uint32(os.Getgid())
				return true
			},
		})
	})

	assert.NoError(t, eg.Wait())

	b = &bytes.Buffer{}
	err = Walk(context.Background(), dest, nil, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, string(b.Bytes()), `file foo
dir zzz
file zzz/aa
dir zzz/bb
dir zzz/bb/cc
symlink:../../ zzz/bb/cc/dd
file zzz/bb/cc/foo
file zzz.aa
`)

	dt, err = ioutil.ReadFile(filepath.Join(dest, "zzz/bb/cc/foo"))
	assert.NoError(t, err)
	assert.Equal(t, "data5", string(dt))

	h, ok = ts.Hash("zzz/bb/cc/dd")
	assert.True(t, ok)
	assert.Equal(t, digest.Digest("sha256:eca07e8f2d09bd574ea2496312e6de1685ef15b8e6a49a534ed9e722bcac8adc"), h)

	h, ok = ts.Hash("zzz/bb/cc/foo")
	assert.True(t, ok)
	assert.Equal(t, digest.Digest("sha256:cd14a931fc2e123ded338093f2864b173eecdee578bba6ec24d0724272326c3a"), h)

	_, ok = ts.Hash("foo2")
	assert.False(t, ok)

	k, ok = chs.c["foo2"]
	assert.Equal(t, ok, true)
	assert.Equal(t, k, ChangeKindDelete)

	k, ok = chs.c["zzz/bb/cc/foo"]
	assert.Equal(t, ok, true)
	assert.Equal(t, k, ChangeKindAdd)

	_, ok = chs.c["zzz/aa"]
	assert.Equal(t, ok, false)

	_, ok = chs.c["zzz.aa"]
	assert.Equal(t, ok, false)
}

func sockPair(ctx context.Context) (Stream, Stream) {
	c1 := make(chan *Packet, 32)
	c2 := make(chan *Packet, 32)
	return &fakeConn{ctx, c1, c2}, &fakeConn{ctx, c2, c1}
}

func sockPairProto(ctx context.Context) (Stream, Stream) {
	c1 := make(chan []byte, 32)
	c2 := make(chan []byte, 32)
	return &fakeConnProto{ctx, c1, c2}, &fakeConnProto{ctx, c2, c1}
}

type fakeConn struct {
	ctx      context.Context
	recvChan chan *Packet
	sendChan chan *Packet
}

func (fc *fakeConn) Context() context.Context {
	return fc.ctx
}

func (fc *fakeConn) RecvMsg(m interface{}) error {
	p, ok := m.(*Packet)
	if !ok {
		return errors.Errorf("invalid msg: %#v", m)
	}
	select {
	case <-fc.ctx.Done():
		return fc.ctx.Err()
	case p2 := <-fc.recvChan:
		*p = *p2
		return nil
	}
}

func (fc *fakeConn) SendMsg(m interface{}) error {
	p, ok := m.(*Packet)
	if !ok {
		return errors.Errorf("invalid msg: %#v", m)
	}
	p2 := *p
	p2.Data = append([]byte{}, p2.Data...)
	select {
	case <-fc.ctx.Done():
		return fc.ctx.Err()
	case fc.sendChan <- &p2:
		return nil
	}
}

type fakeConnProto struct {
	ctx      context.Context
	recvChan chan []byte
	sendChan chan []byte
}

func (fc *fakeConnProto) Context() context.Context {
	return fc.ctx
}

func (fc *fakeConnProto) RecvMsg(m interface{}) error {
	p, ok := m.(*Packet)
	if !ok {
		return errors.Errorf("invalid msg: %#v", m)
	}
	select {
	case <-fc.ctx.Done():
		return fc.ctx.Err()
	case dt, ok := <-fc.recvChan:
		if !ok {
			return io.EOF
		}
		return p.Unmarshal(dt)
	}
}

func (fc *fakeConnProto) SendMsg(m interface{}) error {
	p, ok := m.(*Packet)
	if !ok {
		return errors.Errorf("invalid msg: %#v", m)
	}
	dt, err := p.Marshal()
	if err != nil {
		return err
	}
	select {
	case <-fc.ctx.Done():
		return fc.ctx.Err()
	case fc.sendChan <- dt:
		return nil
	}
}

func (fc *fakeConnProto) closeSend() {
	close(fc.sendChan)
}

type changes struct {
	c  map[string]ChangeKind
	fn ChangeFunc
	mu sync.Mutex
}

func (c *changes) HandleChange(kind ChangeKind, p string, fi os.FileInfo, err error) error {
	c.mu.Lock()
	if c.c == nil {
		c.c = make(map[string]ChangeKind)
	}
	c.c[p] = kind
	c.mu.Unlock()
	return c.fn(kind, p, fi, err)
}

func simpleSHA256Hasher(s *Stat) (hash.Hash, error) {
	h := sha256.New()
	ss := *s
	ss.ModTime = 0

	if os.FileMode(ss.Mode)&os.ModeSymlink != 0 {
		ss.Mode = ss.Mode | 0777
	}

	dt, err := ss.Marshal()
	if err != nil {
		return nil, err
	}
	h.Write(dt)
	return h, nil
}
