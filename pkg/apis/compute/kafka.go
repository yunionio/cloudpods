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
	KAFKA_STATUS_AVAILABLE     = compute.KAFKA_STATUS_AVAILABLE
	KAFKA_STATUS_UNAVAILABLE   = compute.KAFKA_STATUS_UNAVAILABLE
	KAFKA_STATUS_CREATING      = compute.KAFKA_STATUS_CREATING
	KAFKA_STATUS_DELETING      = compute.KAFKA_STATUS_DELETING
	KAFKA_STATUS_DELETE_FAILED = "delete_failed"
	KAFKA_STATUS_UNKNOWN       = compute.KAFKA_STATUS_UNKNOWN
	KAFKA_UPDATE_TAGS          = "update_tags"
	KAFKA_UPDATE_TAGS_FAILED   = "update_tags_fail"
)

type KafkaCreateInput struct {
}

// 资源返回详情
type KafkaDetails struct {
	apis.VirtualResourceDetails
	ManagedResourceInfo
	CloudregionResourceInfo
	VpcResourceInfoBase
	NetworkResourceInfoBase
	ZoneResourceInfoBase

	SKafka
}

// 资源列表请求参数
type KafkaListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput
	apis.DeletePreventableResourceBaseListInput

	VpcFilterListInput
}
