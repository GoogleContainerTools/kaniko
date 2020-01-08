// Copyright 2018, OpenCensus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package view

import (
	"reflect"
	"testing"
)

func TestDataClone(t *testing.T) {
	dist := newDistributionData([]float64{1, 2, 3, 4})
	dist.Count = 7
	dist.Max = 11
	dist.Min = 1
	dist.CountPerBucket = []int64{0, 2, 3, 2}
	dist.Mean = 4
	dist.SumOfSquaredDev = 1.2

	tests := []struct {
		name string
		src  AggregationData
	}{
		{
			name: "count data",
			src:  &CountData{Value: 5},
		},
		{
			name: "distribution data",
			src:  dist,
		},
		{
			name: "sum data",
			src:  &SumData{Value: 65.7},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.src.clone()
			if !reflect.DeepEqual(got, tt.src) {
				t.Errorf("AggregationData.clone() = %v, want %v", got, tt.src)
			}
			// TODO(jbd): Make sure that data is deep copied.
			if got == tt.src {
				t.Errorf("AggregationData.clone() returned the same pointer")
			}
		})
	}
}
