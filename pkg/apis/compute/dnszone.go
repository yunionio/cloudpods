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
	"reflect"

	"yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	DNS_ZONE_STATUS_AVAILABLE     = compute.DNS_ZONE_STATUS_AVAILABLE // 可用
	DNS_ZONE_STATUS_CREATING      = "creating"                        // 创建中
	DNS_ZONE_STATUS_CREATE_FAILE  = "create_failed"                   // 创建失败
	DNS_ZONE_STATUS_DELETING      = "deleting"                        // 删除中
	DNS_ZONE_STATUS_DELETE_FAILED = "delete_failed"                   // 删除失败
	DNS_ZONE_STATUS_UNKNOWN       = compute.DNS_ZONE_STATUS_UNKNOWN   // 未知
)

type DnsZoneFilterListBase struct {
	DnsZoneId string `json:"dns_zone_id"`
	ManagedResourceListInput
}

type DnsZoneCreateInput struct {
	apis.SharableVirtualResourceCreateInput

	CloudproviderResourceInput

	// 区域类型
	//
	//
	// | 类型            | 说明    |
	// |----------       |---------|
	// | PublicZone      | 公有    |
	// | PrivateZone     | 私有    |
	ZoneType string `json:"zone_type"`
	// 额外参数

	// VPC id列表, 仅在zone_type为PrivateZone时生效, vpc列表必须属于同一个账号
	VpcIds []string `json:"vpc_ids"`
}

type DnsZoneDetails struct {
	apis.SharableVirtualResourceDetails
	ManagedResourceInfo
	SDnsZone

	// Dns记录数量
	DnsRecordCount int `json:"dns_record_count"`
	// 关联vpc数量
	VpcCount int `json:"vpc_count"`
}

type DnsZoneListInput struct {
	apis.SharableVirtualResourceListInput

	ManagedResourceListInput

	// 区域类型
	//
	//
	// | 类型            | 说明    |
	// |----------       |---------|
	// | PublicZone      | 公有    |
	// | PrivateZone     | 私有    |
	ZoneType string `json:"zone_type"`

	// Filter dns zone By vpc
	VpcId string `json:"vpc_id"`
}

type DnsZoneSyncStatusInput struct {
}

type DnsZoneAddVpcsInput struct {
	// VPC id列表
	VpcIds []string `json:"vpc_ids"`
}

type DnsZoneRemoveVpcsInput struct {
	// VPC id列表
	VpcIds []string `json:"vpc_ids"`
}

type DnsZonePurgeInput struct {
}

type SNameServers []string

func (ns SNameServers) String() string {
	return jsonutils.Marshal(ns).String()
}

func (ns SNameServers) IsZero() bool {
	return len(ns) == 0
}

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&SNameServers{}), func() gotypes.ISerializable {
		return &SNameServers{}
	})
}
