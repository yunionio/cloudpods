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

package progress

// ComputeStats type supports computing running average with a given window size.
//
type ComputeStats struct {
	history []float64
	size    int
	p       int
}

// NewComputeStats returns a new instance of ComputeStats. The parameter size is the
// maximum number of values that ComputeStats track at any given point of time.
//
func NewComputeStats(size int) *ComputeStats {
	return &ComputeStats{
		history: make([]float64, size),
		size:    size,
		p:       0,
	}
}

// NewComputestateDefaultSize returns a new instance of ComputeStats that can tracks
// maximum of 60 values.
//
func NewComputestateDefaultSize() *ComputeStats {
	return NewComputeStats(60)
}

// ComputeAvg adds the given value to a list containing set of previous values added
// and returns the average of the values in the list. If the values list reached the
// maximum size then oldest value will be removed
//
func (s *ComputeStats) ComputeAvg(current float64) float64 {
	s.history[s.p] = current
	s.p++
	if s.p == s.size {
		s.p = 0
	}

	sum := float64(0)
	for i := 0; i < s.size; i++ {
		sum += s.history[i]
	}

	return sum / float64(s.size)
}
