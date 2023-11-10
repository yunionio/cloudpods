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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type DataSourceCreateOptions struct {
	NAME string
}

type DataSourceListOptions struct {
	options.BaseListOptions
}

func (d DataSourceListOptions) Params() (jsonutils.JSONObject, error) {
	return d.BaseListOptions.Params()
}

type DataSourceDeleteOptions struct {
	ID string `json:"-"`
}

type NotificationListOptions struct {
	options.BaseListOptions
}

type NotificationShowOptions struct {
	ID string `help:"ID or name of the alert notification config" json:"-"`
}

type NotificationDeleteOptions struct {
	ID []string `help:"ID or name of the alert notification config" json:"-"`
}

type NotificationFields struct {
	Frequency             string `help:"notify frequency, e.g. 5m, 1h"`
	IsDefault             *bool  `help:"set as default notification"`
	DisableResolveMessage *bool  `help:"disable notify recover message"`
	SendReminder          *bool  `help:"send reminder"`
}

type NotificationCreateOptions struct {
	NAME string `help:"notification config name"`
	NotificationFields
}

func (opt NotificationCreateOptions) Params() (*monitor.NotificationCreateInput, error) {
	ret := &monitor.NotificationCreateInput{
		Name:                  opt.NAME,
		SendReminder:          opt.SendReminder,
		DisableResolveMessage: opt.DisableResolveMessage,
	}
	if opt.IsDefault != nil && *opt.IsDefault {
		ret.IsDefault = true
	}
	return ret, nil
}

type NotificationDingDingCreateOptions struct {
	NotificationCreateOptions
	URL     string `help:"dingding webhook url"`
	MsgType string `help:"message type" choices:"markdown|actionCard" default:"markdown"`
}

func (opt NotificationDingDingCreateOptions) Params() (*monitor.NotificationCreateInput, error) {
	out, err := opt.NotificationCreateOptions.Params()
	if err != nil {
		return nil, err
	}
	out.Type = monitor.AlertNotificationTypeDingding
	out.Settings = jsonutils.Marshal(monitor.NotificationSettingDingding{
		Url:         opt.URL,
		MessageType: opt.MsgType,
	})
	return out, nil
}

type NotificationFeishuCreateOptions struct {
	NotificationCreateOptions
	APPID     string `help:"feishu robot appId"`
	APPSECRET string `help:"feishu robt appSecret"`
}

func (opt NotificationFeishuCreateOptions) Params() (*monitor.NotificationCreateInput, error) {
	out, err := opt.NotificationCreateOptions.Params()
	if err != nil {
		return nil, err
	}
	out.Type = monitor.AlertNotificationTypeFeishu
	out.Settings = jsonutils.Marshal(monitor.NotificationSettingFeishu{
		AppId:     opt.APPID,
		AppSecret: opt.APPSECRET,
	})
	return out, nil
}

type NotificationUpdateOptions struct {
	NotificationFields

	ID                  string `help:"ID or name of the alert notification config" json:"-"`
	DisableDefault      *bool  `help:"disable as default notification" json:"-"`
	ResolveMessage      *bool  `help:"enable notify recover message" json:"-"`
	DisableSendReminder *bool  `help:"disable send reminder" json:"-"`
}

func (opt NotificationUpdateOptions) Params() (*monitor.NotificationUpdateInput, error) {
	if opt.DisableDefault != nil && *opt.DisableDefault {
		tmp := false
		opt.IsDefault = &tmp
	}
	if opt.ResolveMessage != nil && *opt.ResolveMessage {
		tmp := false
		opt.DisableDefault = &tmp
	}
	if opt.DisableSendReminder != nil && *opt.DisableSendReminder {
		tmp := false
		opt.SendReminder = &tmp
	}
	ret := &monitor.NotificationUpdateInput{
		IsDefault:             opt.IsDefault,
		DisableResolveMessage: opt.DisableResolveMessage,
		SendReminder:          opt.SendReminder,
	}
	return ret, nil
}
