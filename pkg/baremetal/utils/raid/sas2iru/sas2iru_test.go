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

package sas2iru

import "testing"

func Test_getLineAdapterIndex(t *testing.T) {
	tests := []struct {
		line string
		want int
	}{
		{
			line: "0    SAS3008    100h    97h    00h:5eh:00h:00h    1000h    0097h",
			want: 0,
		},
		{
			line: "SAS3IRCU: Utility Completed Successfully.",
			want: -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := getLineAdapterIndex(tt.line); got != tt.want {
				t.Errorf("getLineAdapterIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}
