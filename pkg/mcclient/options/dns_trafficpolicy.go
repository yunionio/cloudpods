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

type DnsTrafficPolicyListOptions struct {
	BaseListOptions
	PolicyType string `choices:"Simple|ByCarrier|ByGeoLocation|BySearchEngine|IpRange|Weighted|Failover|MultiValueAnswer|Latency"`
}

func (opts *DnsTrafficPolicyListOptions) Params() (jsonutils.JSONObject, error) {
	return ListStructToParams(opts)
}

type DnsTrafficPolicyCreateOptions struct {
	BaseCreateOptions
	PROVIDER      string `choices:"Aws|Qcloud"`
	POLICY_TYPE   string `choices:"Simple|ByCarrier|ByGeoLocation|BySearchEngine|IpRange|Weighted|Failover|MultiValueAnswer|Latency"`
	PolicyValue   string
	PolicyOptions string
}

func (opts *DnsTrafficPolicyCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts).(*jsonutils.JSONDict)
	params.Remove("policy_options")
	if len(opts.PolicyOptions) > 0 {
		policyParams, err := jsonutils.Parse([]byte(opts.PolicyOptions))
		if err != nil {
			return nil, errors.Wrapf(err, "jsonutils.Parse(%s)", opts.PolicyOptions)
		}
		params.Add(policyParams, "options")
	}
	return params, nil
}
