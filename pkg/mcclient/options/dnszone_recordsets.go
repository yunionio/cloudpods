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

package options

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type DnsRecordSetListOptions struct {
	BaseListOptions
	DnsZoneId string `help:"DnsZone Id or Name"`
}

func (opts *DnsRecordSetListOptions) Params() (jsonutils.JSONObject, error) {
	return ListStructToParams(opts)
}

type DnsRecordSetCreateOptions struct {
	EnabledStatusCreateOptions
	DNS_ZONE_ID   string `help:"Dns Zone Id"`
	DNS_TYPE      string `choices:"A|AAAA|CAA|CNAME|MX|NS|SRV|SOA|TXT|PTR|DS|DNSKEY|IPSECKEY|NAPTR|SPF|SSHFP|TLSA|REDIRECT_URL|FORWARD_URL"`
	DNS_VALUE     string `help:"Dns Value"`
	TTL           int64  `help:"Dns ttl"`
	MxPriority    int64  `help:"dns mx type mxpriority"`
	Provider      string `help:"Dns triffic policy provider" choices:"Aws|Qcloud|Aliyun"`
	PolicyType    string `choices:"Simple|ByCarrier|ByGeoLocation|BySearchEngine|IpRange|Weighted|Failover|MultiValueAnswer|Latency"`
	PolicyValue   string `help:"Dns Traffic policy value"`
	PolicyOptions string
}

func (opts *DnsRecordSetCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts).(*jsonutils.JSONDict)
	params.Remove("policy_type")
	params.Remove("policy_options")
	if len(opts.PolicyType) > 0 && len(opts.Provider) > 0 {
		policies := jsonutils.NewArray()
		policy := jsonutils.NewDict()
		policy.Add(jsonutils.NewString(opts.PolicyType), "policy_type")
		policy.Add(jsonutils.NewString(opts.Provider), "provider")
		policy.Add(jsonutils.NewString(opts.PolicyValue), "policy_value")
		if len(opts.PolicyOptions) > 0 {
			policyParams, err := jsonutils.Parse([]byte(opts.PolicyOptions))
			if err != nil {
				return nil, errors.Wrapf(err, "jsonutils.Parse(%s)", opts.PolicyOptions)
			}
			policy.Add(policyParams, "policy_options")
		}
		policies.Add(policy)
		params.Add(policies, "traffic_policies")
	}
	return params, nil
}

type DnsRecordSetIdOptions struct {
	ID string
}

func (opts *DnsRecordSetIdOptions) GetId() string {
	return opts.ID
}

func (opts *DnsRecordSetIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type DnsRecordSetUpdateOptions struct {
	BaseUpdateOptions
	DnsType       string `choices:"A|AAAA|CAA|CNAME|MX|NS|SRV|SOA|TXT|PRT|DS|DNSKEY|IPSECKEY|NAPTR|SPF|SSHFP|TLSA|REDIRECT_URL|FORWARD_URL"`
	DnsValue      string
	Ttl           int64
	MxPriority    int64  `help:"dns mx type mxpriority"`
	Provider      string `help:"Dns triffic policy provider" choices:"Aws|Qcloud|Aliyun"`
	PolicyType    string `choices:"Simple|ByCarrier|ByGeoLocation|BySearchEngine|IpRange|Weighted|Failover|MultiValueAnswer|Latency"`
	PolicyValue   string `help:"Dns Traffic policy value"`
	PolicyOptions string
}

func (opts *DnsRecordSetUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts).(*jsonutils.JSONDict)
	params.Remove("id")
	params.Remove("policy_type")
	params.Remove("policy_options")
	if len(opts.PolicyType) > 0 && len(opts.Provider) > 0 {
		policies := jsonutils.NewArray()
		policy := jsonutils.NewDict()
		policy.Add(jsonutils.NewString(opts.PolicyType), "policy_type")
		policy.Add(jsonutils.NewString(opts.Provider), "provider")
		policy.Add(jsonutils.NewString(opts.PolicyValue), "policy_value")
		if len(opts.PolicyOptions) > 0 {
			policyParams, err := jsonutils.Parse([]byte(opts.PolicyOptions))
			if err != nil {
				return nil, errors.Wrapf(err, "jsonutils.Parse(%s)", opts.PolicyOptions)
			}
			policy.Add(policyParams, "policy_options")
		}
		policies.Add(policy)
		params.Add(policies, "traffic_policies")
	}
	return params, nil
}

type DnsRecordSetTrafficPolicyOptions struct {
	DnsRecordSetIdOptions

	PROVIDER      string `help:"provider" choices:"Qcloud|Aws"`
	POLICY_TYPE   string `help:"PolicyType" choices:"Simple|ByCarrier|ByGeoLocation|BySearchEngine|IpRange|Weighted|Failover|MultiValueAnswer|Latency"`
	PolicyValue   string
	PolicyOptions string `help:"Json format policy"`
}

func (opts DnsRecordSetTrafficPolicyOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewArray()
	policy := jsonutils.NewDict()
	policy.Add(jsonutils.NewString(opts.PROVIDER), "provider")
	policy.Add(jsonutils.NewString(opts.POLICY_TYPE), "policy_type")
	policy.Add(jsonutils.NewString(opts.PolicyValue), "policy_value")
	if len(opts.PolicyOptions) > 0 {
		value, err := jsonutils.Parse([]byte(opts.PolicyOptions))
		if err != nil {
			return nil, errors.Wrapf(err, "jsonutils.Parse(%s)", opts.PolicyOptions)
		}
		policy.Add(value, "policy_options")
	}
	params.Add(policy)
	return jsonutils.Marshal(map[string]interface{}{"traffic_policies": params}), nil
}
