// Copyright 2019 Yunion
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

package bitmap

import (
	"math"
	"testing"
)

func TestUint2IntArray(t *testing.T) {
	oneCase := make([]int, 32)
	for i := range oneCase {
		oneCase[i] = i
	}
	testCase := []struct {
		input uint32
		want  []int
	}{
		{24, []int{3, 4}},
		{0, []int{}},
		{math.MaxUint32, oneCase},
	}

	for _, tc := range testCase {
		real := Uint2IntArray(tc.input)
		if !sliceEqual(real, tc.want) {
			t.Fatalf("want %v, but %v\n", tc.want, real)
		}
	}
}

func sliceEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestIntArray2Uint(t *testing.T) {
	oneCase := make([]int, 32)
	for i := range oneCase {
		oneCase[i] = i
	}
	testCase := []struct {
		want  uint32
		input []int
	}{
		{24, []int{3, 4}},
		{0, []int{}},
		{math.MaxUint32, oneCase},
	}

	for _, tc := range testCase {
		real := IntArray2Uint(tc.input)
		if tc.want != real {
			t.Fatalf("want %d, but %d\n", tc.want, real)
		}
	}
}
