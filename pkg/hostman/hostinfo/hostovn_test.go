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

package hostinfo

import (
	"testing"
)

func TestMustGetOvnVersion(t *testing.T) {
	cases := []struct {
		in  string
		out string
	}{
		{
			in: `
ovn-controller (Open vSwitch) 2.9
OpenFlow versions 0x4:0x4
`,
			out: "2.9",
		},
		{
			in: `
ovn-controller (Open vSwitch) 2.9.6
OpenFlow versions 0x4:0x4
`,
			out: "2.9.6",
		},
		{
			in: `
ovn-controller (Open vSwitch) 2.9.100
OpenFlow versions 0x4:0x4
`,
			out: "2.9.100",
		},
		{
			in: `
ovn-controller (Open vSwitch) 2.9.1000
OpenFlow versions 0x4:0x4
`,
			out: "",
		},
		{
			in: `
ovn-controller (Open vSwitch) 2.9.6.1
`,
			out: "",
		},
	}
	for _, c := range cases {
		got := ovnExtractVersion(c.in)
		if got != c.out {
			t.Fatalf("got: %s, want: %s, input:\n%s", got, c.out, c.in)
		}
	}
}
