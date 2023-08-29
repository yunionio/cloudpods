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

type DnsRecordListOptions struct {
	options.BaseListOptions
	DnsZoneId string `help:"DnsZone Id or Name"`
}

func (opts *DnsRecordListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type DnsRecordCreateOptions struct {
	options.EnabledStatusCreateOptions
	DNS_ZONE_ID string `help:"Dns Zone Id"`
	DNS_TYPE    string `choices:"A|AAAA|CAA|CNAME|MX|NS|SRV|SOA|TXT|PTR|DS|DNSKEY|IPSECKEY|NAPTR|SPF|SSHFP|TLSA|REDIRECT_URL|FORWARD_URL"`
	DNS_VALUE   string `help:"Dns Value"`
	Ttl         int64  `help:"Dns ttl" default:"300"`
	MxPriority  int64  `help:"dns mx type mxpriority"`
	PolicyType  string `choices:"Simple|ByCarrier|ByGeoLocation|BySearchEngine|IpRange|Weighted|Failover|MultiValueAnswer|Latency"`
	PolicyValue string `help:"Dns Traffic policy value"`
}

func (opts *DnsRecordCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts).(*jsonutils.JSONDict)
	return params, nil
}

type DnsRecordIdOptions struct {
	ID string
}

func (opts *DnsRecordIdOptions) GetId() string {
	return opts.ID
}

func (opts *DnsRecordIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type DnsRecordUpdateOptions struct {
	options.BaseUpdateOptions
	DnsType     string `choices:"A|AAAA|CAA|CNAME|MX|NS|SRV|SOA|TXT|PRT|DS|DNSKEY|IPSECKEY|NAPTR|SPF|SSHFP|TLSA|REDIRECT_URL|FORWARD_URL"`
	DnsValue    string
	Ttl         *int64
	MxPriority  *int64 `help:"dns mx type mxpriority"`
	PolicyType  string `choices:"Simple|ByCarrier|ByGeoLocation|BySearchEngine|IpRange|Weighted|Failover|MultiValueAnswer|Latency"`
	PolicyValue string `help:"Dns Traffic policy value"`
}

func (opts *DnsRecordUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts).(*jsonutils.JSONDict)
	params.Remove("id")
	return params, nil
}
