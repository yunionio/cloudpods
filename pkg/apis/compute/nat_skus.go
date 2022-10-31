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
	"yunion.io/x/cloudmux/pkg/apis/compute"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	NAT_SKU_AVAILABLE = compute.NAT_SKU_AVAILABLE
	NAT_SKU_SOLDOUT   = compute.NAT_SKU_SOLDOUT

	ALIYUN_NAT_SKU_DEFAULT = compute.ALIYUN_NAT_SKU_DEFAULT
)

type NatSkuListInput struct {
	apis.EnabledStatusStandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput

	RegionalFilterListInput

	PostpaidStatus string `json:"postpaid_stauts"`
	PrepaidStatus  string `json:"prepaid_status"`

	Providers []string `json:"providers"`
	// swagger:ignore
	// Deprecated
	Provider []string `json:"provider" yunion-deprecated-by:"providers"`
}

type NatSkuDetails struct {
	apis.EnabledStatusStandaloneResourceDetails

	CloudregionResourceInfo

	// 云环境
	CloudEnv string `json:"cloud_env"`
}
