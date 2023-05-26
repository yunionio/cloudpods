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

import "yunion.io/x/onecloud/pkg/apis"

type TemplateCreateInput struct {
	apis.StandaloneAnonResourceCreateInput

	// description: Contact type, specifically, setting it to all means all contact type
	// require: true
	// example: email
	ContactType string `json:"contact_type"`
	// description: Template type
	// enum: title,content,remote
	// example: title
	TemplateType string `json:"template_type"`

	// description: Template topic
	// required: true
	// example: IMAGE_ACTIVE
	Topic string `json:"topic"`

	// description: Template content
	// required: true
	// example: 镜像 {{.name}} 上传完成
	Content string `json:"content"`
	// description: Example for using this template
	// required: true
	// example: {"name": "centos7.6"}
	Example string `json:"example"`
	// description: Language
	// enum: cn,en
	Lang string `json:"lang"`
}

type TemplateManagerSaveInput struct {
	ContactType string
	Templates   []TemplateCreateInput
	Force       bool
}

type TemplateListInput struct {
	apis.StandaloneAnonResourceListInput

	// description: Contact type, specifically, setting it to all means all contact type
	// require: true
	// example: email
	ContactType string `json:"contact_type"`

	// description: Template type
	// enum: title,content,remote
	// example: title
	TemplateType string `json:"template_type"`

	// description: template topic
	// required: true
	// example: IMAGE_ACTIVE
	Topic string `json:"topic"`

	// description: Language
	// enum: cn,en
	Lang string `json:"lang"`
}

type TemplateUpdateInput struct {
	apis.StandaloneAnonResourceBaseUpdateInput

	// description: template content
	// required: true
	// example: 镜像 {{.name}} 上传完成
	Content string `json:"content"`
	// description: all example for using this template
	// required: true
	// example: {"name": "centos7.6"}
	Example string `json:"example"`
}

type TemplateDetails struct {
	apis.StandaloneResourceDetails

	STemplate
}

// 密码即将失效通知
const (
	PWD_EXPIRE_SOON_TITLE_CN = `{{- $d := .resource_details -}}
	{{ $d.account }}:您的密码有效期将过`
	PWD_EXPIRE_SOON_TITLE_EN = `{{- $d := .resource_details -}}
	{{ $d.account }}:Your password is valid and will expire soon`
	PWD_EXPIRE_SOON_CONTENT_CN = `{{- $d := .resource_details -}}
	{{ $d.account }}:您的密码有效期将过，请及时登录平台更新密码。`
	PWD_EXPIRE_SOON_CONTENT_EN = `{{- $d := .resource_details -}}
	{{ $d.account }}:Your password is valid and will expire soon. Please log in to the platform in time to update your password.`
)

// 资源即将到期通知
const (
	EXPIRED_RELEASE_TITLE_CN = `{{- $d := .resource_details -}}
	{{ $d.project }}项目的
	{{ .resource_type_display }}{{ $d.name }}到期前{{ .advance_days }}天通知`
	EXPIRED_RELEASE_TITLE_EN = `{{- $d := .resource_details -}}
	{{ .advance_days }} days notice before {{ .resource_type }} {{ $d.name }} {{ if $d.project -}} in project {{ $d.project }} {{ end -}} expiration`
	EXPIRED_RELEASE_CONTENT_CN = `{{- $d := .resource_details -}}
	您在{{ $d.project }}项目的
	{{- if $d.brand -}}
	{{ $d.brand }}平台
	{{- end -}}
	
	{{- if $d.private_dns -}}
	，内网地址为{{ $d.private_dns }}:{{ $d.private_connect_port }}
	{{- end -}}
	
	{{- if $d.public_dns -}}
	，外网地址为{{ $d.public_dns }}:{{ $d.public_connect_port }}的
	{{- end -}}
	{{ .resource_type_display }}{{ $d.name }}还有{{ .advance_days }}天就要到期释放，{{ if $d.auto_renew }}到期已开启自动续费，{{ end }}如有其它变更，请尽快前往控制台处理`
	EXPIRED_RELEASE_CONTENT_EN = `{{- $d := .resource_details -}}
	Your {{ if $d.brand -}} {{ $d.brand }} {{ end -}} {{ .resource_type }} {{ $d.name }} {{ if $d.public_dns -}} with external address {{ $d.public_dns }}:{{ $d.public_connect_port }} {{ end -}} {{ if $d.project -}} in project {{ $d.project }} {{ end -}} will expire and be released in {{ .advance_days }} days. {{ if $d.auto_renew }}It has turned on automatic renewal. {{ end }}If there are other changes, please go to the console as soon as possible.`
)

// 服务崩溃通知
const (
	PANIC_TITLE_CN = `{{- $d := .resource_details -}}
	{{ .resource_type }} {{ $d.task_name }} 崩溃了`
	PANIC_TITLE_EN = `{{- $d := .resource_details -}}
	{{ .resource_type }} {{ $d.task_name }} PANIC`
	PANIC_CONTENT_CN = `{{- $d := .resource_details -}}
	{{ .resource_type }} {{ $d.task_name }} PANIC {{- if $d.error -}} 错误: {{ $d.error }} {{- end -}}
	堆栈信息: 
	{{ $d.stack }}`
	PANIC_CONTENT_EN = `{{- $d := .resource_details -}}
	{{ .resource_type }} {{ $d.task_name }} PANIC {{- if $d.error -}} Error: {{ $d.error }} {{- end -}}
	Stack Info: 
	{{ $d.stack }}`
)

// 日志容量超限通知
const (
	ACTION_LOG_EXCEED_COUNT_TITLE_CN = `{{- $d := .resource_details -}}
	操作日志超出设置数量{{ $d.exceed_count }}条，当前{{ $d.current_count }}条`
	ACTION_LOG_EXCEED_COUNT_TITLE_EN = `{{- $d := .resource_details -}}
	Action logs excced expected count {{ $d.exceed_count }}, current count is {{ $d.current_count }}`
	ACTION_LOG_EXCEED_COUNT_CONTENT_CN = `{{- $d := .resource_details.action -}}
	当前日志 ID: {{ $d.id }}`
	ACTION_LOG_EXCEED_COUNT_CONTENT_EN = `{{- $d := .resource_details.action -}}
	Current log ID: {{ $d.id }}`
)

// 完整性校验失败通知
const (
	CHECKSUM_TEST_FAILED_TITLE_CN = `{{- $d := .resource_details -}}
	{{- if eq .resource_type "cloudpods_component" }}
	{{ $d.title }}
	{{- else -}}
	{{ .resource_type_display }}完整性校验失败
	{{- end -}}`
	CHECKSUM_TEST_FAILED_TITLE_EN = `{{- $d := .resource_details -}}
	{{- if eq .resource_type "cloudpods_component" }}
	{{ $d.title }}
	{{- else -}}
	The checksum of {{ .resource_type_display }} test failed
	{{- end -}}`
	CHECKSUM_TEST_FAILED_CONTENT_CN = `{{- $d := .resource_details -}}
	{{- if eq .resource_type "db_table_record" }}
	表{{ $d.table_name }}记录{{ $d.name }}被修改，完整性校验失败。期望校验和({{ $d.expected_checksum }}) != 计算校验和({{ $d.calculated_checksum }})。
	{{- end -}}
	{{- if eq .resource_type "cloudpods_component" }}
	{{ $d.details }}
	{{- end -}}
	{{- if eq .resource_type "snapshot" }}
	快照{{ $d.name }}的内存快照完整性校验失败
	{{- end -}}
	{{- if eq .resource_type "image" }}
	镜像{{ $d.name }}完整性校验失败
	{{- end -}}
	{{- if eq .resource_type "vm_integrity" }}
	主机{{ $d.name }}完整性校验失败
	{{- end -}}`
	CHECKSUM_TEST_FAILED_CONTENT_EN = `{{- $d := .resource_details -}}
	{{- if eq .resource_type "db_table_record" }}
	The record {{ $d.name }} in table {{ $d.table_name }} of the database {{ $d.db_name }} has been modified because the checksum test failed. Expected_checksum({{ $d.expected_checksum }}) != Calculated_checksum({{ $d.calculated_checksum }}).
	{{- end -}}
	{{- if eq .resource_type "cloudpods_component" }}
	{{ $d.details }}
	{{- end -}}
	{{- if eq .resource_type "snapshot" }}
	The checksum of the memory snapshot of the snapshot {{ $d.name }} test failed.
	{{- end -}}
	{{- if eq .resource_type "image" }}
	The checksum of the image {{ $d.name }} test failed.
	{{- end -}}`
)

// 通用通知
const (
	COMMON_TITLE_CN = `{{- $d := .resource_details -}}
	{{- if $d.project -}}
	{{ $d.project }}项目的
	{{- end -}}
	{{ .resource_type_display }}{{ $d.name }}{{ .action_display }}{{ .result_display }}`
	COMMON_TITLE_EN = `{{- $d := .resource_details -}}
	The {{ .resource_type_display }} {{ .Name }} {{ if $d.project }} in poject {{ $d.project }} {{ end -}} {{ .action_display }} {{ .result_display }}`
	COMMON_CONTENT_CN = `{{- $d := .resource_details -}}
	您
	{{- if $d.project -}}
	在{{ $d.project }}项目
	{{- end -}}
	{{- if $d.brand -}}
	{{ $d.brand }}平台
	{{- end -}}
	的{{ .resource_type_display }}{{ $d.name }}{{ .action_display }}{{.result_display}}
	{{- if eq .result "failed" -}}
	，请尽快前往控制台进行处理
	{{- end -}}`
	COMMON_CONTENT_EN = `{{- $d := .resource_details -}}
	Your {{ if $d.brand -}} {{ $d.brand }} {{ end -}} {{ .resource_type_display }} {{ $d.name }} {{ if $d.project -}} in project {{ $d.project }} {{ end -}} has been {{ .action_display }} {{ .result_display }}
	{{- if eq .result "failed" -}}
	. And please go to the console as soon as possible to process.
	{{- end -}}`
)

// 服务组件异常通知
const (
	EXCEPTION_TITLE_CN = `{{- $d := .resource_details -}}
	{{ .resource_type }} {{ $d.name }} 发生异常: {{ $d.message }}`
	EXCEPTION_TITLE_EN = `{{- $d := .resource_details -}}
	{{ .resource_type }} {{ $d.name }} exception occurs: {{ $d.message }}`
	EXCEPTION_CONTENT_CN = `{{- $d := .resource_details -}}
	{{ .resource_type }} {{ $d.name }} 发生异常: {{ $d.message }}`
	EXCEPTION_CONTENT_EN = `{{- $d := .resource_details -}}
	{{ .resource_type }} {{ $d.name }} exception occurs: {{ $d.message }}`
)

// 数据库主从同步不一致通知
const (
	MYSQL_OUT_OF_SYNC_TITLE_CN = `{{- $d := .resource_details -}}
	数据库 {{$d.ip}} 的主从同步不一致，请及时检查。`
	MYSQL_OUT_OF_SYNC_TITLE_EN = `{{- $d := .resource_details -}}
	The primary and secondary synchronization of the database ({{ $d.ip }}) is inconsistent, please check in time.`
	MYSQL_OUT_OF_SYNC_CONTENT_CN = `{{- $d := .resource_details -}}
	数据库 {{$d.ip}} 的主从同步不一致，请及时检查。
	{{ range $status := $d.status }}
	数据库 {{$status.ip}} 状态:
	{{- if not $status.operator_error }}
	  - Slave_IO_Running: {{$status.slave_io_running}}
	  - Slave_SQL_Running: {{$status.slave_sql_running}}
	  {{- if $status.last_error }}
	  - Last_Error: {{$status.last_error}}
	  {{- end -}}
	  {{- if $status.last_io_error }}
	  - Last_IO_Error: {{$status.last_io_error}}
	  {{- end }}
	{{else}}
	  - Operator_Error: {{$status.operator_error}}
	{{- end}}
	{{end}}`
	MYSQL_OUT_OF_SYNC_CONTENT_EN = `{{- $d := .resource_details -}}
	The primary and secondary synchronization of the database ({{ $d.ip }}) is inconsistent, please check in time.
	{{ range $status := $d.status }}
	Database {{$status.ip}} status:
	{{- if not $status.operator_error }}
	  - Slave_IO_Running: {{$status.slave_io_running}}
	  - Slave_SQL_Running: {{$status.slave_sql_running}}
	  {{- if $status.last_error }}
	  - Last_Error: {{$status.last_error}}
	  {{- end -}}
	  {{- if $status.last_io_error }}
	  - Last_IO_Error: {{$status.last_io_error}}
	  {{- end }}
	{{else}}
	  - Operator_Error: {{$status.operator_error}}
	{{- end}}
	{{end}}`
)

// 网络拓扑不一致通知
const (
	NET_OUT_OF_SYNC_TITLE_CN = `{{- $d := .resource_details -}}
	{{ $d.service_name }}服务的网络拓扑信息同步不一致，请及时检查。	`
	NET_OUT_OF_SYNC_TITLE_EN = `{{- $d := .resource_details -}}
	{{ $d.service_name }}: The network topology information of the service is inconsistent, please check in time.`
	NET_OUT_OF_SYNC_CONTENT_CN = `{{- $d := .resource_details -}}
	{{ $d.service_name }}服务的网络拓扑信息同步不一致，请及时检查。	`
	NET_OUT_OF_SYNC_CONTENT_EN = `{{- $d := .resource_details -}}
	{{ $d.service_name }}: The network topology information of the service is inconsistent, please check in time.`
)

// 离线通知
const (
	OFFLINE_TITLE_CN = `{{- $d := .resource_details -}}
	{{ .resource_type }} {{ $d.name }} 离线`
	OFFLINE_TITLE_EN = `{{- $d := .resource_details -}}
	The {{ .resource_type }} {{ $d.name }} Offline `
	OFFLINE_CONTENT_CN = `{{- $d := .resource_details -}}
	{{ .resource_type }} {{ $d.name }} 离线 {{- if $d.reason -}} 原因: {{ $d.reason }} {{- end -}}`
	OFFLINE_CONTENT_EN = `{{- $d := .resource_details -}}
	The {{ .resource_type }} {{ $d.name }} Offline {{- if $d.reason -}} Reason: {{ $d.reason }}的 {{- end -}}`
)

// 资源加入回收站通知
const (
	PENDING_DELETE_TITLE_CN = `{{- $d := .resource_details -}}
	{{- if $d.project -}}
	{{ $d.project }}项目的
	{{- end -}}
	{{ .resource_type_display }}{{ $d.name }}成功加入回收站`
	PENDING_DELETE_TITLE_EN = `{{- $d := .resource_details -}}
	The {{ .resource_type_display }} {{ .Name }} {{ if $d.project }} in poject {{ $d.project }} {{ end -}} has been added to recycle bin successfully`
	PENDING_DELETE_CONTENT_CN = `{{- $d := .resource_details -}}
	您在{{ $d.project }}项目的
	
	{{- if $d.ips -}}
	IP地址为{{ $d.ips }}的
	{{- end -}}
	
	{{- if $d.ip_addr -}}
	IP地址为{{ $d.ip_addr }}的
	{{- end -}}
	{{- if $d.brand -}}
	{{ $d.brand }}平台
	{{- end -}}
	{{ .resource_type_display }}{{ $d.name }}成功加入回收站`
	PENDING_DELETE_CONTENT_EN = `{{- $d := .resource_details -}}
	Your {{ if $d.brand -}} {{ $d.brand }} {{ end -}} {{ .resource_type_display }} {{ $d.name }} 
	{{ if $d.project -}} in project {{ $d.project }} {{ end -}} has been added to recycle bin successfully
	{{- if $d.private_dns -}}
	, the intranet address is {{ $d.private_dns }}:{{ $d.private_connect_port }}
	{{- end -}}
	
	{{- if $d.public_dns -}}
	, the external address is {{ $d.public_dns }}:{{ $d.public_connect_port }}
	{{- end -}}
	
	{{- if and $d.address_type $d.address -}}
	, the service address is {{ $d.address_type }}{{ $d.address }}
	{{- end -}}
	
	{{- if $d.ips -}}
	, the IP address is {{ $d.ips }}
	{{- end -}}
	
	{{- if $d.ip_addr -}}
	, the IP address is {{ $d.ip_addr }}
	{{- end -}}`
)

// 虚拟机崩溃通知
const (
	SERVER_PANICKED_TITLE_CN = `{{- $d := .resource_details -}}
	您在{{ $d.project }}项目的虚拟机{{ $d.name }}崩溃了`
	SERVER_PANICKED_TITLE_EN = `{{- $d := .resource_details -}}
	{{ .resource_type }} {{ $d.task_name }} PANIC`
	SERVER_PANICKED_CONTENT_CN = `{{- $d := .resource_details -}}
	您在{{ $d.project }}项目的虚拟机{{ $d.name }}崩溃了`
	SERVER_PANICKED_CONTENT_EN = `{{- $d := .resource_details -}}
	The server {{ $d.name }} in project {{ $d.project }} panicked.`
)

// 资源定时调度任务通知
const (
	SCHEDULEDTASK_EXECUTE_TITLE_CN = `{{- $d := .resource_details -}}
	您在{{ $d.project }}项目的定时任务执行成功`
	SCHEDULEDTASK_EXECUTE_TITLE_EN = `{{- $d := .resource_details -}}
	The scheduled task in {{ $d.project }} execute successfully`
	SCHEDULEDTASK_EXECUTE_CONTENT_CN = `{{- $d := .resource_details -}}
	您在{{ $d.project }}项目的定时任务执行成功`
	SCHEDULEDTASK_EXECUTE_CONTENT_EN = `{{- $d := .resource_details -}}
	The scheduled task in {{ $d.project }} successfully {{ $d.operation_display }} the {{ $d.resource_type_display }} {{ $d.resource_name }}`
)

// 服务异常通知
const (
	SERVICE_ABNORMAL_TITLE_CN = `{{- $d := .resource_details -}}
	服务{{ $d.service_name }}异常`
	SERVICE_ABNORMAL_TITLE_EN = `{{- $d := .resource_details -}}
	Server {{ $d.service_name }} abnormal`
	SERVICE_ABNORMAL_CONTENT_CN = `{{- $d := .resource_details -}}
	服务: {{ $d.service_name }} 异常。
	方法: {{ $d.method }}
	路径: {{ $d.path }} 
	{{- if $d.body }}
	请求: {{ $d.body -}}
	{{ end }}
	错误: {{ $d.error }}`
	SERVICE_ABNORMAL_CONTENT_EN = `{{- $d := .resource_details -}}
	Service: {{ $d.service_name }} abnormal
	Method: {{ $d.method }}
	Path: {{ $d.path }} 
	{{- if $d.body }}
	Body: {{ $d.body -}}
	{{ end }}
	Error: {{ $d.error }}`
)

// 弹性伸缩组策略生效通知
const (
	SCALINGPOLICY_EXECUTE_TITLE_CN = `{{- $d := .resource_details -}}
	{{ $d.project }}项目的弹性伸缩组{{ $d.scaling_group }}中的伸缩策略{{ $d.name }}满足触发条件`
	SCALINGPOLICY_EXECUTE_TITLE_EN = `{{- $d := .resource_details -}}
	The scaling policy {{ $d.name }} in the scaling group {{ $d.scaling_group }} triggered`
	SCALINGPOLICY_EXECUTE_CONTENT_CN = `{{- $d := .resource_details -}}
	您在{{ $d.project }}项目的弹性伸缩组{{ $d.scaling_group }}中的{{ $d.trigger_type_display }}类型伸缩策略{{ $d.name }}满足触发条件，成功{{ $d.action_display }}{{ $d.number }}{{ $d.unit_display }}实例`
	SCALINGPOLICY_EXECUTE_CONTENT_EN = `{{- $d := .resource_details -}}
	The {{ $d.trigger_type_display }} type scaling policy {{ $d.name }} in the scaling group {{ $d.scaling_group }} of the {{ $d.project }} project satisfies the trigger conditions, and {{ $d.action_display }} {{ $d.number }}{{ $d.unit_display }} instances successfully`
)

// 自动快照策略生效通知
const (
	SNAPSHOTPOLICY_EXECUTE_TITLE_CN = `{{- $d := .resource_details -}}
	{{ $d.project }}项目的自动快照策略{{ $d.name }}生效`
	SNAPSHOTPOLICY_EXECUTE_TITLE_EN = `{{- $d := .resource_details -}}
	The snapshot policy {{ $d.name }} in the {{ $d.project }} project executed`
	SNAPSHOTPOLICY_EXECUTE_CONTENT_CN = `{{- $d := .resource_details -}}
	您在{{ $d.project }}项目的自动快照策略{{ $d.name }}为硬盘{{ $d.disk }}创建快照成功`
	SNAPSHOTPOLICY_EXECUTE_CONTENT_EN = `{{- $d := .resource_details -}}
	The snapshot policy {{ $d.name }} in the {{ $d.project }} project successfully creates a snapshot for the disk {{ $d.disk }}`
)

// 资源变更通知
const (
	UPDATE_TITLE_CN = `{{- $d := .resource_details -}}
	if {{ $d.project }}
	{{ $d.project }}项目的
	{{- end -}}
	{{ .resource_type_display }}{{ $d.name }}{{ .action_display }}成功`
	UPDATE_TITLE_EN = `{{- $d := .resource_details -}}
	The {{ .resource_type }} {{ $d.name }} {{ if $d.project -}} in project {{ $d.project }} {{ end -}} {{ .action_display }} successfully`
	UPDATE_CONTENT_CN = `{{- $d := .resource_details -}}
	您
	if {{$d.project }}
	在{{ $d.project }}项目
	{{- end -}}
	{{- if $d.brand -}}
	{{ $d.brand }}平台
	{{- end -}}
	的{{ .resource_type_display }}{{ $d.name }}{{ .action_display }}成功
	{{- if $d.account -}}
	，帐号为{{ $d.account }}
	{{- end -}}
	
	{{- if $d.password -}}
	，密码为{{ $d.password }}
	{{- end -}}
	，更多信息请前往控制台进行查看`
	UPDATE_CONTENT_EN = `{{- $d := .resource_details -}}
	Your {{ if $d.brand -}} {{ $d.brand }} {{ end -}} {{ .resource_type }} {{ $d.name }} {{ if $d.project -}} in project {{ $d.project }} {{ end -}} has been {{ .action_display }} successfully
	{{- if $d.account -}}
	, the acount is {{ $d.account }}
	{{- end -}}
	
	{{- if $d.password -}}
	, the password is {{ $d.password }}
	{{- end -}}
	, and please go to the console to view more information`
)

// 用户锁定通知
const (
	USER_LOCK_TITLE_CN = `{{- $d := .resource_details -}}
	账号{{ $d.name }}已被锁定`
	USER_LOCK_TITLE_EN = `{{- $d := .resource_details -}}
	Account {{ $d.name }} has been locked`
	USER_LOCK_CONTENT_CN = `{{- $d := .resource_details -}}
	账号{{ $d.name }}由于异常登录已被锁定，请核实情况，如果需要为用户解锁，请到用户列表启用该用户。`
	USER_LOCK_CONTENT_EN = `{{- $d := .resource_details -}}
	The account {{ $d.name }} has been locked due to abnormal login. Please verify the situation. If you need to unlock the user, please go to the user list to enable the user.`
)

// 云账号状态异常通知
const (
	SYNC_ACCOUNT_STATUS_TITLE_CN = `{{- $d := .resource_details -}}
	云账号{{ $d.name}}状态异常`
	SYNC_ACCOUNT_STATUS_TITLE_EN = `{{- $d := .resource_details -}}
	The account {{ $d.name }} status is abnormal`
	SYNC_ACCOUNT_STATUS_CONTENT_CN = `{{- $d := .resource_details -}}
	云账号{{ $d.name }}状态异常，请及时检查。`
	SYNC_ACCOUNT_STATUS_CONTENT_EN = `{{- $d := .resource_details -}}
	The account {{ $d.name }} status is abnormal. Please check in time.`
)

// work阻塞通知
const (
	WORK_BLOCK_TITLE_CN = `{{- $d := .resource_details -}}
	服务{{ d.service_name}} worker阻塞半小时，请及时检查。`
	WORK_BLOCK_TITLE_EN = `{{- $d := .resource_details -}}
	The service: {{ d.service_name}} worker has been block 30 minutes.Please verify the service in time.`
	WORK_BLOCK_CONTENT_CN = `{{- $d := .resource_details -}}
	服务{{ d.service_name}} worker阻塞半小时，请及时检查。`
	WORK_BLOCK_CONTENT_EN = `{{- $d := .resource_details -}}
	The service: {{ d.service_name}} worker has been block 30 minutes.Please verify the service in time.`
)
