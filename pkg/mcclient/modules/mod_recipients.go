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

type RecipientsManager struct {
	modulebase.ResourceManager
}

func (this *RecipientsManager) DoDeleteRecipient(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s", this.KeywordPlural)

	body := jsonutils.NewDict()
	if params != nil {
		body.Add(params, this.Keyword)
	}

	return modulebase.Delete(this.ResourceManager, s, path, body, this.Keyword)
}

var (
	Recipients RecipientsManager
)

func init() {

	Recipients = RecipientsManager{NewServiceTreeManager("recipient", "recipients",
		[]string{"id", "type", "details", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark"},
		[]string{})}

	register(&Recipients)
}
