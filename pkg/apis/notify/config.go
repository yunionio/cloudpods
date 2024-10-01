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
	ContactType      string
	Topic            string
	Message          string
	Event            SNotifyEvent
	AdvanceDays      int
	RobotUseTemplate bool
}

type SBatchSendParams struct {
	ContactType string
	Contacts    []string
	Topic       string
	Message     string
	Priority    string
	Lang        string
}

type SNotifyReceiver struct {
	Contact  string
	DomainId string
	Enabled  bool
	Lang     string

	callback func(error)
}

func (self *SNotifyReceiver) Callback(err error) {
	if self.callback != nil && err != nil {
		self.callback(err)
	}
}

type SSendParams struct {
	ContactType    string
	Contact        string
	Topic          string
	Message        string
	Priority       string
	Title          string
	RemoteTemplate string
	Lang           string
	Receiver       SNotifyReceiver
}

type SNotificationGroupSearchInput struct {
	StartTime   time.Time
	EndTime     time.Time
	GroupKey    string
	ReceiverId  string
	ContactType string
}

type SendParams struct {
	Title               string
	Message             string
	Priority            string
	RemoteTemplate      string
	Topic               string
	Event               string
	Receivers           SNotifyReceiver
	EmailMsg            SEmailMessage
	Header              jsonutils.JSONObject
	Body                jsonutils.JSONObject
	MsgKey              string
	DomainId            string
	RemoteTemplateParam SRemoteTemplateParam
	GroupKey            string
	// minutes
	GroupTimes uint
	ReceiverId string
	SendTime   time.Time
}

type SRemoteTemplateParam struct {
	Code      string `json:"code"`
	Domain    string `json:"domain"`
	User      string `json:"user"`
	Type      string `json:"type"`
	AlertName string `json:"alert_name"`
}

type SSMSSendParams struct {
	RemoteTemplate      string
	RemoteTemplateParam SRemoteTemplateParam

	AppKey        string
	AppSecret     string
	From          string
	To            string
	TemplateId    string
	TemplateParas string
	Signature     string
}

type NotifyConfig struct {
	SNotifyConfigContent
	Attribution string
	DomainId    string
}

func (self *NotifyConfig) GetDomainId() string {
	if self.Attribution == CONFIG_ATTRIBUTION_SYSTEM {
		return CONFIG_ATTRIBUTION_SYSTEM
	}
	return self.DomainId
}

type SNotifyConfigContent struct {
	// Email
	Hostname      string
	Hostport      int
	Password      string
	SslGlobal     bool
	Username      string
	SenderAddress string
	//Lark
	AppId       string
	AppSecret   string
	AccessToken string
	// workwx
	AgentId string
	CorpId  string
	Secret  string
	// dingtalk
	//AgentId string
	//AppSecret string
	AppKey string
	// sms
	VerifiyCode     string
	AlertsCode      string
	ErrorCode       string
	PhoneNumber     string
	AccessKeyId     string
	AccessKeySecret string
	ServiceUrl      string
	Signature       string
	SmsDriver       string
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
