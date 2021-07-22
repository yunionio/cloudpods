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
	KAFKA_STATUS_AVAILABLE     = "available"
	KAFKA_STATUS_UNAVAILABLE   = "unavailable"
	KAFKA_STATUS_CREATING      = "creating"
	KAFKA_STATUS_DELETING      = "deleting"
	KAFKA_STATUS_DELETE_FAILED = "delete_failed"
	KAFKA_STATUS_UNKNOWN       = "unknown"
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
}

// 资源列表请求参数
type KafkaListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput
	apis.DeletePreventableResourceBaseListInput

	RegionalFilterListInput
	ManagedResourceListInput
	VpcFilterListInput
}
