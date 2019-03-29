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
)

func TestScoreBucket_String(t *testing.T) {
	type fields struct {
		scores *Scores
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "EmptyScores",
			fields: fields{newScores()},
			want:   "",
		},
		{
			name: "Scores100",
			fields: fields{newScores().Append(
				NewMidScore("mid"),
				NewZeroScore(),
				NewZeroScore(),
			)},
			want: "mid: 1, zero: 0, zero: 0",
		},
		{
			name: "Scores201-1",
			fields: fields{newScores().Append(
				NewMaxScore("max"),
				NewZeroScore(),
				NewMidScore("mid"),
				NewMinScore("min"),
			)},
			want: "max: 2, zero: 0, mid: 1, min: -1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &ScoreBucket{
				scores: tt.fields.scores,
			}
			if got := b.String(); got != tt.want {
				t.Errorf("ScoreBucket.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLess(t *testing.T) {
	type args struct {
		b1 *ScoreBucket
		b2 *ScoreBucket
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "equal",
			args: args{
				b1: NewScoreBuckets(),
				b2: NewScoreBuckets(),
			},
			want: false,
		},
		{
			name: "extendEqual",
			args: args{
				b1: NewScoreBuckets().Append(
					NewZeroScore(), NewMidScore("1"),
				),
				b2: NewScoreBuckets().Append(NewMidScore("1")),
			},
			want: false,
		},
		{
			name: "10<100",
			args: args{
				b1: NewScoreBuckets().Append(
					NewMidScore("1"),
					NewZeroScore(),
				),
				b2: NewScoreBuckets().Append(
					NewMidScore("1"),
					NewZeroScore(),
					NewZeroScore(),
				),
			},
			want: true,
		},
		{
			name: "101>10",
			args: args{
				b1: NewScoreBuckets().Append(
					NewMidScore("1"),
					NewZeroScore(),
					NewMidScore("1"),
				),
				b2: NewScoreBuckets().Append(
					NewMidScore("1"),
					NewZeroScore(),
				),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Less(tt.args.b1, tt.args.b2); got != tt.want {
				t.Errorf("Less() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScoreBucket_DigitString(t *testing.T) {
	type fields struct {
		scores *Scores
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "2-101",
			fields: fields{newScores().Append(
				NewMaxScore(""),
				NewMinScore(""),
				NewZeroScore(),
				NewMidScore(""),
			)},
			want: "2-101",
		},
		{
			name:   "empty",
			fields: fields{newScores()},
			want:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &ScoreBucket{
				scores: tt.fields.scores,
			}
			if got := b.DigitString(); got != tt.want {
				t.Errorf("ScoreBucket.DigitString() = %v, want %v", got, tt.want)
			}
		})
	}
}
