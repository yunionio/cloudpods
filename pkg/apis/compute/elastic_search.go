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
	ELASTIC_SEARCH_STATUS_AVAILABLE     = compute.ELASTIC_SEARCH_STATUS_AVAILABLE
	ELASTIC_SEARCH_STATUS_UNAVAILABLE   = compute.ELASTIC_SEARCH_STATUS_UNAVAILABLE
	ELASITC_SEARCH_STATUS_CREATING      = compute.ELASITC_SEARCH_STATUS_CREATING
	ELASTIC_SEARCH_STATUS_DELETING      = compute.ELASTIC_SEARCH_STATUS_DELETING
	ELASTIC_SEARCH_STATUS_DELETE_FAILED = "delete_failed"
	ELASTIC_SEARCH_STATUS_UNKNOWN       = "unknown"
)

const (
	ELASTIC_SEARCH_UPDATE_TAGS        = "update_tags"
	ELASTIC_SEARCH_UPDATE_TAGS_FAILED = "update_tags_fail"
)

// 资源创建参数, 目前仅占位
type ElasticSearchCreateInput struct {
}

// 资源返回详情
type ElasticSearchDetails struct {
	apis.VirtualResourceDetails
	ManagedResourceInfo
	CloudregionResourceInfo
	VpcResourceInfoBase
	NetworkResourceInfoBase
	ZoneResourceInfoBase
}

// 资源列表请求参数
type ElasticSearchListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput
	apis.DeletePreventableResourceBaseListInput

	RegionalFilterListInput
	ManagedResourceListInput
	VpcFilterListInput
}

type ElasticSearchAccessInfoInput struct {
}
