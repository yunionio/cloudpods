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

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	DNS_RECORDSET_STATUS_AVAILABLE = "available"
)

type DnsRecordPolicy struct {
	Provider      string              `json:"provider"`
	PolicyType    string              `json:"policy_type"`
	PolicyValue   string              `json:"policy_value"`
	PolicyOptions *jsonutils.JSONDict `json:"policy_options"`
}

type DnsRecordSetCreateInput struct {
	apis.EnabledStatusStandaloneResourceCreateInput

	DnsZoneId string `json:"dns_zone_id"`
	DnsType   string `json:"dns_type"`
	DnsValue  string `json:"dns_value"`
	TTL       int64  `json:"ttl"`

	TrafficPolicies []DnsRecordPolicy `json:"traffic_policies"`
}

type DnsRecordSetDetails struct {
	apis.EnabledStatusStandaloneResourceDetails
	SDnsRecordSet
	TrafficPolicies []DnsRecordPolicy
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
