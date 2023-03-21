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
	"testing"
	"time"
)

func patchTime(timeFunc func() time.Time) func() {
	old := currentTimeFunc
	currentTimeFunc = timeFunc
	return func() {
		currentTimeFunc = old
	}
}

func mockTimeFunc(t time.Time) func() time.Time {
	return func() time.Time {
		return t
	}
}

func TestTimedRun_StartStop(t *testing.T) {
	type args struct {
		categories map[string]time.Duration
		category   string
		waitTime   time.Duration
	}
	tests := []struct {
		name string
		args args
		want time.Duration
	}{
		{
			name: "new category",
			args: args{
				categories: map[string]time.Duration{},
				category:   "foo",
				waitTime:   3 * time.Second,
			},
			want: 3 * time.Second,
		},
		{
			name: "existing category",
			args: args{
				categories: map[string]time.Duration{
					"foo": 4 * time.Second,
				},
				category: "foo",
				waitTime: 2 * time.Second,
			},
			want: 6 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &TimedRun{
				categories: tt.args.categories,
			}

			timer := Timer{
				category:  tt.args.category,
				startTime: time.Time{},
			}

			defer patchTime(mockTimeFunc(timer.startTime.Add(tt.args.waitTime)))()
			tr.Stop(&timer)
			if got := tr.categories[tt.args.category]; got != tt.want {
				t.Errorf("Expected %d, got %d", tt.want, got)
			}
		})
	}
}

func TestTimedRun_Summary(t *testing.T) {
	type fields struct {
		categories map[string]time.Duration
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "single key",
			fields: fields{
				categories: map[string]time.Duration{
					"foo": 3 * time.Second,
				},
			},
			want: "foo: 3s\n",
		},
		{
			name: "two keys",
			fields: fields{
				categories: map[string]time.Duration{
					"foo": 3 * time.Second,
					"bar": 1 * time.Second,
				},
			},
			want: "bar: 1s\nfoo: 3s\n",
		},
		{
			name: "units",
			fields: fields{
				categories: map[string]time.Duration{
					"foo": 3 * time.Second,
					"bar": 1 * time.Millisecond,
				},
			},
			want: "bar: 1ms\nfoo: 3s\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &TimedRun{
				categories: tt.fields.categories,
			}
			if got := tr.Summary(); got != tt.want {
				t.Errorf("TimedRun.Summary() = %v, want %v", got, tt.want)
			}
		})
	}
}
