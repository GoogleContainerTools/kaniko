package heartbeat

import (
	"testing"
	"time"
)

func TestHeartbeatBeat(t *testing.T) {
	ch := make(chan struct{})
	hb := New(200*time.Millisecond, func() {
		close(ch)
	})
	for i := 0; i < 4; i++ {
		time.Sleep(100 * time.Millisecond)
		hb.Beat()
	}
	hb.Stop()
	select {
	case <-ch:
		t.Fatal("Heartbeat was expired")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestHeartbeatTimeout(t *testing.T) {
	ch := make(chan struct{})
	hb := New(100*time.Millisecond, func() {
		close(ch)
	})
	defer hb.Stop()
	select {
	case <-ch:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeoutFunc wasn't called in timely fashion")
	}
}

func TestHeartbeatReactivate(t *testing.T) {
	ch := make(chan struct{}, 2)
	hb := New(100*time.Millisecond, func() {
		ch <- struct{}{}
	})
	defer hb.Stop()
	time.Sleep(200 * time.Millisecond)
	hb.Beat()
	time.Sleep(200 * time.Millisecond)
	for i := 0; i < 2; i++ {
		select {
		case <-ch:
		case <-time.After(500 * time.Millisecond):
			t.Fatal("timeoutFunc wasn't called in timely fashion")
		}
	}
}

func TestHeartbeatUpdate(t *testing.T) {
	ch := make(chan struct{})
	hb := New(1*time.Second, func() {
		close(ch)
	})
	defer hb.Stop()
	hb.Update(100 * time.Millisecond)
	hb.Beat()
	time.Sleep(200 * time.Millisecond)
	select {
	case <-ch:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeoutFunc wasn't called in timely fashion")
	}
}
