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

package compute

// IP: 20
// UDP: 8
// GENEVE HDR: 8 + 4x
// total: 36 + 4x
const VPC_OVN_ENCAP_COST = 60

const (
	VPC_EXTERNAL_ACCESS_MODE_DISTGW     = "distgw"     // distgw only
	VPC_EXTERNAL_ACCESS_MODE_EIP_DISTGW = "eip-distgw" // eip when available, distgw otherwise
	VPC_EXTERNAL_ACCESS_MODE_EIP        = "eip"        // eip only
	VPC_EXTERNAL_ACCESS_MODE_NONE       = "none"       // no external access
)

var (
	VPC_EXTERNAL_ACCESS_MODES = []string{
		VPC_EXTERNAL_ACCESS_MODE_DISTGW,
		VPC_EXTERNAL_ACCESS_MODE_EIP_DISTGW,
		VPC_EXTERNAL_ACCESS_MODE_EIP,
		VPC_EXTERNAL_ACCESS_MODE_NONE,
	}
)
