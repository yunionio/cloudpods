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

const (
	CDN_DOMAIN_STATUS_ONLINE        = "online"
	CDN_DOMAIN_STATUS_OFFLINE       = "offline"
	CDN_DOMAIN_STATUS_DELETING      = "deleting"
	CDN_DOMAIN_STATUS_DELETE_FAILED = "delete_failed"
	CDN_DOMAIN_STATUS_PROCESSING    = "processing"
	CDN_DOMAIN_STATUS_REJECTED      = "rejected"
	CDN_DOMAIN_STATUS_UNKNOWN       = "unknown"

	CDN_DOMAIN_AREA_MAINLAND       = "mainland"
	CDN_DOMAIN_AREA_OVERSEAS       = "overseas"
	CDN_DOMAIN_AREA_GLOBAL         = "global"
	CDN_DOMAIN_ORIGIN_TYPE_DOMAIN  = "domain"
	CDN_DOMAIN_ORIGIN_TYPE_IP      = "ip"
	CDN_DOMAIN_ORIGIN_TYPE_BUCKET  = "bucket"
	CDN_DOMAIN_ORIGIN_THIRED_PARTY = "third_party"

	// Qcloud
	CDN_SERVICE_TYPE_WEB      = "web"      // 静态加速
	CND_SERVICE_TYPE_DOWNLOAD = "download" // 下载加速
	CND_SERVICE_TYPE_MEDIA    = "media"    // 流媒体点播加速
)
