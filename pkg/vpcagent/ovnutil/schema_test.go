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

package ovnutil

import (
	"reflect"
	"testing"
)

func TestUnmarshal(t *testing.T) {
	d := `{"data":[[["uuid","281f3163-0430-4626-8b94-eb4e79e6d85a"],["set",[]],["set",[]],["map",[["oc-vpc-id","uuididid"]]],["set",[]],"ls0",["map",[["subnet","192.168.2.0/24"]]],["set",[]],["set",[]]]],"headings":["_uuid","acls","dns_records","external_ids","load_balancer","name","other_config","ports","qos_rules"]}`
	got := &LogicalSwitchTable{}
	if err := UnmarshalJSON([]byte(d), got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := &LogicalSwitchTable{
		LogicalSwitch{
			Uuid:        "281f3163-0430-4626-8b94-eb4e79e6d85a",
			Name:        "ls0",
			ExternalIds: map[string]string{"oc-vpc-id": "uuididid"},
			OtherConfig: map[string]string{"subnet": "192.168.2.0/24"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("\n got %#v\nwant %#v", got, want)
	}

	t.Run("logical switch", func(t *testing.T) {
		d := `{"data":[[["uuid","d9dbe3cc-7efe-4cf2-91f0-f8f2143037f1"],"32:38:30:34:61:64 169.254.169.3",["set",[]],["set",[]],["set",[]],["set",[]],["map",[]],"subnet-lsmp-subnet3",["map",[]],["set",[]],["set",[]],["set",[]],["set",[]],"localport",false],[["uuid","90fd42c1-ca9c-4fb5-a37a-a950faef4d19"],"router",["set",[]],["set",[]],["set",[]],["set",[]],["map",[]],"subnet-lsp-subnet3",["map",[["router-port","subnet-lrp-subnet3"]]],["set",[]],["set",[]],["set",[]],["set",[]],"router",true],[["uuid","f8fefe76-083b-405c-ae1e-1e70eab9b903"],"00:33:00:00:00:09 192.168.3.9",["uuid","34466b0c-038d-44df-b444-f127d19ef188"],["set",[]],["set",[]],["set",[]],["map",[]],"iface-p39",["map",[]],["set",[]],"00:33:00:00:00:09 192.168.3.9/24",["set",[]],["set",[]],"",true],[["uuid","e514e6da-080b-46b8-9822-d62596c28918"],"router",["set",[]],["set",[]],["set",[]],["set",[]],["map",[]],"subnet-lsp-subnet2",["map",[["router-port","subnet-lrp-subnet2"]]],["set",[]],["set",[]],["set",[]],["set",[]],"router",true],[["uuid","804d671e-20d4-4ad3-82e8-735463dfed09"],"00:22:00:00:00:04 192.168.2.4",["uuid","2cd3dfeb-c521-4e88-9ffb-c89006f8f92b"],["set",[]],["set",[]],["set",[]],["map",[]],"iface-p24",["map",[]],["set",[]],"00:22:00:00:00:04 192.168.2.4/24",["set",[]],["set",[]],"",true],[["uuid","91742362-93b9-4d06-92df-d5e0e2c28dc3"],"00:22:00:00:00:03 192.168.2.3",["uuid","2cd3dfeb-c521-4e88-9ffb-c89006f8f92b"],["set",[]],["set",[]],["set",[]],["map",[]],"iface-p23",["map",[]],["set",[]],"00:22:00:00:00:03 192.168.2.3/24",["set",[]],["set",[]],"",true],[["uuid","91d9efcf-fba3-4b5b-adaa-4c64948f8824"],"32:31:36:37:62:38 169.254.169.2",["set",[]],["set",[]],["set",[]],["set",[]],["map",[]],"subnet-lsmp-subnet2",["map",[]],["set",[]],["set",[]],["set",[]],["set",[]],"localport",false]],"headings":["_uuid","addresses","dhcpv4_options","dhcpv6_options","dynamic_addresses","enabled","external_ids","name","options","parent_name","port_security","tag","tag_request","type","up"]}`
		got := &LogicalSwitchPortTable{}
		if err := UnmarshalJSON([]byte(d), got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
	})
}
