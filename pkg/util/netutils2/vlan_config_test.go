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

	"yunion.io/x/jsonutils"
)

func TestParseVlanConfig(t *testing.T) {
	const content = `VLAN Dev name	 | VLAN ID
	Name-Type: VLAN_NAME_TYPE_RAW_PLUS_VID_NO_PAD
	eth0.2048      | 2048  | eth0`
	conf, err := parseVlanConfigContent(content)
	if err != nil {
		t.Errorf("parseVlanConfig error %s", err)
	} else {
		t.Logf("%s", jsonutils.Marshal(conf))
	}
}
