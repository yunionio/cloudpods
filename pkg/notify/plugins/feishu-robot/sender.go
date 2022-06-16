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

package feishu

import (
	"context"
	"fmt"
	"strings"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/monitor/notifydrivers/feishu"
	"yunion.io/x/onecloud/pkg/notify/models"
	"yunion.io/x/pkg/errors"
)

const (
	ApiWebhookRobotV2SendMessage = "https://open.feishu.cn/open-apis/bot/v2/hook/"
)

type SFeishuRobotSender struct {
	config map[string]api.SNotifyConfigContent
}

func (self *SFeishuRobotSender) GetSenderType() string {
	return api.FEISHU_ROBOT
}

func (self *SFeishuRobotSender) Send(ctx context.Context, args api.SendParams) error {
	errs := []error{}
	for _, recv := range args.Receivers {
		var token string
		switch {
		case strings.HasPrefix(recv.Contact, ApiWebhookRobotV2SendMessage):
			token = recv.Contact[len(ApiWebhookRobotV2SendMessage):]
		case strings.HasPrefix(recv.Contact, feishu.ApiWebhookRobotSendMessage):
			token = recv.Contact[len(feishu.ApiWebhookRobotSendMessage):]
		default:
			errs = append(errs, fmt.Errorf("invalid address: %s", recv.Contact))
			continue
		}
		req := feishu.WebhookRobotMsgReq{
			Title: args.Title,
			Text:  args.Message,
		}
		resp, err := feishu.SendWebhookRobotMessage(token, req)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "SendWebhookRobotMessage"))
			continue
		}
		if !resp.Ok {
			errs = append(errs, fmt.Errorf(resp.Error))
		}
	}
	return errors.NewAggregate(errs)
}

func (self *SFeishuRobotSender) ValidateConfig(ctx context.Context, config api.NotifyConfig) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SFeishuRobotSender) UpdateConfig(ctx context.Context, config api.NotifyConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SFeishuRobotSender) AddConfig(ctx context.Context, config api.NotifyConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SFeishuRobotSender) DeleteConfig(ctx context.Context, config api.NotifyConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SFeishuRobotSender) ContactByMobile(ctx context.Context, mobile, domainId string) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SFeishuRobotSender) IsPersonal() bool {
	return false
}

func (self *SFeishuRobotSender) IsRobot() bool {
	return true
}

func (self *SFeishuRobotSender) IsValid() bool {
	return len(self.config) > 0
}

func init() {
	models.Register(&SFeishuRobotSender{
		config: map[string]api.SNotifyConfigContent{},
	})
}
