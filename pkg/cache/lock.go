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
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

const defaultHeartbeat = time.Second

type FileLock struct {
	stopBeat  chan bool
	lockPath  string
	heartbeat time.Duration
}

func (fl *FileLock) keepAlive() {
	for {
		select {
		case s := <-fl.stopBeat:
			if s {
				return
			}
		default:
			f, err := os.OpenFile(fl.lockPath, os.O_WRONLY, 0600)
			if err != nil {
				if os.IsNotExist(err) {
					logrus.Errorf("Lock file does not exist: %s", fl.lockPath)
				}
				logrus.Errorf("Failed to open lock file: %s", err)
			}
			fmt.Fprint(f, strconv.FormatInt(time.Now().Unix(), 10))
			f.Close()
			time.Sleep(fl.heartbeat)
		}
	}
}
func (fl *FileLock) Unlock() {
	fl.stopBeat <- true
	os.Remove(fl.lockPath)
}

func FLock(lockPath string) (filelock *FileLock) {
	lockfile, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_EXCL|os.O_CREATE, 0666)
	if err != nil {
		return nil
	}
	fmt.Fprint(lockfile, strconv.FormatInt(time.Now().Unix(), 10))
	defer lockfile.Close()
	lock := &FileLock{
		stopBeat:  make(chan bool),
		lockPath:  lockPath,
		heartbeat: defaultHeartbeat,
	}
	go lock.keepAlive()
	return lock
}

const expireDura = time.Second * 5

func isDeadlock(lockPath string) (bool, error) {
	f, err := os.Stat(lockPath)
	if err != nil {
		// If not exist at this moment, treat as unexpired.
		if os.IsNotExist(err) {
			return false, nil
		}
		logrus.Errorf("Failed to read lockfile %s timestamp: %s", lockPath, err)
		return false, err
	}
	modTime := f.ModTime()
	expTime := modTime.Add(expireDura)
	return time.Now().After(expTime), nil
}

func ClearDeadlock(lockPath string) bool {
	expired, _ := isDeadlock(lockPath) // Ignore error and try again.
	if expired {
		removeLockPath := lockPath + "-remove"
		fl := FLock(removeLockPath) // Add a remove-lock for remove operating.
		if fl == nil {
			if expired, _ := isDeadlock(removeLockPath); expired {
				// Remove remove-lock directly, remove main-lock should be done in 5 secs.
				// Don't remove main-lock here, retry to remove lock in next attempt.
				os.Remove(removeLockPath)
				return false
			}
		} else {
			os.Remove(lockPath)
			defer fl.Unlock()
			return true
		}
	}
	return false
}
