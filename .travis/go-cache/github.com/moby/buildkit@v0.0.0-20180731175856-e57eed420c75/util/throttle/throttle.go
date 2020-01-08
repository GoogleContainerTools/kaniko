package throttle

import (
	"sync"
	"time"
)

// Throttle wraps a function so that internal function does not get called
// more frequently than the specified duration.
func Throttle(d time.Duration, f func()) func() {
	var next, running bool
	var mu sync.Mutex
	return func() {
		mu.Lock()
		defer mu.Unlock()

		next = true
		if !running {
			running = true
			go func() {
				for {
					mu.Lock()
					if next == false {
						running = false
						mu.Unlock()
						return
					}
					mu.Unlock()
					time.Sleep(d)
					mu.Lock()
					next = false
					mu.Unlock()
					f()
				}
			}()
		}
	}
}
