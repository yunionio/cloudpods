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

package system_service

import (
	"testing"
)

func TestParseSysv(t *testing.T) {
	cases := []struct {
		in1        string
		in2        string
		name       string
		wantLoaded bool
		wantActive bool
	}{
		{`openvswitch    	0:off	1:off	2:off	3:off	4:off	5:off	6:off`, `ovsdb-server is running with pid 114553
ovs-vswitchd is running with pid 114563`, "openvswitch", true, true},
		{`yunion-host    	0:off	1:off	2:on	3:on	4:on	5:on	6:off`, "", "yunion-host", true, false},
	}
	for _, c := range cases {
		status := parseSysvStatus(c.in1, c.in2, c.name)
		if status.Loaded != c.wantLoaded || status.Active != c.wantActive {
			t.Errorf("parseSysVStatus %s Loaded want %v got %v Active want %v got %v", c.name, c.wantLoaded, status.Loaded, c.wantActive, status.Active)
		}
	}
}
