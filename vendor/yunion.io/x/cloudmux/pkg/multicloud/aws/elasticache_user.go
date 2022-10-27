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
	"github.com/aws/aws-sdk-go/service/elasticache"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

func (region *SRegion) DescribeUsers(engine string) ([]*elasticache.User, error) {
	ecClient, err := region.getAwsElasticacheClient()
	if err != nil {
		return nil, errors.Wrap(err, "client.getAwsElasticacheClient")
	}

	input := elasticache.DescribeUsersInput{}
	if len(engine) > 0 {
		input.Engine = &engine
	}
	marker := ""
	maxrecords := (int64)(50)
	input.MaxRecords = &maxrecords

	users := []*elasticache.User{}
	for {
		if len(marker) >= 0 {
			input.Marker = &marker
		}
		out, err := ecClient.DescribeUsers(&input)
		if err != nil {
			return nil, errors.Wrap(err, "ecClient.DescribeCacheClusters")
		}
		users = append(users, out.Users...)

		if out.Marker != nil && len(*out.Marker) > 0 {
			marker = *out.Marker
		} else {
			break
		}
	}

	return users, nil
}

type SElasticacheUser struct {
	multicloud.SElasticcacheAccountBase
	AwsTags
	region *SRegion
	user   *elasticache.User
}

func (self *SElasticacheUser) GetId() string {
	return *self.user.UserId
}

func (self *SElasticacheUser) GetName() string {
	return *self.user.UserName
}

func (self *SElasticacheUser) GetGlobalId() string {
	return self.GetId()
}

func (self *SElasticacheUser) GetStatus() string {
	if self.user == nil || self.user.Status == nil {
		return ""
	}
	//  "active", "modifying" or "deleting"
	switch *self.user.Status {
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
	users, err := self.region.DescribeUsers("")
	if err != nil {
		return errors.Wrap(err, "region.DescribeUsers")
	}
	for i := range users {
		if users[i] != nil && users[i].UserId != nil && *users[i].UserId == self.GetId() {
			self.user = users[i]
		}
	}
	return nil
}

func (self *SElasticacheUser) GetAccountType() string {
	return ""
}

func (self *SElasticacheUser) GetAccountPrivilege() string {
	if self.user != nil && self.user.AccessString != nil {
		return *self.user.AccessString
	}
	return ""
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
