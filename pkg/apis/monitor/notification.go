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

package monitor

import (
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

type AlertNotificationStateType string

var (
	AlertNotificationStatePending   = AlertNotificationStateType("pending")
	AlertNotificationStateCompleted = AlertNotificationStateType("completed")
	AlertNotificationStateUnknown   = AlertNotificationStateType("unknown")
)

const (
	AlertNotificationTypeOneCloud      = "onecloud"
	AlertNotificationTypeDingding      = "dingding"
	AlertNotificationTypeFeishu        = "feishu"
	AlertNotificationTypeAutoScaling   = "autoscaling"
	AlertNotificationTypeAutoMigration = "automigration"
)

type NotificationCreateInput struct {
	apis.Meta

	// 报警通知名称
	Name string `json:"name"`
	// 类型
	Type string `json:"type"`
	// 是否为默认通知配置
	IsDefault bool `json:"is_default"`
	// 是否一直提醒
	SendReminder *bool `json:"send_reminder"`
	// 是否禁用报警恢复提醒
	DisableResolveMessage *bool `json:"disable_resolve_message"`
	// 发送频率 单位：s
	Frequency time.Duration `json:"frequency"`
	// 通知配置
	Settings jsonutils.JSONObject `json:"settings"`
}

type NotificationUpdateInput struct {
	apis.Meta

	// 报警通知名称
	Name string `json:"name"`
	// 是否为默认通知配置
	IsDefault *bool `json:"is_default"`
	// 是否一直提醒
	SendReminder *bool `json:"send_reminder"`
	// 是否禁用报警恢复提醒
	DisableResolveMessage *bool `json:"disable_resolve_message"`
	// 发送频率
	Frequency *time.Duration `json:"frequency"`
}

type NotificationListInput struct {
	apis.VirtualResourceListInput
	// 类型
	Type string `json:"type"`
}

type NotificationSettingOneCloud struct {
	Channel  string   `json:"channel"`
	UserIds  []string `json:"user_ids"`
	RobotIds []string `json:"robot_ids"`
}

type SendWebhookSync struct {
	Url         string
	User        string
	Password    string
	Body        string
	HttpMethod  string
	HttpHeader  map[string]string
	ContentType string
}

type NotificationSettingDingding struct {
	Url         string `json:"url"`
	MessageType string `json:"message_type"`
}

type NotificationSettingFeishu struct {
	// Url         string `json:"url"`
	AppId     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

type NotificationSettingAutoMigration struct {
	MigrationAlertSettings
}
