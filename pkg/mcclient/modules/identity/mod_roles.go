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
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type RolesManager struct {
	modulebase.ResourceManager
}

var (
	Roles   RolesManager
	RolesV3 RolesManager
)

func (this *RolesManager) Delete(session *mcclient.ClientSession, id string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return this.DeleteInContexts(session, id, body, nil)
}

func (this *RolesManager) DeleteInContexts(session *mcclient.ClientSession, id string, body jsonutils.JSONObject, ctxs []modulebase.ManagerContext) (jsonutils.JSONObject, error) {
	if ctxs == nil {
		err := httputils.JSONClientError{}
		err.Code = 403
		err.Details = fmt.Sprintf("role %s did not allowed deleted", id)

		if id == "admin" || id == "_member_" {
			return nil, &err
		}

		resp, e := this.Get(session, id, body)
		if e != nil {
			return nil, e
		} else {
			name, _ := resp.GetString("name")
			if name == "admin" || name == "_member_" {
				return nil, &err
			}
		}
	}

	return this.ResourceManager.DeleteInContexts(session, id, body, ctxs)
}

func (this *RolesManager) BatchDelete(session *mcclient.ClientSession, idlist []string, body jsonutils.JSONObject) []modulebase.SubmitResult {
	return this.BatchDeleteInContexts(session, idlist, body, nil)
}

func (this *RolesManager) BatchDeleteInContexts(session *mcclient.ClientSession, idlist []string, body jsonutils.JSONObject, ctxs []modulebase.ManagerContext) []modulebase.SubmitResult {
	return modulebase.BatchDo(idlist, func(id string) (jsonutils.JSONObject, error) {
		return this.DeleteInContexts(session, id, body, ctxs)
	})
}

func init() {
	Roles = RolesManager{ResourceManager: modules.NewIdentityManager("role", "roles",
		[]string{},
		[]string{"ID", "Name"})}

	Roles.SetVersion("v2.0/OS-KSADM")

	modules.Register(&Roles)

	RolesV3 = RolesManager{ResourceManager: modules.NewIdentityV3Manager("role", "roles",
		[]string{},
		[]string{"ID", "Name", "Domain_Id", "Project_Domain", "Description", "is_public"})}

	modules.Register(&RolesV3)
}
