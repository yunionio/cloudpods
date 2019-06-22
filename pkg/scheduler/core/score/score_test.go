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

package score

import (
	"testing"

	"yunion.io/x/pkg/tristate"
)

func TestLess(t *testing.T) {
	type args struct {
		b1       *ScoreBucket
		b2       *ScoreBucket
		lessFunc func(s1, s2 *ScoreBucket) bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "equal",
			args: args{
				b1:       NewScoreBuckets(),
				b2:       NewScoreBuckets(),
				lessFunc: NormalLess,
			},
			want: false,
		},
		{
			name: "less",
			args: args{
				b1:       NewScoreBuckets().SetScore(SScore{1, "p1"}, tristate.True),
				b2:       NewScoreBuckets().SetScore(SScore{1, "p1"}, tristate.True).SetScore(SScore{2, "p2"}, tristate.True),
				lessFunc: PreferLess,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.lessFunc(tt.args.b1, tt.args.b2); got != tt.want {
				t.Errorf("Less() = %v, want %v", got, tt.want)
			}
		})
	}
}
