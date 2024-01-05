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

package aliyun

import (
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SAccessGroupRule struct {
	region *SRegion

	AccessGroupName string
	RWAccess        string
	UserAccess      string
	Priority        int
	SourceCidrIp    string
	AccessRuleId    string
}

func (self *SAccessGroupRule) GetGlobalId() string {
	return self.AccessRuleId
}

func (self *SAccessGroupRule) Delete() error {
	return self.region.DeleteAccessGroupRule(self.AccessGroupName, self.AccessRuleId)
}

func (self *SAccessGroupRule) GetPriority() int {
	return self.Priority
}

func (self *SAccessGroupRule) GetSource() string {
	return self.SourceCidrIp
}

func (self *SAccessGroupRule) GetRWAccessType() cloudprovider.TRWAccessType {
	switch self.RWAccess {
	case "RDWR":
		return cloudprovider.RWAccessTypeRW
	case "RDONLY":
		return cloudprovider.RWAccessTypeR
	default:
		return cloudprovider.TRWAccessType(self.RWAccess)
	}
}

func (self *SAccessGroupRule) GetUserAccessType() cloudprovider.TUserAccessType {
	switch self.UserAccess {
	case "no_squash":
		return cloudprovider.UserAccessTypeNoRootSquash
	case "root_squash":
		return cloudprovider.UserAccessTypeRootSquash
	case "all_squash":
		return cloudprovider.UserAccessTypeAllSquash
	default:
		return cloudprovider.TUserAccessType(self.UserAccess)
	}
}
