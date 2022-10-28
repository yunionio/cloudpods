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
	"github.com/aws/aws-sdk-go/service/route53"

	"yunion.io/x/pkg/errors"
)

type STrafficPolicy struct {
	client   *SAwsClient
	Comment  string `json:"Comment"`
	Document string `json:"Document"` //require decode
	Id       string `json:"Id"`
	Name     string `json:"Name"`
	DNSType  string `json:"Type"`
	Version  int64  `json:"Version"`
}

func (client *SAwsClient) GetSTrafficPolicyById(TrafficPolicyInstanceId string) (*STrafficPolicy, error) {
	s, err := client.getAwsRoute53Session()
	if err != nil {
		return nil, errors.Wrap(err, "region.getAwsRoute53Session()")
	}
	route53Client := route53.New(s)
	params := route53.GetTrafficPolicyInput{}
	params.Id = &TrafficPolicyInstanceId
	var Version int64 = 1
	params.Version = &Version
	ret, err := route53Client.GetTrafficPolicy(&params)
	if err != nil {
		return nil, errors.Wrap(err, "route53Client.GetTrafficPolicy")
	}
	result := STrafficPolicy{}
	err = unmarshalAwsOutput(ret, "TrafficPolicy", &result)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalAwsOutput(TrafficPolicy)")
	}
	return &result, nil
}
