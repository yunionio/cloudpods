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

var (
	Servernetworks JointResourceManager
)

func init() {
	Servernetworks = NewJointComputeManager(
		"guestnetwork",
		"guestnetworks",
		[]string{"Guest_ID", "Guest",
			"Network_ID", "Network", "Mac_addr",
			"IP_addr", "Driver", "BW_limit", "Index",
			"Virtual", "Ifname", "team_with"},
		[]string{},
		&Servers,
		&Networks)
	registerCompute(&Servernetworks)
}
