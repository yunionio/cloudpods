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

package identity

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type GroupManager struct {
	modulebase.ResourceManager
}

func (this *GroupManager) GetUsers(s *mcclient.ClientSession, gid string, query jsonutils.JSONObject) (*printutils.ListResult, error) {
	url := fmt.Sprintf("/groups/%s/users", gid)
	if query != nil {
		qs := query.QueryString()
		if len(qs) > 0 {
			url = fmt.Sprintf("%s?%s", url, qs)
		}
	}
	return modulebase.List(this.ResourceManager, s, url, "users")
}

var (
	Groups GroupManager
)

func (this *GroupManager) GetProjects(session *mcclient.ClientSession, uid string) (*printutils.ListResult, error) {
	url := fmt.Sprintf("/groups/%s/projects?admin=true", uid)
	return modulebase.List(this.ResourceManager, session, url, "projects")
}

func init() {
	Groups = GroupManager{modules.NewIdentityV3Manager("group", "groups",
		[]string{},
		[]string{"ID", "Name", "Domain_Id", "project_domain",
			"User_Count", "Description"})}

	modules.Register(&Groups)
}
