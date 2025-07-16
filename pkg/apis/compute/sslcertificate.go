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
	"yunion.io/x/cloudmux/pkg/cloudprovider"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	SSL_ISSUER_LETSENCRYPT = cloudprovider.SSL_ISSUER_LETSENCRYPT
	SSL_ISSUER_ZEROSSL     = cloudprovider.SSL_ISSUER_ZEROSSL
)

type SSLCertificateCreateInput struct {
	apis.VirtualResourceCreateInput

	Issuer    string `json:"issuer"`
	DnsZoneId string `json:"dns_zone_id"`
	Sans      string `json:"sans"`

	Province    string
	Common      string
	Country     string
	City        string
	OrgName     string
	Certificate string
	PrivateKey  string
}

type SSLCertificateDetails struct {
	apis.VirtualResourceDetails
	ManagedResourceInfo
	CloudregionResourceInfo

	// 是否过期
	IsExpired bool `json:"is_expired"`
}

type SSLCertificateListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput
	apis.DeletePreventableResourceBaseListInput

	RegionalFilterListInput
	ManagedResourceListInput
	DnsZoneFilterListBase

	IsExpired *bool `json:"is_expired"`
}
