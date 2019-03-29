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

package models

import (
	"reflect"
	"testing"
)

func Test_diff(t *testing.T) {
	type args struct {
		origins  []string
		compares []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Test array diff",
			args: args{
				origins:  []string{"1", "2", "3"},
				compares: []string{"2", "3", "5"},
			},
			want: []string{"1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := diff(tt.args.origins, tt.args.compares); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("diff() = %v, want %v", got, tt.want)
			}
		})
	}
}
