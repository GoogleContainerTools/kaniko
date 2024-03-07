/*
Copyright 2023 Google LLC

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
