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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

func (self *SRegion) GetElasticacheUsers(engine string) ([]SElasticacheUser, error) {
	params := map[string]string{}
	if len(engine) > 0 {
		params["Engine"] = engine
	}
	ret := []SElasticacheUser{}
	for {
		result := struct {
			Marker string
			Users  []SElasticacheUser `xml:"Users>member"`
		}{}
		err := self.redisRequest("DescribeUsers", params, &result)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeUsers")
		}
		ret = append(ret, result.Users...)
		if len(result.Marker) == 0 || len(result.Users) == 0 {
			break
		}
		params["Marker"] = result.Marker
	}
	return ret, nil
}

type Authentication struct {
	PasswordCount int64  `xml:"PasswordCount"`
	Type          string `xml:"Type"`
}

type SElasticacheUser struct {
	multicloud.SElasticcacheAccountBase
	multicloud.AwsTags
	cache *SElasticache

	ARN            string         `xml:"ARN"`
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
		return ""
	}
}

func (self *SElasticacheUser) Refresh() error {
	users, err := self.cache.region.GetElasticacheUsers("")
	if err != nil {
		return errors.Wrap(err, "region.DescribeUsers")
	}
	for i := range users {
		if users[i].UserId == self.GetId() {
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
