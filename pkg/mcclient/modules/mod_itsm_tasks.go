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

type SProcessTasksManager struct {
	modulebase.ResourceManager
}

var (
	ProcessTasks SProcessTasksManager
)

func init() {
	ProcessTasks = SProcessTasksManager{NewITSMManager("process-task", "process-tasks",
		[]string{"id", "name", "description", "owner", "priority", "tenant_id"},
		[]string{},
	)}

	register(&ProcessTasks)
}

func (this *SProcessTasksManager) QuickComplete(s *mcclient.ClientSession, params jsonutils.JSONObject) error {
	path := fmt.Sprintf("/%s/quick-complete", this.KeywordPlural)
	if params != nil {
		qs := params.QueryString()
		if len(qs) > 0 {
			path = fmt.Sprintf("%s?%s", path, qs)
		}
	}
	_, err := modulebase.Get(this.ResourceManager, s, path, "")
	return err
}
