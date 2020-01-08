package flightcontrol

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
)

func TestNoCancel(t *testing.T) {
	t.Parallel()
	g := &Group{}
	eg, ctx := errgroup.WithContext(context.Background())
	var r1, r2 string
	var counter int64
	f := testFunc(100*time.Millisecond, "bar", &counter)
	eg.Go(func() error {
		ret1, err := g.Do(ctx, "foo", f)
		if err != nil {
			return err
		}
		r1 = ret1.(string)
		return nil
	})
	eg.Go(func() error {
		ret2, err := g.Do(ctx, "foo", f)
		if err != nil {
			return err
		}
		r2 = ret2.(string)
		return nil
	})
	err := eg.Wait()
	assert.NoError(t, err)
	assert.Equal(t, "bar", r1)
	assert.Equal(t, "bar", r2)
	assert.Equal(t, counter, int64(1))
}

func TestCancelOne(t *testing.T) {
	t.Parallel()
	g := &Group{}
	eg, ctx := errgroup.WithContext(context.Background())
	var r1, r2 string
	var counter int64
	f := testFunc(100*time.Millisecond, "bar", &counter)
	ctx2, cancel := context.WithCancel(ctx)
	eg.Go(func() error {
		ret1, err := g.Do(ctx2, "foo", f)
		assert.Error(t, err)
		assert.Equal(t, errors.Cause(err), context.Canceled)
		if err == nil {
			r1 = ret1.(string)
		}
		return nil
	})
	eg.Go(func() error {
		ret2, err := g.Do(ctx, "foo", f)
		if err != nil {
			return err
		}
		r2 = ret2.(string)
		return nil
	})
	eg.Go(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(30 * time.Millisecond):
			cancel()
			return nil
		}
	})
	err := eg.Wait()
	assert.NoError(t, err)
	assert.Equal(t, "", r1)
	assert.Equal(t, "bar", r2)
	assert.Equal(t, counter, int64(1))
}

func TestCancelBoth(t *testing.T) {
	t.Parallel()
	g := &Group{}
	eg, ctx := errgroup.WithContext(context.Background())
	var r1, r2 string
	var counter int64
	f := testFunc(200*time.Millisecond, "bar", &counter)
	ctx2, cancel2 := context.WithCancel(ctx)
	ctx3, cancel3 := context.WithCancel(ctx)
	eg.Go(func() error {
		ret1, err := g.Do(ctx2, "foo", f)
		assert.Error(t, err)
		assert.Equal(t, errors.Cause(err), context.Canceled)
		if err == nil {
			r1 = ret1.(string)
		}
		return nil
	})
	eg.Go(func() error {
		ret2, err := g.Do(ctx3, "foo", f)
		assert.Error(t, err)
		assert.Equal(t, errors.Cause(err), context.Canceled)
		if err == nil {
			r2 = ret2.(string)
		}
		return nil
	})
	eg.Go(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(20 * time.Millisecond):
			cancel2()
			return nil
		}
	})
	eg.Go(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
			cancel3()
			return nil
		}
	})
	err := eg.Wait()
	assert.NoError(t, err)
	assert.Equal(t, "", r1)
	assert.Equal(t, "", r2)
	assert.Equal(t, counter, int64(1))

	ret1, err := g.Do(context.TODO(), "foo", f)
	assert.NoError(t, err)
	assert.Equal(t, ret1, "bar")

	f2 := testFunc(100*time.Millisecond, "baz", &counter)
	ret1, err = g.Do(context.TODO(), "foo", f2)
	assert.NoError(t, err)
	assert.Equal(t, ret1, "baz")
	ret1, err = g.Do(context.TODO(), "abc", f)
	assert.NoError(t, err)
	assert.Equal(t, ret1, "bar")

	assert.Equal(t, counter, int64(4))
}

func testFunc(wait time.Duration, ret string, counter *int64) func(ctx context.Context) (interface{}, error) {
	return func(ctx context.Context) (interface{}, error) {
		atomic.AddInt64(counter, 1)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
			return ret, nil
		}
	}
}
