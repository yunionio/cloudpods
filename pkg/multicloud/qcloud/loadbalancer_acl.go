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

package qcloud

import (
	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

// 腾讯云没有LB ACL
type SLBACL struct{}

func (self *SLBACL) GetAclListenerID() string {
	return ""
}

func (self *SLBACL) Sync(acl *cloudprovider.SLoadbalancerAccessControlList) error {
	return nil
}

func (self *SLBACL) Delete() error {
	return nil
}

func (self *SLBACL) GetId() string {
	return ""
}

func (self *SLBACL) GetName() string {
	return ""
}

func (self *SLBACL) GetGlobalId() string {
	return ""
}

func (self *SLBACL) GetStatus() string {
	return api.LB_BOOL_OFF
}

func (self *SLBACL) Refresh() error {
	return nil
}

func (self *SLBACL) IsEmulated() bool {
	return false
}

func (self *SLBACL) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SLBACL) GetAclEntries() *jsonutils.JSONArray {
	return nil
}

func (self *SLBACL) GetProjectId() string {
	return ""
}
