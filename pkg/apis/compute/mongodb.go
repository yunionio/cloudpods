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

type MongoDBCreateInput struct {
	apis.VirtualResourceCreateInput
	DeletePreventableCreateInput
}

type SMongoDBChangeConfigInput struct {
	apis.Meta

	InstanceType string
	DiskSizeGB   int
}

type MongoDBListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput
	apis.DeletePreventableResourceBaseListInput

	VpcFilterListInput

	ZoneResourceInput

	VcpuCount int `json:"vcpu_count"`

	VmemSizeMb int `json:"vmem_size_mb"`

	Category string `json:"category"`

	Engine string `json:"engine"`

	EngineVersion string `json:"engine_version"`

	InstanceType string `json:"instance_type"`
}

type MongoDBDetails struct {
	apis.VirtualResourceDetails
	CloudregionResourceInfo
	ZoneResourceInfoBase
	ManagedResourceInfo

	VpcResourceInfoBase

	// IP子网名称
	// example: test-network
	Network string `json:"network"`
}

type MongoDBResourceInfoBase struct {
	// MongoDB实例名称
	MongoDB string `json:"mongodb"`
}

type MongoDBResourceInfo struct {
	MongoDBResourceInfoBase

	// 归属VPC ID
	VpcId string `json:"vpc_id"`

	VpcResourceInfo
}

type MongoDBResourceInput struct {
	// MongoDB实例(ID or Name)
	MongoDBId string `json:"mongodb_id"`

	// swagger:ignore
	// Deprecated
	MongoDB string `json:"mongodb" yunion-deprecated-by:"mongodb_id"`
}

type MongoDBFilterListInputBase struct {
	MongoDBResourceInput

	// 以MongoDB实例名字排序
	OrderByMongoDB string `json:"order_by_mongodb"`
}

type MongoDBFilterListInput struct {
	MongoDBFilterListInputBase

	VpcFilterListInput
}

type MongoDBJoinListInput struct {
	apis.VirtualJointResourceBaseListInput
	MongoDBFilterListInput
}

type MongoDBRemoteUpdateInput struct {
	// 是否覆盖替换所有标签
	ReplaceTags *bool `json:"replace_tags" help:"replace all remote tags"`
}

type MongoDBNetworkListInput struct {
	MongoDBJoinListInput

	NetworkFilterListInput
}

type MongoDBAutoRenewInput struct {
	// 是否自动续费
	AutoRenew bool `json:"auto_renew"`
}

type MongoDBSetSecgroupInput struct {
	SecgroupIds []string `json:"secgroup_ids"`
}
