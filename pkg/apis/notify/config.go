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
	"reflect"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"

	"yunion.io/x/onecloud/pkg/apis"
)

type ConfigCreateInput struct {
	apis.DomainLevelResourceCreateInput

	// description: config type
	// required: true
	// example: feishu
	Type string `json:"type"`

	// description: config content
	// required: true
	// example: {"app_id": "123456", "app_secret": "feishu_nihao"}
	Content *SNotifyConfigContent `json:"content"`

	// description: attribution
	// required: true
	// enum: ["system","domain"]
	// example: system
	Attribution string `json:"attribution"`
}

type ConfigUpdateInput struct {
	// description: config content
	// required: true
	// example: {"app_id": "123456", "app_secret": "feishu_nihao"}
	Content *SNotifyConfigContent `json:"content"`
}

type ConfigDetails struct {
	apis.DomainLevelResourceDetails

	SConfig
}

type ConfigListInput struct {
	apis.DomainLevelResourceListInput
	Type        string `json:"type"`
	Attribution string `json:"attribution"`
}

type ConfigValidateInput struct {
	// description: config type
	// required: true
	// example: feishu
	Type string `json:"type"`

	// description: config content
	// required: true
	// example: {"app_id": "123456", "app_secret": "feishu_nihao"}
	Content *SNotifyConfigContent `json:"content"`
}

type ConfigValidateOutput struct {
	IsValid bool   `json:"is_valid"`
	Message string `json:"message"`
}

type ConfigManagerGetTypesInput struct {
	// description: View the available notification channels for the receivers
	// required: false
	Receivers []string `json:"receivers"`
	// description: View the available notification channels for the domains with these DomainIds
	// required: false
	DomainIds []string `json:"domain_ids"`
	// description: Operation of reduce
	// required: false
	// enum: ["union","merge"]
	Operation string `json:"operation"`
}

type ConfigManagerGetTypesOutput struct {
	Types []string `json:"types"`
}

type SsNotification struct {
	ContactType      string       `json:"contact_type"`
	Topic            string       `json:"topic"`
	Message          string       `json:"message"`
	Event            SNotifyEvent `json:"event"`
	AdvanceDays      int          `json:"advance_days"`
	RobotUseTemplate bool         `json:"robot_use_template"`
}

type SBatchSendParams struct {
	ContactType string   `json:"contact_type"`
	Contacts    []string `json:"contacts"`
	Topic       string   `json:"topic"`
	Message     string   `json:"message"`
	Priority    string   `json:"priority"`
	Lang        string   `json:"lang"`
}

type SNotifyReceiver struct {
	Contact  string `json:"contact"`
	DomainId string `json:"domain_id"`
	Enabled  bool   `json:"enabled"`
	Lang     string `json:"lang"`

	callback func(error)
}

func (self *SNotifyReceiver) Callback(err error) {
	if self.callback != nil && err != nil {
		self.callback(err)
	}
}

type SSendParams struct {
	ContactType    string          `json:"contact_type"`
	Contact        string          `json:"contact"`
	Topic          string          `json:"topic"`
	Message        string          `json:"message"`
	Priority       string          `json:"priority"`
	Title          string          `json:"title"`
	RemoteTemplate string          `json:"remote_template"`
	Lang           string          `json:"lang"`
	Receiver       SNotifyReceiver `json:"receiver"`
}

type SNotificationGroupSearchInput struct {
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	GroupKey    string    `json:"group_key"`
	ReceiverId  string    `json:"receiver_id"`
	ContactType string    `json:"contact_type"`
}

type SendParams struct {
	Title               string               `json:"title"`
	Message             string               `json:"message"`
	Priority            string               `json:"priority"`
	RemoteTemplate      string               `json:"remote_template"`
	Topic               string               `json:"topic"`
	Event               string               `json:"event"`
	Receivers           SNotifyReceiver      `json:"receivers"`
	EmailMsg            SEmailMessage        `json:"email_msg"`
	Header              jsonutils.JSONObject `json:"header"`
	Body                jsonutils.JSONObject `json:"body"`
	MsgKey              string               `json:"msg_key"`
	DomainId            string               `json:"domain_id"`
	RemoteTemplateParam SRemoteTemplateParam `json:"remote_template_param"`
	GroupKey            string               `json:"group_key"`
	// minutes
	GroupTimes uint      `json:"group_times"`
	ReceiverId string    `json:"receiver_id"`
	SendTime   time.Time `json:"send_time"`
}

type SRemoteTemplateParam struct {
	Code      string `json:"code"`
	Domain    string `json:"domain"`
	User      string `json:"user"`
	Type      string `json:"type"`
	AlertName string `json:"alert_name"`
}

type SSMSSendParams struct {
	RemoteTemplate      string               `json:"remote_template"`
	RemoteTemplateParam SRemoteTemplateParam `json:"remote_template_param"`

	AppKey        string `json:"app_key"`
	AppSecret     string `json:"app_secret"`
	From          string `json:"from"`
	To            string `json:"to"`
	TemplateId    string `json:"template_id"`
	TemplateParas string `json:"template_paras"`
	Signature     string `json:"signature"`
}

type NotifyConfig struct {
	SNotifyConfigContent
	Attribution string `json:"attribution"`
	DomainId    string `json:"domain_id"`
}

func (self *NotifyConfig) GetDomainId() string {
	if self.Attribution == CONFIG_ATTRIBUTION_SYSTEM {
		return CONFIG_ATTRIBUTION_SYSTEM
	}
	return self.DomainId
}

type SNotifyConfigContent struct {
	// Email
	Hostname      string `json:"hostname"`
	Hostport      int    `json:"hostport"`
	Password      string `json:"password"`
	SslGlobal     bool   `json:"ssl_global"`
	Username      string `json:"username"`
	SenderAddress string `json:"sender_address"`
	//Lark
	AppId                 string    `json:"app_id"`
	AppSecret             string    `json:"app_secret"`
	AccessToken           string    `json:"access_token"`
	AccessTokenExpireTime time.Time `json:"access_token_expire_time"`
	// workwx
	AgentId string `json:"agent_id"`
	CorpId  string `json:"corp_id"`
	Secret  string `json:"secret"`
	// dingtalk
	//AgentId string
	//AppSecret string
	AppKey string `json:"app_key"`
	// sms
	VerifiyCode     string `json:"verifiy_code"`
	AlertsCode      string `json:"alerts_code"`
	ErrorCode       string `json:"error_code"`
	PhoneNumber     string `json:"phone_number"`
	AccessKeyId     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	ServiceUrl      string `json:"service_url"`
	Signature       string `json:"signature"`
	SmsDriver       string `json:"sms_driver"`
}

func (self SNotifyConfigContent) String() string {
	return jsonutils.Marshal(self).String()
}

func (self SNotifyConfigContent) IsZero() bool {
	return jsonutils.Marshal(self).Equals(jsonutils.Marshal(SNotifyConfigContent{}))
}

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&SNotifyConfigContent{}), func() gotypes.ISerializable {
		return &SNotifyConfigContent{}
	})
}
