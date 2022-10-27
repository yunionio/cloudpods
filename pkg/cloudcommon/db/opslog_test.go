// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package db

import (
	"testing"

	"yunion.io/x/pkg/util/timeutils"
)

func TestCurrentTimestamp(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{
			in:   "2021-12-09 11:01:02",
			want: 20211209110102000,
		},
	}
	for _, c := range cases {
		tm, err := timeutils.ParseTimeStr(c.in)
		if err != nil {
			t.Errorf("parseTime %s error %s", c.in, err)
		} else {
			got := CurrentTimestamp(tm)
			if got != c.want {
				t.Errorf("currentTimestamp %s want %d got %d", c.in, c.want, got)
			}
		}
	}
}
