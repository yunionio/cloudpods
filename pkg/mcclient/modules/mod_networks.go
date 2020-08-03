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

package modules

import "yunion.io/x/onecloud/pkg/mcclient/modulebase"

var (
	Networks modulebase.ResourceManager
)

func init() {
	Networks = NewComputeManager("network", "networks",
		[]string{"ID", "Name", "Guest_ip_start", "zone", "zone_id",
			"Guest_ip_end", "Guest_ip_mask",
			"wire_id", "wire", "is_public", "public_scope", "exit", "Ports",
			"vnics", "guest_gateway",
			"group_vnics", "bm_vnics", "reserve_vnics", "lb_vnics",
			"server_type", "Status", "is_auto_alloc"},
		[]string{})

	registerCompute(&Networks)
}
