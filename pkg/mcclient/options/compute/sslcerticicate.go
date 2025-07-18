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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type SslCertificateListOptions struct {
	options.BaseListOptions
}

func (opts *SslCertificateListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type SslCertificateCreateOptions struct {
	options.BaseCreateOptions

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

func (opts *SslCertificateCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}
