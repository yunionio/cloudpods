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

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/notify/models"
)

const (
	ApiWebhookRobotV2SendMessage = "https://open.feishu.cn/open-apis/bot/v2/hook/"
)

type SFeishuSender struct {
	config map[string]api.SNotifyConfigContent
}

func (self *SFeishuSender) GetSenderType() string {
	return api.FEISHU
}

func (self *SFeishuSender) Send(ctx context.Context, args api.SendParams) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SFeishuSender) ValidateConfig(ctx context.Context, config api.NotifyConfig) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SFeishuSender) UpdateConfig(ctx context.Context, config api.NotifyConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SFeishuSender) AddConfig(ctx context.Context, config api.NotifyConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SFeishuSender) DeleteConfig(ctx context.Context, config api.NotifyConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SFeishuSender) ContactByMobile(ctx context.Context, mobile, domainId string) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SFeishuSender) IsPersonal() bool {
	return true
}

func (self *SFeishuSender) IsRobot() bool {
	return true
}

func (self *SFeishuSender) IsValid() bool {
	return len(self.config) > 0
}

func (self *SFeishuSender) IsPullType() bool {
	return true
}

func (self *SFeishuSender) IsSystemConfigContactType() bool {
	return true
}

func init() {
	models.Register(&SFeishuSender{
		config: map[string]api.SNotifyConfigContent{},
	})
}
