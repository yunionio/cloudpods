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

package aws

import (
	"fmt"
)

type SDnsTrafficPolicyInstance struct {
	Id                   string `xml:"Id"`
	Name                 string `xml:"Name"`
	TrafficPolicyId      string `xml:"TrafficPolicyId"`
	TrafficPolicyType    string `xml:"TrafficPolicyType"`
	TrafficPolicyVersion string `xml:"TrafficPolicyVersion"`
}

func (self *SAwsClient) GetDnsTrafficPolicyInstance(id string) (*SDnsTrafficPolicyInstance, error) {
	params := map[string]string{"Id": fmt.Sprintf("trafficpolicyinstance/%s", id)}
	ret := &struct {
		TrafficPolicyInstance SDnsTrafficPolicyInstance `xml:"TrafficPolicyInstance"`
	}{}
	err := self.dnsRequest("GetTrafficPolicyInstance", params, ret)
	if err != nil {
		return nil, err
	}
	return &ret.TrafficPolicyInstance, nil
}

type SDnsTrafficPolicy struct {
	Comment  string `xml:"Comment"`
	Document string `xml:"Document"`
	Name     string `xml:"Name"`
	Type     string `xml:"Type"`
}

func (self *SAwsClient) GetTrafficPolicy(id string, version string) (*SDnsTrafficPolicy, error) {
	params := map[string]string{"Id": fmt.Sprintf("trafficpolicy/%s/%s", id, version)}
	ret := &struct {
		TrafficPolicy SDnsTrafficPolicy `xml:"TrafficPolicy"`
	}{}
	err := self.dnsRequest("GetTrafficPolicy", params, ret)
	if err != nil {
		return nil, err
	}
	return &ret.TrafficPolicy, nil
}
