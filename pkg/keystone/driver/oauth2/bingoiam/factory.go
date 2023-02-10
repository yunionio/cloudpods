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

package bingoiam

import (
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/keystone/driver/oauth2"
)

type SBingoIAMDriverFactory struct{}

func (drv SBingoIAMDriverFactory) NewDriver(appId string, secret string) oauth2.IOAuth2Driver {
	return NewBingoIAMOAuth2Driver(appId, secret)
}

func (drv SBingoIAMDriverFactory) TemplateName() string {
	return api.IdpTemplateBingoIAM
}

func (drv SBingoIAMDriverFactory) IdpAttributeOptions() api.SIdpAttributeOptions {
	return api.SIdpAttributeOptions{
		UserNameAttribute:        "name",
		UserIdAttribute:          "user_id",
		UserDisplaynameAttribtue: "display_name",
		DomainIdAttribute:        "tenant_id",
		DomainNameAttribute:      "tenant_name",
	}
}

func (drv SBingoIAMDriverFactory) ValidateConfig(conf api.SOAuth2IdpConfigOptions) error {
	return nil
}

func init() {
	oauth2.Register(&SBingoIAMDriverFactory{})
}
