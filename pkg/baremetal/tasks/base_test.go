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

package tasks

import (
	"testing"
)

func TestQueue(t *testing.T) {
	type test struct {
		queue    *Queue
		expected string
	}
	q123 := NewQueue().Append("1").Append("2").Append("3")
	q123Pop := NewQueue().Append("1").Append("2").Append("3")
	q123Pop.Pop()
	qEmptyPop := NewQueue().Append("1").Append("2")
	qEmptyPop.Pop()
	qEmptyPop.Pop()
	qEmptyPop.Pop()
	tests := map[string]test{
		"Empty queue": {
			queue:    NewQueue(),
			expected: "[]",
		},
		"Queue append": {
			queue:    q123,
			expected: "[1 2 3]",
		},
		"Queue pop": {
			queue:    q123Pop,
			expected: "[2 3]",
		},
		"Queue pop to empty": {
			queue:    qEmptyPop,
			expected: "[]",
		},
	}
	for name, testCase := range tests {
		output := testCase.queue.String()
		expected := testCase.expected
		if output != expected {
			t.Errorf("TestCase %q failed, output: %v, expected: %v", name, output, expected)
		}
	}
}
