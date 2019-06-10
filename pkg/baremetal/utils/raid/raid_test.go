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

package raid

import (
	"reflect"
	"testing"
)

func TestReverseLogicalArray(t *testing.T) {
	tests := []struct {
		name  string
		input []*RaidLogicalVolume
		want  []*RaidLogicalVolume
	}{
		{
			name:  "empty input",
			input: []*RaidLogicalVolume{},
			want:  []*RaidLogicalVolume{},
		},
		{
			name: "reverse",
			input: []*RaidLogicalVolume{
				{Index: 1},
				{Index: 2},
			},
			want: []*RaidLogicalVolume{
				{Index: 2},
				{Index: 1},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReverseLogicalArray(tt.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReverseIntArray() = %v, want %v", got, tt.want)
			}
		})
	}
}
