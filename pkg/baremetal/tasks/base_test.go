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
	"reflect"
	"testing"
	"time"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
)

func TestHasIPMIAddress(t *testing.T) {
	tests := []struct {
		name string
		nic  *types.SNic
		want bool
	}{
		{name: "nil nic"},
		{name: "empty address", nic: &types.SNic{}},
		{name: "configured address", nic: &types.SNic{IpAddr: "192.0.2.10"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasIPMIAddress(tt.nic); got != tt.want {
				t.Errorf("hasIPMIAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSameIPMIAddress(t *testing.T) {
	tests := []struct {
		name  string
		left  string
		right string
		want  bool
	}{
		{name: "same IPv4", left: "192.0.2.10", right: "192.0.2.10", want: true},
		{name: "equivalent IPv6", left: "2001:0db8:0:0:0:0:0:10", right: "2001:db8::10", want: true},
		{name: "different addresses", left: "192.0.2.10", right: "192.0.2.11"},
		{name: "invalid address", left: "not-an-ip", right: "not-an-ip"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sameIPMIAddress(tt.left, tt.right); got != tt.want {
				t.Errorf("sameIPMIAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWaitIPMIAddress(t *testing.T) {
	tests := []struct {
		name      string
		responses []*types.SNic
		maxTries  int
		wantCalls int
		want      *types.SNic
	}{
		{name: "nil responses", responses: []*types.SNic{nil, nil}, maxTries: 2, wantCalls: 2},
		{name: "delayed address", responses: []*types.SNic{nil, {IpAddr: ""}, {IpAddr: "192.0.2.10"}}, maxTries: 3, wantCalls: 3, want: &types.SNic{IpAddr: "192.0.2.10"}},
		{name: "stops after configured address", responses: []*types.SNic{{IpAddr: "192.0.2.10"}, {IpAddr: "192.0.2.11"}}, maxTries: 2, wantCalls: 1, want: &types.SNic{IpAddr: "192.0.2.10"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			fetch := func() *types.SNic {
				response := tt.responses[calls]
				calls++
				return response
			}
			got := waitIPMIAddress(fetch, tt.maxTries, 0*time.Second)
			if calls != tt.wantCalls {
				t.Errorf("waitIPMIAddress() calls = %d, want %d", calls, tt.wantCalls)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("waitIPMIAddress() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

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
