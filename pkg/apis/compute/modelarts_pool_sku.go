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

type ModelartsPoolSkuDetails struct {
	apis.EnabledStatusStandaloneResourceDetails
	ManagedResourceInfo
	CloudregionResourceInfo
}

const (
	MODELARTS_POOL_SKU_AVAILABLE = compute.MODELARTS_POOL_SKU_AVAILABLE
	MODELARTS_POOL_SKU_SOLDOUT   = compute.MODELARTS_POOL_SKU_SOLDOUT
)

type ModelartsPoolSkuListInput struct {
	apis.EnabledStatusStandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput

	RegionalFilterListInput
	ProcessorType string
	CpuArch       string
}
