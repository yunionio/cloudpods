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

package modules

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type ContactsManager struct {
	modulebase.ResourceManager
}

func (this *ContactsManager) PerformActionWithArrayParams(s *mcclient.ClientSession, id string, action string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s/%s/%s", this.KeywordPlural, id, action)

	body := jsonutils.NewDict()
	if params != nil {
		body.Add(params, this.KeywordPlural)
	}

	return modulebase.Post(this.ResourceManager, s, path, body, this.Keyword)
}

func (this *ContactsManager) DoBatchDeleteContacts(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	path := "/contacts/delete-contact"

	return modulebase.Post(this.ResourceManager, s, path, params, this.Keyword)
}

var (
	Contacts ContactsManager
)

func init() {
	Contacts = ContactsManager{NewNotifyManager("contact", "contacts",
		[]string{"id", "name", "display_name", "details", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "project_id", "remark"},
		[]string{})}

	register(&Contacts)
}
