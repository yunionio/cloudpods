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

package sqlchemy

import (
	"testing"
)

func TestConditions(t *testing.T) {
	field := &SRawQueryField{"name"}
	cond1 := Equals(field, "zone1")
	t.Logf("%s %s", cond1.WhereClause(), cond1.Variables())
	cond2 := Equals(field, "zone2")
	t.Logf("%s %s", cond2.WhereClause(), cond2.Variables())
	cond3 := OR(cond1, cond2)
	t.Logf("%s %s", cond3.WhereClause(), cond3.Variables())
	cond4 := Equals(field, "zone3")
	cond5 := AND(cond4, cond3)
	t.Logf("%s %s", cond5.WhereClause(), cond5.Variables())
	cond6 := IsFalse(field)
	cond7 := AND(cond6, cond5)
	t.Logf("%s %s", cond7.WhereClause(), cond7.Variables())
	cond8 := AND(cond5, cond7)
	t.Logf("%s %s", cond8.WhereClause(), cond8.Variables())
}

func Test_likeEscape(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test Like escape",
			args: args{
				"test%%%",
			},
			want: "test\\%\\%\\%",
		},
		{
			name: "test Like escape2",
			args: args{
				"test_%_%",
			},
			want: "test\\_\\%\\_\\%",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := likeEscape(tt.args.s); got != tt.want {
				println(len(got), len(tt.want))
				t.Errorf("likeEscape() = %v, want %v", got, tt.want)
			}
		})
	}
}
