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

package snapshot

type LayeredMap struct {
	layers []map[string]string
	hasher func(string) string
}

func NewLayeredMap(h func(string) string) *LayeredMap {
	l := LayeredMap{
		hasher: h,
	}
	l.layers = []map[string]string{}
	return &l
}

func (l *LayeredMap) Snapshot() {
	l.layers = append(l.layers, map[string]string{})
}

func (l *LayeredMap) Get(s string) (string, bool) {
	for i := len(l.layers) - 1; i >= 0; i-- {
		if v, ok := l.layers[i][s]; ok {
			return v, ok
		}
	}
	return "", false
}

func (l *LayeredMap) MaybeAdd(s string) bool {
	oldV, ok := l.Get(s)
	newV := l.hasher(s)
	if ok && newV == oldV {
		return false
	}
	l.layers[len(l.layers)-1][s] = newV
	return true
}
