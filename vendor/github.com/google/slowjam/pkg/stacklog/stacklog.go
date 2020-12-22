/*
Copyright 2020 Google LLC

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

// Package stacklog logs the Go stack to disk in a loop for later analysis
package stacklog

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

var (
	// DefaultPoll is how often to poll stack status by default
	defaultPoll = 125 * time.Millisecond

	// DefaultQuiet can be set to disable stderr messages by default
	defaultQuiet = false
)

// Config defines how to configure a stack logger.
type Config struct {
	Path  string
	Poll  time.Duration
	Quiet bool
}

// Start begins logging stacks to an output file.
func Start(c Config) (*Stacklog, error) {
	if c.Poll == 0 {
		c.Poll = defaultPoll
	}

	if c.Path == "" {
		tf, err := ioutil.TempFile("", "*.slog")
		if err != nil {
			return nil, fmt.Errorf("default path: %w", err)
		}

		c.Path = tf.Name()
	}

	if !c.Quiet {
		fmt.Fprintf(os.Stderr, "stacklog: logging to %s, sampling every %s\n", c.Path, c.Poll)
	}

	s := &Stacklog{
		ticker: time.NewTicker(c.Poll),
		path:   c.Path,
		quiet:  c.Quiet,
	}

	f, err := os.Create(c.Path)
	if err != nil {
		return s, err
	}

	s.f = f
	go s.loop()

	return s, nil
}

// MustStartFromEnv logs stacks to an output file based on the environment.
func MustStartFromEnv(key string) *Stacklog {
	val := os.Getenv(key)
	if val == "" {
		return &Stacklog{}
	}

	s, err := Start(Config{Path: val, Quiet: defaultQuiet, Poll: defaultPoll})
	if err != nil {
		panic(fmt.Sprintf("stacklog from environment %q: %v", key, err))
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		s.Stop()
	}()

	return s
}

// Stacklog controls the stack logger.
type Stacklog struct {
	ticker  *time.Ticker
	f       *os.File
	quiet   bool
	path    string
	samples int
}

// loop periodically records the stack log to disk.
func (s *Stacklog) loop() {
	for range s.ticker.C {
		if _, err := s.f.Write([]byte(fmt.Sprintf("%d\n", time.Now().UnixNano()))); err != nil {
			if !s.quiet {
				fmt.Fprintf(os.Stderr, "stacklog: write failed: %v", err)
			}
		}

		if _, err := s.f.Write(DumpStacks()); err != nil {
			if !s.quiet {
				fmt.Fprintf(os.Stderr, "stacklog: write failed: %v", err)
			}
		}

		if _, err := s.f.Write([]byte("-\n")); err != nil {
			if !s.quiet {
				fmt.Fprintf(os.Stderr, "stacklog: write failed: %v", err)
			}
		}

		s.samples++
	}
}

// DumpStacks returns a formatted stack trace of goroutines, using a large enough buffer to capture the entire trace.
func DumpStacks() []byte {
	buf := make([]byte, 1024)

	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			return buf[:n]
		}

		buf = make([]byte, 2*len(buf))
	}
}

// Stop stops logging stacks to disk.
func (s *Stacklog) Stop() {
	if s == nil || s.f == nil {
		return
	}

	s.ticker.Stop()

	if !s.quiet {
		fmt.Fprintf(os.Stderr, "stacklog: stopped. stored %d samples to %s\n", s.samples, s.path)
	}
}
