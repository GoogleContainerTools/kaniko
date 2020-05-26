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

package integration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type result struct {
	totalBuildTime float64
	resolvingFiles float64
	walkingFiles   float64
}

func TestSnapshotBenchmark(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	contextDir := filepath.Join(cwd, "benchmark_fs")

	nums := []int{10000, 50000, 100000, 200000, 300000, 500000, 700000, 800000}

	var timeMap sync.Map
	var wg sync.WaitGroup
	for _, num := range nums {
		t.Run(fmt.Sprintf("test_benchmark_%d", num), func(t *testing.T) {
			wg.Add(1)
			var err error
			go func(num int, err error) {
				dockerfile := "Dockerfile_fs_benchmark"
				kanikoImage := fmt.Sprintf("%s_%d", GetKanikoImage(config.imageRepo, dockerfile), num)
				buildArgs := []string{"--build-arg", fmt.Sprintf("NUM=%d", num)}
				var benchmarkDir string
				benchmarkDir, err = buildKanikoImage("", dockerfile,
					buildArgs, []string{}, kanikoImage, contextDir, config.gcsBucket,
					config.serviceAccount, false)
				if err != nil {
					return
				}
				r := newResult(t, filepath.Join(benchmarkDir, dockerfile))
				timeMap.Store(num, r)
				wg.Done()
				defer os.Remove(benchmarkDir)
			}(num, err)
			if err != nil {
				t.Errorf("could not run benchmark results for num %d due to %s", num, err)
			}
		})
	}
	wg.Wait()

	fmt.Println("Number of Files,Total Build Time,Walking Filesystem, Resolving Files")
	timeMap.Range(func(key interface{}, value interface{}) bool {
		d, _ := key.(int)
		v, _ := value.(result)
		fmt.Println(fmt.Sprintf("%d,%f,%f,%f", d, v.totalBuildTime, v.walkingFiles, v.resolvingFiles))
		return true
	})

}

func newResult(t *testing.T, f string) result {
	var current map[string]time.Duration
	jsonFile, err := os.Open(f)
	defer jsonFile.Close()
	if err != nil {
		t.Errorf("could not read benchmark file %s", f)
	}
	byteValue, _ := ioutil.ReadAll(jsonFile)
	if err := json.Unmarshal(byteValue, &current); err != nil {
		t.Errorf("could not unmarshal benchmark file")
	}
	r := result{}
	if c, ok := current["Resolving Paths"]; ok {
		r.resolvingFiles = c.Seconds()
	}
	if c, ok := current["Walking filesystem"]; ok {
		r.walkingFiles = c.Seconds()
	}
	if c, ok := current["Total Build Time"]; ok {
		r.totalBuildTime = c.Seconds()
	}
	return r
}
