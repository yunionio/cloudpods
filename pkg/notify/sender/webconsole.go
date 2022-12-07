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
package sender

import (
	"yunion.io/x/cloudmux/pkg/cloudprovider"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/notify/models"
)

type SWebconsoleSender struct {
	config map[string]api.SNotifyConfigContent
}

func (self *SWebconsoleSender) GetSenderType() string {
	return api.WEBCONSOLE
}

func (self *SWebconsoleSender) Send(args api.SendParams) error {
	// var token string
	// var errs []error
	// title, msg := args.Title, args.Message
	// // for _, recevier := range args.Receivers {
	// webhook := args.Receivers.Contact
	// switch {
	// case strings.HasPrefix(webhook, ApiWebhookRobotV2SendMessage):
	// 	token = webhook[len(ApiWebhookRobotV2SendMessage):]
	// case strings.HasPrefix(webhook, feishu.ApiWebhookRobotSendMessage):
	// 	token = webhook[len(feishu.ApiWebhookRobotSendMessage):]
	// default:
	// 	return errors.Wrap(InvalidWebhook, webhook)
	// }
	// req := feishu.WebhookRobotMsgReq{
	// 	Title: title,
	// 	Text:  msg,
	// }
	// rep, err := feishu.SendWebhookRobotMessage(token, req)
	// if err != nil {
	// 	return errors.Wrap(err, "SendWebhookRobotMessage")
	// }
	// if !rep.Ok {
	// 	if strings.Contains(rep.Error, "token") {
	// 		return ErrNoSuchWebhook
	// 	} else {
	// 		return fmt.Errorf("SendWebhookRobotMessage failed: %s", rep.Error)
	// 	}
	// }
	// if err != nil {
	// 	if errs == nil {
	// 		errs = []error{}
	// 	}
	// 	errs = append(errs, err)
	// }
	// return errors.NewAggregate(errs)
	return nil
}

func (websender *SWebconsoleSender) ValidateConfig(config api.NotifyConfig) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (websender *SWebconsoleSender) UpdateConfig(config api.NotifyConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (websender *SWebconsoleSender) AddConfig(config api.NotifyConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (websender *SWebconsoleSender) DeleteConfig(config api.NotifyConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (websender *SWebconsoleSender) ContactByMobile(mobile, domainId string) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (websender *SWebconsoleSender) IsPersonal() bool {
	return true
}

func (websender *SWebconsoleSender) IsRobot() bool {
	return true
}

func (websender *SWebconsoleSender) IsValid() bool {
	return len(websender.config) > 0
}

func (websender *SWebconsoleSender) IsPullType() bool {
	return true
}

func (websender *SWebconsoleSender) IsSystemConfigContactType() bool {
	return true
}

func (websender *SWebconsoleSender) GetAccessToken() error {
	return nil
}

func init() {
	models.Register(&SWebconsoleSender{
		config: map[string]api.SNotifyConfigContent{},
	})
}
