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

package ansiblev2

import (
	"testing"
)

func TestInventory(t *testing.T) {
	inv := NewInventory(
		"ansible_user", "root",
		"ansible_become", "yes",
		"redis_bind_address", "0.0.0.0",
		"redis_bind_port", 9736,
	)
	inv.SetHost("redis", NewHost(
		"ansible_host", "192.168.0.248",
		"ansible_port", 2222,
	))
	inv.SetHost("nameonly", NewHost())
	child := NewHostGroup()
	child.SetHost("aws-bj01", NewHost(
		"ansible_user", "ec2-user",
		"ansible_host", "1.2.2.3",
	))
	child.SetHost("ali-hk01", NewHost(
		"ansible_user", "cloudroot",
		"ansible_host", "1.2.2.4",
	))
	inv.SetChild("netmon-agents", child)
	t.Logf("\n%s", inv.String())
}
