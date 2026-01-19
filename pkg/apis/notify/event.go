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

	// 创建
	ActionCreate SAction = "create"
	// 删除
	ActionDelete SAction = "delete"
	// 放入回收站
	ActionPendingDelete SAction = "pending_delete"
	// 更新
	ActionUpdate SAction = "update"
	// 重装系统
	ActionRebuildRoot SAction = "rebuild_root"
	// 重置密码
	ActionResetPassword SAction = "reset_password"
	// 配置变更
	ActionChangeConfig SAction = "change_config"
	// 扩容
	ActionResize SAction = "resize"
	// 到期释放
	ActionExpiredRelease SAction = "expired_release"
	// 执行，例如自动快照策略执行创建快照、弹性伸缩策略执行扩容、定时任务执行等
	ActionExecute SAction = "execute"
	// IP变更
	ActionChangeIpaddr SAction = "change_ipaddr"
	// 同步状态
	ActionSyncStatus SAction = "sync_status"
	// 清理数据
	ActionCleanData SAction = "clean_data"
	// 迁移
	ActionMigrate SAction = "migrate"

	// 添加备份服务器
	ActionCreateBackupServer SAction = "add_backup_server"
	// 删除备份服务器
	ActionDelBackupServer SAction = "delete_backup_server"

	// 同步新建
	ActionSyncCreate SAction = "sync_create"
	// 同步更新
	ActionSyncUpdate SAction = "sync_update"
	// 同步删除
	ActionSyncDelete SAction = "sync_delete"
	// 同步账号状态
	ActionSyncAccountStatus SAction = "sync_account_status"

	// 下线
	ActionOffline SAction = "offline"
	// 系统崩溃
	ActionSystemPanic SAction = "panic"
	// 系统异常
	ActionSystemException SAction = "exception"
	// 锁定，例如用户锁定
	ActionLock SAction = "lock"
	// 超出数量
	ActionExceedCount SAction = "exceed_count"
	// 密码即将过期
	ActionPasswordExpireSoon SAction = "password_expire_soon"
	// 一致性检查
	ActionChecksumTest SAction = "checksum_test"
	// 任务队列阻塞
	ActionWorkerBlock SAction = "woker_block"
	// 网络同步失败
	ActionNetOutOfSync SAction = "net_out_of_sync"
	// MySQL同步失败
	ActionMysqlOutOfSync SAction = "mysql_out_of_sync"
	// 服务异常
	ActionServiceAbnormal SAction = "service_abnormal"
	// 服务器崩溃
	ActionServerPanicked SAction = "server_panicked"
	// 挂载
	ActionAttach SAction = "attach"
	// 卸载
	ActionDetach SAction = "detach"
	// 透传设备创建
	ActionIsolatedDeviceCreate SAction = "isolated_device_create"
	// 透传设备更新
	ActionIsolatedDeviceUpdate SAction = "isolated_device_update"
	// 透传设备删除
	ActionIsolatedDeviceDelete SAction = "isolated_device_delete"
	// 状态变更
	ActionStatusChanged SAction = "status_changed"
	// 运行任务
	ActionRunTask SAction = "run_task"

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
