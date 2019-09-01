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

// Uint2IntArray transfer bitmap displayed as n to int array
func Uint2IntArray(n uint32) []int {
	ret := make([]int, 0, 2)
	var i uint = 0
	for n != 0 {
		if n&(1<<i) != 0 {
			n &= uint32(^(1 << i))
			ret = append(ret, int(i))
		}
		i++
	}
	return ret
}

// IntArray2Uint transfer int array nums to bitmap number
func IntArray2Uint(nums []int) uint32 {
	var ret uint32 = 0
	for _, i := range nums {
		ret |= (1 << uint(i))
	}
	return ret
}

// Determine if int slice a euqals b
func IntSliceEqual(a, b []int) bool {
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
