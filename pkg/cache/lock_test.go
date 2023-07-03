package cache

import (
	"fmt"
	"log"
	"testing"
	"time"
)

func TestClearDeadlock(t *testing.T) {
	// Create a deadlock.
	lockfile := "deadlock.lock"
	fl := FLock(lockfile)
	if fl == nil {
		fmt.Printf("deadlock already existed")
	} else {
		fl.stopBeat <- true
	}

	for retryCounter := 10; retryCounter > 0; retryCounter-- {
		fl := FLock(lockfile)
		if fl == nil {
			log.Println("get lock failed, try to clear")
			ClearDeadlock(lockfile)
			time.Sleep(1 * time.Second)
		} else {
			defer fl.Unlock()
			log.Println("got lock!")
			return
		}
	}
	t.Error("clear deadlock timeout")
}
