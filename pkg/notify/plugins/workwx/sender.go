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

package workwx

import (
	"context"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/notify/models"
)

type SWorkWxSender struct {
	config map[string]api.SNotifyConfigContent
}

func (self *SWorkWxSender) GetSenderType() string {
	return api.FEISHU
}

func (self *SWorkWxSender) Send(ctx context.Context, args api.SendParams) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SWorkWxSender) ValidateConfig(ctx context.Context, config api.NotifyConfig) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SWorkWxSender) UpdateConfig(ctx context.Context, config api.NotifyConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SWorkWxSender) AddConfig(ctx context.Context, config api.NotifyConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SWorkWxSender) DeleteConfig(ctx context.Context, config api.NotifyConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SWorkWxSender) ContactByMobile(ctx context.Context, mobile, domainId string) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SWorkWxSender) IsPersonal() bool {
	return true
}

func (self *SWorkWxSender) IsRobot() bool {
	return true
}

func (self *SWorkWxSender) IsValid() bool {
	return len(self.config) > 0
}

func (self *SWorkWxSender) IsPullType() bool {
	return true
}

func init() {
	models.Register(&SWorkWxSender{
		config: map[string]api.SNotifyConfigContent{},
	})
}
