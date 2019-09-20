/*
Copyright 2018 Google LLC

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

package timing

import (
	"bytes"
	"encoding/json"
	"sync"
	"text/template"
	"time"
)

// For testing
var currentTimeFunc = time.Now

// DefaultRun is the default "singleton" TimedRun instance.
var DefaultRun = NewTimedRun()

// TimedRun provides a running store of how long is spent in each category.
type TimedRun struct {
	cl         sync.Mutex
	categories map[string]time.Duration // protected by cl
}

// Stop stops the specified timer and increments the time spent in that category.
func (tr *TimedRun) Stop(t *Timer) {
	stop := currentTimeFunc()
	tr.cl.Lock()
	defer tr.cl.Unlock()
	if _, ok := tr.categories[t.category]; !ok {
		tr.categories[t.category] = 0
	}
	tr.categories[t.category] += stop.Sub(t.startTime)
}

// Start starts a new Timer and returns it.
func Start(category string) *Timer {
	t := Timer{
		category:  category,
		startTime: currentTimeFunc(),
	}
	return &t
}

// NewTimedRun returns an initialized TimedRun instance.
func NewTimedRun() *TimedRun {
	tr := TimedRun{
		categories: map[string]time.Duration{},
	}
	return &tr
}

// Timer represents a running timer.
type Timer struct {
	category  string
	startTime time.Time
}

// DefaultFormat is a default format string used by Summary.
var DefaultFormat = template.Must(template.New("").Parse("{{range $c, $t := .}}{{$c}}: {{$t}}\n{{end}}"))

// Summary outputs a summary of the DefaultTimedRun.
func Summary() string {
	return DefaultRun.Summary()
}

func JSON() (string, error) {
	return DefaultRun.JSON()
}

// Summary outputs a summary of the specified TimedRun.
func (tr *TimedRun) Summary() string {
	b := bytes.Buffer{}

	tr.cl.Lock()
	defer tr.cl.Unlock()
	DefaultFormat.Execute(&b, tr.categories)
	return b.String()
}

func (tr *TimedRun) JSON() (string, error) {
	b, err := json.Marshal(tr.categories)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
