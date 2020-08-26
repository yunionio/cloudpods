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

package qywechat

import (
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/driver/oauth2"
)

type SQywxDriverFactory struct{}

func (drv SQywxDriverFactory) NewDriver(appId string, secret string) oauth2.IOAuth2Driver {
	return NewQywxOAuth2Driver(appId, secret)
}

func (drv SQywxDriverFactory) TemplateName() string {
	return api.IdpTemplateQywechat
}

func (drv SQywxDriverFactory) IdpAttributeOptions() api.SIdpAttributeOptions {
	return api.SIdpAttributeOptions{
		UserNameAttribute:        "name",
		UserIdAttribute:          "user_id",
		UserDisplaynameAttribtue: "displayname",
		UserEmailAttribute:       "email",
		UserMobileAttribute:      "mobile",
	}
}

func (drv SQywxDriverFactory) ValidateConfig(conf api.SOAuth2IdpConfigOptions) error {
	cid, aid, err := splitAppId(conf.AppId)
	if err != nil {
		return err
	}
	if len(cid) == 0 {
		return errors.Wrap(httperrors.ErrInputParameter, "empty corp_id")
	}
	if len(aid) == 0 {
		return errors.Wrap(httperrors.ErrInputParameter, "empty agent_id")
	}
	return nil
}

func init() {
	oauth2.Register(&SQywxDriverFactory{})
}
