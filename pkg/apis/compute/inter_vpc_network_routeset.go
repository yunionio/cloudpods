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

import "yunion.io/x/onecloud/pkg/apis"

const (
	INTER_VPCNETWORK_ATTACHED_INSTAMCE_TYPE_VPC = "VPC"
	INTER_VPCNETWORK_ATTACHED_INSTAMCE_TYPE_VBR = "VBR" // 边界路由器
)

type InterVpcNetworkRouteSetEnableInput struct {
	apis.PerformEnableInput
}

type InterVpcNetworkRouteSetDisableInput struct {
	apis.PerformDisableInput
}

type InterVpcNetworkRouteSetDetails struct {
	apis.EnabledStatusStandaloneResourceDetails
	VpcResourceInfo
}

type InterVpcNetworkRouteSetListInput struct {
	apis.EnabledStatusStandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput
	VpcFilterListInput
	InterVpcNetworkId string
	Cidr              string `json:"cidr"`
}
