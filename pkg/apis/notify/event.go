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
	Event SNotifyEvent

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

	ActionSyncCreate        SAction = "sync_create"
	ActionSyncUpdate        SAction = "sync_update"
	ActionSyncDelete        SAction = "sync_delete"
	ActionSyncAccountStatus SAction = "sync_account_status"

	ActionOffline         SAction = "offline"
	ActionSystemPanic     SAction = "panic"
	ActionSystemException SAction = "exception"

	ActionChecksumTest SAction = "checksum_test"

	ActionLock SAction = "lock"

	ActionExceedCount          SAction = "exceed_count"
	ActionPasswordExpireSoon   SAction = "password_expire_soon"
	ActionWorkerBlock          SAction = "woker_block"
	ActionNetOutOfSync         SAction = "net_out_of_sync"
	ActionMysqlOutOfSync       SAction = "mysql_out_of_sync"
	ActionServiceAbnormal      SAction = "service_abnormal"
	ActionServerPanicked       SAction = "server_panicked"
	ActionAttach               SAction = "attach"
	ActionDetach               SAction = "detach"
	ActionIsolatedDeviceCreate SAction = "isolated_device_create"
	ActionIsolatedDeviceUpdate SAction = "isolated_device_update"
	ActionIsolatedDeviceDelete SAction = "isolated_device_delete"
	ActionStatusChanged        SAction = "status_changed"

	ResultFailed  SResult = "failed"
	ResultSucceed SResult = "succeed"
)

const (
	DelimiterInEvent = "/"
)

type SAction string

type SResult string

type SNotifyEvent struct {
	resourceType string
	action       SAction
	result       SResult
}

func (se SNotifyEvent) WithResourceType(rt string) SNotifyEvent {
	se.resourceType = rt
	return se
}

func (se SNotifyEvent) WithAction(a SAction) SNotifyEvent {
	se.action = a
	return se
}

func (se SNotifyEvent) WithResult(r SResult) SNotifyEvent {
	se.result = r
	return se
}

func (se SNotifyEvent) ResourceType() string {
	return se.resourceType
}

func (se SNotifyEvent) Action() SAction {
	return se.action
}

func (se SNotifyEvent) ActionWithResult(delimiter string) string {
	ar := string(se.action)
	if len(se.result) > 0 {
		ar += delimiter + string(se.result)
	}
	return strings.ToUpper(ar)
}

func (se SNotifyEvent) Result() SResult {
	if se.result == "" {
		return ResultSucceed
	}
	return se.result
}

func (se SNotifyEvent) String() string {
	return se.StringWithDeli(DelimiterInEvent)
}

func (se SNotifyEvent) StringWithDeli(delimiter string) string {
	str := strings.ToUpper(fmt.Sprintf("%s%s%s", se.ResourceType(), delimiter, se.Action()))
	if se.result != "" {
		str += delimiter + strings.ToUpper(string(se.result))
	}
	return str
}
