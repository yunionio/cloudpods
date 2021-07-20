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

package ovn

import (
	apis "yunion.io/x/onecloud/pkg/apis/compute"
	agentmodels "yunion.io/x/onecloud/pkg/vpcagent/models"
)

func vpcHasDistgw(vpc *agentmodels.Vpc) bool {
	mode := vpc.ExternalAccessMode
	switch mode {
	case
		apis.VPC_EXTERNAL_ACCESS_MODE_DISTGW,
		apis.VPC_EXTERNAL_ACCESS_MODE_EIP_DISTGW:
		return true
	default:
		return false
	}
}

func vpcHasEipgw(vpc *agentmodels.Vpc) bool {
	mode := vpc.ExternalAccessMode
	switch mode {
	case
		apis.VPC_EXTERNAL_ACCESS_MODE_EIP,
		apis.VPC_EXTERNAL_ACCESS_MODE_EIP_DISTGW:
		return true
	default:
		return false
	}
}
