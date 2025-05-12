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
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
)

const (
	DNS_RECORDSET_STATUS_AVAILABLE = compute.DNS_RECORDSET_STATUS_AVAILABLE
	DNS_RECORDSET_STATUS_CREATING  = apis.STATUS_CREATING
)

type DnsRecordCreateInput struct {
	apis.EnabledStatusStandaloneResourceCreateInput

	DnsZoneId  string `json:"dns_zone_id"`
	DnsType    string `json:"dns_type"`
	DnsValue   string `json:"dns_value"`
	TTL        int64  `json:"ttl"`
	MxPriority int64  `json:"mx_priority"`
	Proxied    *bool  `json:"proxied"`

	PolicyType  string `json:"policy_type"`
	PolicyValue string `json:"policy_value"`
}

type DnsRecordUpdateInput struct {
	apis.EnabledStatusStandaloneResourceBaseUpdateInput

	DnsType    string `json:"dns_type"`
	DnsValue   string `json:"dns_value"`
	TTL        *int64 `json:"ttl"`
	MxPriority *int64 `json:"mx_priority"`
	Proxied    *bool  `json:"proxied"`
}

type DnsRecordDetails struct {
	apis.EnabledStatusStandaloneResourceDetails
	SDnsRecord

	DnsZone string `json:"dns_zone"`
}

type DnsRecordListInput struct {
	apis.EnabledStatusStandaloneResourceListInput

	DnsZoneFilterListBase
}

type DnsRecordEnableInput struct {
	apis.PerformEnableInput
}

type DnsRecordDisableInput struct {
	apis.PerformDisableInput
}

func (record *SDnsRecord) ValidateDnsrecordValue() error {
	switch cloudprovider.TDnsType(record.DnsType) {
	case cloudprovider.DnsTypeMX:
		if record.MxPriority < 1 || record.MxPriority > 50 {
			return httperrors.NewOutOfRangeError("mx_priority range limited to [1,50]")
		}
		if !regutils.MatchDomainName(record.DnsValue) {
			return httperrors.NewInputParameterError("invalid domain %s for MX record", record.DnsValue)
		}
	case cloudprovider.DnsTypeA:
		if !regutils.MatchIP4Addr(record.DnsValue) {
			return httperrors.NewInputParameterError("invalid ipv4 %s for A record", record.DnsValue)
		}
	case cloudprovider.DnsTypeAAAA:
		if !regutils.MatchIP6Addr(record.DnsValue) {
			return httperrors.NewInputParameterError("invalid ipv6 %s for AAAA record", record.DnsValue)
		}
	case cloudprovider.DnsTypeCNAME:
		if !regutils.MatchDomainName(record.DnsValue) {
			return httperrors.NewInputParameterError("invalid domain %s for CNAME record", record.DnsValue)
		}
	}
	return nil
}

type SDnsResolveResult struct {
	DnsValue string `json:"dns_value"`
	TTL      int64  `json:"ttl"`
	DnsName  string `json:"dns_name"`
}
