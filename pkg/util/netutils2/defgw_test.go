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
	"testing"
)

func TestFindDefaultNic(t *testing.T) {
	cases := []struct {
		nics  [][]string
		mac   string
		index int
	}{
		{
			nics: [][]string{
				{"192.168.202.147", "00:22:00:00:00:01", "192.168.202.1", "", "", "false"},
			},
			mac:   "00:22:00:00:00:01",
			index: 0,
		},
		{
			nics: [][]string{
				{"192.168.202.147", "00:22:00:00:00:01", "192.168.202.1", "", "", "false"},
				{"192.168.203.147", "00:22:00:00:00:02", "192.168.203.1", "", "", "false"},
			},
			mac:   "00:22:00:00:00:01",
			index: 0,
		},
		{
			nics: [][]string{
				{"192.168.202.147", "00:22:00:00:00:01", "192.168.202.1", "", "", "false"},
				{"192.168.203.147", "00:22:00:00:00:02", "192.168.203.1", "", "", "true"},
			},
			mac:   "00:22:00:00:00:02",
			index: 1,
		},
		{
			nics: [][]string{
				{"", "00:22:00:00:00:01", "", "3ffe:3200:fe::fb", "3ffe:3200:fe::1", "false"},
				{"", "00:22:00:00:00:02", "", "3ffe:3200:ff::fb", "3ffe:3200:ff::1", "true"},
			},
			mac:   "00:22:00:00:00:02",
			index: 1,
		},
		{
			nics: [][]string{
				{"", "00:22:00:00:00:01", "", "3ffe:3200:fe::fb", "3ffe:3200:fe::1", "false"},
				{"", "00:22:00:00:00:02", "", "3ffe:3200:ff::fb", "", "true"},
			},
			mac:   "00:22:00:00:00:01",
			index: 0,
		},
		{
			nics: [][]string{
				{"192.168.202.147", "00:22:00:00:00:01", "192.168.202.1", "", "", "false"},
				{"202.168.202.147", "00:22:00:00:00:02", "202.168.202.1", "", "", "false"},
			},
			mac:   "00:22:00:00:00:02",
			index: 1,
		},
		{
			nics: [][]string{
				{"202.168.202.147", "00:22:00:00:00:01", "202.168.202.1", "", "", "false"},
				{"192.168.202.147", "00:22:00:00:00:02", "192.168.202.1", "", "", "false"},
			},
			mac:   "00:22:00:00:00:01",
			index: 0,
		},
		{
			nics: [][]string{
				{"192.168.202.147", "00:22:00:00:00:01", "192.168.202.1", "", "", "false"},
				{"", "00:22:00:00:00:02", "", "", "", "false"},
				{"202.168.202.147", "00:22:00:00:00:03", "202.168.202.1", "", "", "false"},
			},
			mac:   "00:22:00:00:00:03",
			index: 2,
		},
	}
	for _, c := range cases {
		nics := SNicInfoList{}
		for _, n := range c.nics {
			nics = nics.Add(n[1], n[0], n[2], n[3], n[4], n[5] == "true")
		}

		gotMac, gotIdx := nics.FindDefaultNicMac()
		if gotMac != c.mac || gotIdx != c.index {
			t.Errorf("want %s (%d) got %s (%d)", c.mac, c.index, gotMac, gotIdx)
		}
	}
}
