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
	"yunion.io/x/cloudmux/pkg/cloudprovider"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	CDN_DOMAIN_STATUS_ONLINE        = compute.CDN_DOMAIN_STATUS_ONLINE
	CDN_DOMAIN_STATUS_OFFLINE       = compute.CDN_DOMAIN_STATUS_OFFLINE
	CDN_DOMAIN_STATUS_DELETING      = "deleting"
	CDN_DOMAIN_STATUS_DELETE_FAILED = "delete_failed"
	CDN_DOMAIN_STATUS_PROCESSING    = compute.CDN_DOMAIN_STATUS_PROCESSING
	CDN_DOMAIN_STATUS_REJECTED      = compute.CDN_DOMAIN_STATUS_REJECTED
	CDN_DOMAIN_STATUS_UNKNOWN       = "unknown"

	CDN_DOMAIN_AREA_MAINLAND       = compute.CDN_DOMAIN_AREA_MAINLAND
	CDN_DOMAIN_AREA_OVERSEAS       = compute.CDN_DOMAIN_AREA_OVERSEAS
	CDN_DOMAIN_AREA_GLOBAL         = compute.CDN_DOMAIN_AREA_GLOBAL
	CDN_DOMAIN_ORIGIN_TYPE_DOMAIN  = "domain"
	CDN_DOMAIN_ORIGIN_TYPE_IP      = "ip"
	CDN_DOMAIN_ORIGIN_TYPE_BUCKET  = compute.CDN_DOMAIN_ORIGIN_TYPE_BUCKET
	CDN_DOMAIN_ORIGIN_THIRED_PARTY = "third_party"

	// Qcloud
	CDN_SERVICE_TYPE_WEB      = "web"      // 静态加速
	CND_SERVICE_TYPE_DOWNLOAD = "download" // 下载加速
	CND_SERVICE_TYPE_MEDIA    = "media"    // 流媒体点播加速
)

type CdnDomain struct {
	// cdn加速域名
	Domain string
	// 状态 rejected(域名未审核)|processing(部署中)|online|offline
	Status string
	// 区域 mainland|overseas|global
	Area string
	// cdn Cname
	Cname string
	// 源站
	Origin string
	// 源站类型 domain|ip|bucket
	OriginType string
}

type CdnDomains struct {
	Data []CdnDomain `json:"data"`
}

type CDNDomainCreateInput struct {
	apis.EnabledStatusInfrasResourceBaseCreateInput

	// 源站信息
	// required: true
	Origins *cloudprovider.SCdnOrigins

	// 服务类型
	// required: true
	// enmu: web, download, media
	ServiceType string `json:"service_type"`
	// 加速区域
	// enmu: mainland, overseas, global
	// requrired: true
	Area string `json:"area"`

	CloudproviderResourceInput
	DeletePreventableCreateInput
}

type CDNDomainDetails struct {
	apis.VirtualResourceDetails
	ManagedResourceInfo
}

type CDNDomainListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput
	apis.EnabledResourceBaseListInput

	ManagedResourceListInput
}

type CDNCustomHostnameOutput struct {
	Data []cloudprovider.CustomHostname
}

type CDNDeleteCustomHostnameInput struct {
	Id string
}
