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

package regutils2

import (
	"reflect"
	"testing"
)

func TestSubGroupMatch(t *testing.T) {
	type args struct {
		pattern string
		line    string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "normalInput",
			args: args{
				pattern: `(?P<idx>\d+)\s+(?P<start>\d+)s\s+(?P<end>\d+)s\s+(?P<count>\d+)s`,
				line:    `1      2048s       314984447s  314982400s  ntfs            Basic data partition  msftdata`,
			},
			want: map[string]string{
				"idx":   "1",
				"start": "2048",
				"end":   "314984447",
				"count": "314982400",
			},
		},
		{
			name: "emptyInput",
			args: args{
				pattern: `%s+`,
				line:    ``,
			},
			want: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SubGroupMatch(tt.args.pattern, tt.args.line); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SubGroupMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}
