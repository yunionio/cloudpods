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

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type SIdentityUsageManager struct {
	modulebase.ResourceManager
}

func (this *SIdentityUsageManager) GetUsage(session *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	url := "/usages"
	if params != nil {
		query := params.QueryString()
		if len(query) > 0 {
			url = fmt.Sprintf("%s?%s", url, query)
		}
	}
	return modulebase.Get(this.ResourceManager, session, url, "usage")
}

var (
	IdentityUsages SIdentityUsageManager
	IdentityLogs   modulebase.ResourceManager
)

func init() {
	IdentityUsages = SIdentityUsageManager{modules.NewIdentityV3Manager("usage", "usages",
		[]string{},
		[]string{})}

	IdentityLogs = modules.NewIdentityV3Manager("event", "events",
		[]string{"id", "ops_time", "obj_id", "obj_type", "obj_name", "user", "user_id", "tenant", "tenant_id", "owner_tenant_id", "action", "notes"},
		[]string{})
}
