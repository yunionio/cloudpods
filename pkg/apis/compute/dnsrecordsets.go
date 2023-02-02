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
	"regexp"

	"yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
)

const (
	DNS_RECORDSET_STATUS_AVAILABLE = compute.DNS_RECORDSET_STATUS_AVAILABLE
)

type DnsRecordPolicy struct {
	// 平台
	Provider      string              `json:"provider"`
	PolicyType    string              `json:"policy_type"`
	PolicyValue   string              `json:"policy_value"`
	PolicyOptions *jsonutils.JSONDict `json:"policy_options"`
}

type DnsRecordSetCreateInput struct {
	apis.EnabledStatusStandaloneResourceCreateInput

	DnsZoneId  string `json:"dns_zone_id"`
	DnsType    string `json:"dns_type"`
	DnsValue   string `json:"dns_value"`
	TTL        int64  `json:"ttl"`
	MxPriority int64  `json:"mx_priority"`

	TrafficPolicies []DnsRecordPolicy `json:"traffic_policies"`
}

type DnsRecordSetUpdateInput struct {
	apis.EnabledStatusStandaloneResourceBaseUpdateInput

	DnsType    string `json:"dns_type"`
	DnsValue   string `json:"dns_value"`
	TTL        *int64 `json:"ttl"`
	MxPriority *int64 `json:"mx_priority"`

	TrafficPolicies []DnsRecordPolicy
}

type DnsRecordSetDetails struct {
	apis.EnabledStatusStandaloneResourceDetails
	SDnsRecordSet

	TrafficPolicies []DnsRecordPolicy

	DnsZone string `json:"dns_zone"`
}

type DnsRecordSetListInput struct {
	apis.EnabledStatusStandaloneResourceListInput

	DnsZoneFilterListBase
}

type DnsRecordEnableInput struct {
	apis.PerformEnableInput
}

type DnsRecordDisableInput struct {
	apis.PerformDisableInput
}

type DnsRecordSetTrafficPoliciesInput struct {
	TrafficPolicies []DnsRecordPolicy `json:"traffic_policies"`
}

func (recordset *SDnsRecordSet) ValidateDnsrecordValue() error {
	domainReg := regexp.MustCompile(`^(([a-zA-Z]{1})|([a-zA-Z]{1}[a-zA-Z]{1})|([a-zA-Z]{1}[0-9]{1})|([0-9]{1}[a-zA-Z]{1})|([a-zA-Z0-9][a-zA-Z0-9-_]{1,61}[a-zA-Z0-9]))\.([a-zA-Z]{2,6}|[a-zA-Z0-9-]{2,30}\.[a-zA-Z]{2,3})$`)
	switch cloudprovider.TDnsType(recordset.DnsType) {
	case cloudprovider.DnsTypeMX:
		if recordset.MxPriority < 1 || recordset.MxPriority > 50 {
			return httperrors.NewOutOfRangeError("mx_priority range limited to [1,50]")
		}
		if !domainReg.MatchString(recordset.DnsValue) {
			return httperrors.NewInputParameterError("invalid domain %s for MX record", recordset.DnsValue)
		}
	case cloudprovider.DnsTypeA:
		if !regutils.MatchIP4Addr(recordset.DnsValue) {
			return httperrors.NewInputParameterError("invalid ipv4 %s for A record", recordset.DnsValue)
		}
	case cloudprovider.DnsTypeAAAA:
		if !regutils.MatchIP6Addr(recordset.DnsValue) {
			return httperrors.NewInputParameterError("invalid ipv6 %s for AAAA record", recordset.DnsValue)
		}
	case cloudprovider.DnsTypeCNAME:
		if !domainReg.MatchString(recordset.DnsValue) {
			return httperrors.NewInputParameterError("invalid domain %s for CNAME record", recordset.DnsValue)
		}
	}
	return nil
}
