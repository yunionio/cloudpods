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

package notify

import (
	"fmt"
	"strings"
)

var (
	Event SEvent

	ActionCreate         SAction = "create"
	ActionDelete         SAction = "delete"
	ActionPendingDelete  SAction = "pending_delete"
	ActionUpdate         SAction = "update"
	ActionRebuildRoot    SAction = "rebuild_root"
	ActionResetPassword  SAction = "reset_password"
	ActionChangeConfig   SAction = "change_config"
	ActionResize         SAction = "resize"
	ActionExpiredRelease SAction = "expired_release"
	ActionExecute        SAction = "execute"
	ActionChangeIpaddr   SAction = "change_ipaddr"
	ActionSyncStatus     SAction = "sync_status"
	ActionCleanData      SAction = "clean_data"
	ActionMigrate        SAction = "migrate"

	ActionCreateBackupServer SAction = "add_backup_server"
	ActionDelBackupServer    SAction = "delete_backup_server"

	ResultFailed  SResult = "failed"
	ResultSucceed SResult = "succeed"
)

const (
	DelimiterInEvent = "/"
)

type SAction string

type SResult string

type SEvent struct {
	resourceType string
	action       SAction
	result       SResult
}

func (se SEvent) WithResourceType(rt string) SEvent {
	se.resourceType = rt
	return se
}

func (se SEvent) WithAction(a SAction) SEvent {
	se.action = a
	return se
}

func (se SEvent) WithResult(r SResult) SEvent {
	se.result = r
	return se
}

func (se SEvent) ResourceType() string {
	return se.resourceType
}

func (se SEvent) Action() SAction {
	return se.action
}

func (se SEvent) ActionWithResult(delimiter string) string {
	ar := string(se.action)
	if len(se.result) > 0 {
		ar += delimiter + string(se.result)
	}
	return strings.ToUpper(ar)
}

func (se SEvent) Result() SResult {
	if se.result == "" {
		return ResultSucceed
	}
	return se.result
}

func (se SEvent) String() string {
	return se.StringWithDeli(DelimiterInEvent)
}

func (se SEvent) StringWithDeli(delimiter string) string {
	str := strings.ToUpper(fmt.Sprintf("%s%s%s", se.ResourceType(), delimiter, se.Action()))
	if se.result != "" {
		str += delimiter + strings.ToUpper(string(se.result))
	}
	return str
}
