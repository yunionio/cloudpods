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

package balancer

import (
	"reflect"
	"testing"
)

type floatC float64

func (f floatC) GetScore() float64 {
	return float64(f)
}

func newFCs(n ...float64) []ICandidate {
	ret := make([]ICandidate, len(n))
	for i := range n {
		ret[i] = floatC(n[i])
	}
	return ret
}

func Test_findFitCandidates(t *testing.T) {
	type args struct {
		input []ICandidate
		delta float64
	}
	tests := []struct {
		name    string
		args    args
		want    []ICandidate
		wantErr bool
	}{
		{
			name: "{}",
			args: args{
				input: newFCs(),
				delta: 3.0,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "{1, 2, 3}, 3",
			args: args{
				input: newFCs(1, 2, 3),
				delta: 3.0,
			},
			want:    newFCs(1, 2),
			wantErr: false,
		},
		{
			name: "{1, 2, 3}, 0.5",
			args: args{
				input: newFCs(1, 2, 3),
				delta: 0.5,
			},
			want:    newFCs(1),
			wantErr: false,
		},
		{
			name: "{1, 2, 3}, 4",
			args: args{
				input: newFCs(1, 2, 3),
				delta: 4,
			},
			want:    newFCs(1, 2, 3),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := findFitCandidates(tt.args.input, tt.args.delta)
			if (err != nil) != tt.wantErr {
				t.Errorf("findN() got = %v, error = %v, wantErr %v, err = %v", got, err, tt.wantErr, err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findN() = %v, want %v", got, tt.want)
			}
		})
	}
}
