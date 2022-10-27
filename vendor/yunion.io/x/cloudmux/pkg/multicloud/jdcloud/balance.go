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

package jdcloud

import (
	"github.com/jdcloud-api/jdcloud-sdk-go/services/asset/apis"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/asset/client"

	"yunion.io/x/pkg/errors"
)

type SBalance struct {
	apis.DescribeAccountAmountResult
}

func (self *SJDCloudClient) DescribeAccountAmount() (*SBalance, error) {
	req := apis.NewDescribeAccountAmountRequest("cn-north-1")
	client := client.NewAssetClient(self.getCredential())
	client.Logger = Logger{debug: self.debug}
	resp, err := client.DescribeAccountAmount(req)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeAccountAmoun")
	}
	return &SBalance{DescribeAccountAmountResult: resp.Result}, nil
}
