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
	"yunion.io/x/onecloud/pkg/apis"
)

const (
	WAF_IPSET_STATUS_AVAILABLE     = "available"
	WAF_IPSET_STATUS_DELETING      = "deleting"
	WAF_IPSET_STATUS_DELETE_FAILED = "delete_failed"
)

type WafIPSetDetails struct {
	apis.StatusInfrasResourceBaseDetails
	SWafIPSet
}

type WafIPSetListInput struct {
	apis.StatusInfrasResourceBaseListInput
}

type WafIPSetCacheDetails struct {
	apis.StatusStandaloneResourceDetails
	ManagedResourceInfo
	CloudregionResourceInfo
	SWafIPSetCache
}

type WafIPSetCacheListInput struct {
	apis.StatusStandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput

	ManagedResourceListInput
	RegionalFilterListInput
}
