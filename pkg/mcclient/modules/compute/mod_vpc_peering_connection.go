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

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	VpcPeeringConnections modulebase.ResourceManager
)

func init() {
	VpcPeeringConnections = modules.NewComputeManager("vpc_peering_connection", "vpc_peering_connections",
		[]string{"ID", "Name", "Enabled", "Status", "vpc_id", "peer_vpc_id", "peer_account_id", "Public_Scope", "Domain_Id", "Domain"},
		[]string{})

	modules.RegisterCompute(&VpcPeeringConnections)
}
