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

package netutils2

import (
	"reflect"
	"testing"
)

func TestExpandCompactIpds(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{
			in: "192.168.100.2",
			want: []string{
				"192.168.100.2",
			},
		},
		{
			in: "192.168.100.2,9,10;10.168.200.1",
			want: []string{
				"192.168.100.2",
				"192.168.100.9",
				"192.168.100.10",
				"10.168.200.1",
			},
		},
	}
	for _, c := range cases {
		out, err := ExpandCompactIps(c.in)
		if err != nil {
			t.Errorf("expand %s error %s", c.in, err)
		} else {
			if !reflect.DeepEqual(c.want, out) {
				t.Errorf("expand %s want %s got %s", c.in, c.want, out)
			}
		}
	}
}
