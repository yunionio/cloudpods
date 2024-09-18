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

func (self *SWebconsoleSender) Send(ctx context.Context, args api.SendParams) error {
	return nil
}

func (websender *SWebconsoleSender) ValidateConfig(ctx context.Context, config api.NotifyConfig) (string, error) {
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

func (websender *SWebconsoleSender) ContactByMobile(ctx context.Context, mobile, domainId string) (string, error) {
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

func (websender *SWebconsoleSender) GetAccessToken(ctx context.Context, key string) error {
	return nil
}

func (websender *SWebconsoleSender) RegisterConfig(config models.SConfig) {
}

func init() {
	models.Register(&SWebconsoleSender{
		config: map[string]api.SNotifyConfigContent{},
	})
}
