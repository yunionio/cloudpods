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
	"net/http"
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/notify/models"
)

type SWebhookSender struct {
	config map[string]api.SNotifyConfigContent
}

func (self *SWebhookSender) GetSenderType() string {
	return api.WEBHOOK_ROBOT
}

func (self *SWebhookSender) Send(args api.SendParams) error {
	log.Infoln("this is in webhookSend ")
	body, err := jsonutils.ParseString(args.Message)
	if err != nil {
		log.Errorf("unable to parse %q: %v", args.Message, err)
	}
	if _, ok := body.(*jsonutils.JSONString); err != nil || ok {
		dict := jsonutils.NewDict()
		dict.Set("Msg", jsonutils.NewString(args.Message))
		body = dict
	}
	event := strings.ToUpper(args.Event)
	header := http.Header{}
	header.Set(EVENT_HEADER, event)
	_, _, err = httputils.JSONRequest(cli, ctx, httputils.POST, args.Receivers.Contact, header, body, true)
	return err
}

func (websender *SWebhookSender) ValidateConfig(config api.NotifyConfig) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (websender *SWebhookSender) UpdateConfig(config api.NotifyConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (websender *SWebhookSender) AddConfig(config api.NotifyConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (websender *SWebhookSender) DeleteConfig(config api.NotifyConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (websender *SWebhookSender) ContactByMobile(mobile, domainId string) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (websender *SWebhookSender) IsPersonal() bool {
	return true
}

func (websender *SWebhookSender) IsRobot() bool {
	return true
}

func (websender *SWebhookSender) IsValid() bool {
	return len(websender.config) > 0
}

func (websender *SWebhookSender) IsPullType() bool {
	return true
}

func (websender *SWebhookSender) IsSystemConfigContactType() bool {
	return true
}

func (websender *SWebhookSender) GetAccessToken() error {
	return nil
}

func init() {
	models.Register(&SWebhookSender{
		config: map[string]api.SNotifyConfigContent{},
	})
}
