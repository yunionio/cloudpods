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
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type Authentication struct {
	PasswordCount int64  `xml:"PasswordCount"`
	Type          string `xml:"Type"`
}

type SElasticacheUser struct {
	multicloud.SElasticcacheAccountBase
	AwsTags
	region *SRegion

	ARN            string         `xml:"Arn"`
	AccessString   string         `xml:"AccessString"`
	Authentication Authentication `xml:"Authentication"`
	Engine         string         `xml:"Engine"`
	Status         string         `xml:"Status"`
	UserGroupIds   []string       `xml:"UserGroupIds>member"`
	UserId         string         `xml:"UserId"`
	UserName       string         `xml:"UserName"`
}

func (self *SElasticacheUser) GetId() string {
	return self.UserId
}

func (self *SElasticacheUser) GetName() string {
	return self.UserName
}

func (self *SElasticacheUser) GetGlobalId() string {
	return self.GetId()
}

func (self *SElasticacheUser) GetStatus() string {
	//  "active", "modifying" or "deleting"
	switch self.Status {
	case "active":
		return api.ELASTIC_CACHE_ACCOUNT_STATUS_AVAILABLE
	case "modifying":
		return api.ELASTIC_CACHE_ACCOUNT_STATUS_MODIFYING
	case "deleting":
		return api.ELASTIC_CACHE_ACCOUNT_STATUS_DELETING
	default:
		return self.Status
	}
}

func (self *SElasticacheUser) Refresh() error {
	users, err := self.region.GetCacheUsers("", self.UserId)
	if err != nil {
		return errors.Wrap(err, "region.DescribeUsers")
	}
	for i := range users {
		if users[i].UserId == self.UserId {
			return jsonutils.Update(self, users[i])
		}
	}
	return errors.Wrapf(cloudprovider.ErrNotFound, self.UserId)
}

func (self *SElasticacheUser) GetAccountType() string {
	return ""
}

func (self *SElasticacheUser) GetAccountPrivilege() string {
	return self.AccessString
}

func (self *SElasticacheUser) Delete() error {
	return cloudprovider.ErrNotSupported
}

func (self *SElasticacheUser) ResetPassword(input cloudprovider.SCloudElasticCacheAccountResetPasswordInput) error {
	return cloudprovider.ErrNotSupported
}

func (self *SElasticacheUser) UpdateAccount(input cloudprovider.SCloudElasticCacheAccountUpdateInput) error {
	return cloudprovider.ErrNotSupported
}

func (region *SRegion) GetCacheUsers(engine, userId string) ([]SElasticacheUser, error) {
	params := map[string]string{}
	if len(engine) > 0 {
		params["Engine"] = engine
	}
	if len(userId) > 0 {
		params["UserId"] = userId
	}
	ret := []SElasticacheUser{}
	for {
		part := struct {
			Marker string             `xml:"Marker"`
			Users  []SElasticacheUser `xml:"Users>member"`
		}{}
		err := region.ecRequest("DescribeUsers", params, &part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Users...)
		if len(part.Marker) == 0 || len(part.Users) == 0 {
			break
		}
	}
	return ret, nil
}
