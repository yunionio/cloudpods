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

package wechat

import (
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/keystone/driver/oauth2"
)

type SWechatDriverFactory struct{}

func (drv SWechatDriverFactory) NewDriver(appId string, secret string) oauth2.IOAuth2Driver {
	return NewWechatOAuth2Driver(appId, secret)
}

func (drv SWechatDriverFactory) TemplateName() string {
	return api.IdpTemplateWechat
}

func (drv SWechatDriverFactory) IdpAttributeOptions() api.SIdpAttributeOptions {
	return api.SIdpAttributeOptions{
		UserNameAttribute:        "name",
		UserIdAttribute:          "user_id",
		UserDisplaynameAttribtue: "name",
	}
}

func (drv SWechatDriverFactory) ValidateConfig(conf api.SOAuth2IdpConfigOptions) error {
	return nil
}

func init() {
	oauth2.Register(&SWechatDriverFactory{})
}
