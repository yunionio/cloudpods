// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package sender

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/monitor/notifydrivers/feishu"
	"yunion.io/x/onecloud/pkg/notify/models"
)

var ErrNoSuchWebhook = errors.Error("No such webhook")
var InvalidWebhook = errors.Error("Invalid webhook")

type SFeishuRobotSender struct {
	config map[string]api.SNotifyConfigContent
}

func (feishuRobotSender *SFeishuRobotSender) GetSenderType() string {
	return api.FEISHU_ROBOT
}

func (feishuRobotSender *SFeishuRobotSender) Send(ctx context.Context, args api.SendParams) error {
	var token string
	var errs []error
	title, msg := args.Title, args.Message
	webhook := args.Receivers.Contact
	switch {
	case strings.HasPrefix(webhook, ApiWebhookRobotV2SendMessage):
		token = webhook[len(ApiWebhookRobotV2SendMessage):]
	case strings.HasPrefix(webhook, feishu.ApiWebhookRobotSendMessage):
		token = webhook[len(feishu.ApiWebhookRobotSendMessage):]
	default:
		return errors.Wrap(InvalidWebhook, webhook)
	}
	req := feishu.WebhookRobotMsgReq{
		Title: title,
		Text:  msg,
	}
	rep, err := feishu.SendWebhookRobotMessage(token, req)
	if err != nil {
		return errors.Wrap(err, "SendWebhookRobotMessage")
	}
	if !rep.Ok {
		if strings.Contains(rep.Error, "token") {
			return ErrNoSuchWebhook
		} else {
			return fmt.Errorf("SendWebhookRobotMessage failed: %s", rep.Error)
		}
	}
	if err != nil {
		if errs == nil {
			errs = []error{}
		}
		errs = append(errs, err)
	}
	return errors.NewAggregate(errs)
}

func (feishuRobotSender *SFeishuRobotSender) ValidateConfig(ctx context.Context, config api.NotifyConfig) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (feishuRobotSender *SFeishuRobotSender) ContactByMobile(ctx context.Context, mobile, domainId string) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (feishuRobotSender *SFeishuRobotSender) IsPersonal() bool {
	return true
}

func (feishuRobotSender *SFeishuRobotSender) IsRobot() bool {
	return true
}

func (feishuRobotSender *SFeishuRobotSender) IsValid() bool {
	return len(feishuRobotSender.config) > 0
}

func (feishuRobotSender *SFeishuRobotSender) IsPullType() bool {
	return true
}

func (feishuRobotSender *SFeishuRobotSender) IsSystemConfigContactType() bool {
	return true
}

func (feishuRobotSender *SFeishuRobotSender) GetAccessToken(ctx context.Context, key string) error {
	return nil
}

func (feishuRobotSender *SFeishuRobotSender) RegisterConfig(config models.SConfig) {
}

func init() {
	models.Register(&SFeishuRobotSender{
		config: map[string]api.SNotifyConfigContent{},
	})
}
