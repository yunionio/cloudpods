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

package esxi

// TODO: rewrite this test
/*
import "testing"

func TestNewClient(t *testing.T) {
	cli, err := NewESXiClient("", "", "10.168.222.104", 443, "root", "123@Vmware")

	if err != nil {
		t.Errorf("%s", err)
	} else {
		t.Logf("%s", cli.About())

		cli.fetchDatacenters()

		host, err := cli.FindHostByIp("10.168.222.104")
		if err != nil {
			t.Errorf("find_host_by_ip %s", err)
		} else {
			t.Logf("host %s %s", host.GetAccessIp(), host.GetName())
		}
	}
}*/
