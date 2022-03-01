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

package bingocloud

import "github.com/pkg/errors"

type SEip struct {
}

func (self *SRegion) GetEips(ip, nextToken string) ([]SEip, string, error) {
	params := map[string]string{}
	if len(ip) > 0 {
		params["publicIp"] = ip
	}
	if len(nextToken) > 0 {
		params["nextToken"] = nextToken
	}
	resp, err := self.invoke("DescribeAddresses", params)
	if err != nil {
		return nil, "", errors.Wrapf(err, "DescribeAddresses")
	}
	ret := struct {
		AddressesSet []SEip
		NextToken    string
	}{}
	resp.Unmarshal(&ret)
	return ret.AddressesSet, ret.NextToken, nil
}
