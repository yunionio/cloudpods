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

package ksyun

import "time"

type SKeypair struct {
	KeyName    string
	KeyId      string
	IsChecked  bool
	CreateTime time.Time
	PublicKey  string
}

func (cli *SKsyunClient) GetKeypairs() ([]SKeypair, error) {
	params := map[string]interface{}{
		"MaxResults": "1000",
	}
	ret := []SKeypair{}
	for {
		body, err := cli.sksRequest("", "DescribeKeys", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			KeySet    []SKeypair
			NextToken string
		}{}
		err = body.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.KeySet...)
		if len(part.NextToken) == 0 {
			break
		}
		params["NextToken"] = part.NextToken
	}
	return ret, nil
}

func (cli *SKsyunClient) CreateKeypair(name, publicKey string) (*SKeypair, error) {
	params := map[string]interface{}{
		"KeyName":   name,
		"PublicKey": publicKey,
		"IsCheck":   "true",
	}
	resp, err := cli.sksRequest("", "ImportKey", params)
	if err != nil {
		return nil, err
	}
	ret := SKeypair{}
	err = resp.Unmarshal(&ret, "Key")
	if err != nil {
		return nil, err
	}
	return &ret, nil
}
