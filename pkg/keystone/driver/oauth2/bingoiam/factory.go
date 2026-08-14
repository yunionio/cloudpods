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
		DomainNameAttribute:      "tenant_name",
		DomainIdAttribute:        "tenant_id",
		UserNameAttribute:        "name",
		UserIdAttribute:          "user_id",
		UserDisplaynameAttribtue: "display_name",
		ProjectAttribute:         "",
		DefaultProjectId:         "",
	}
}

func (drv SBingoIAMDriverFactory) ValidateConfig(conf api.SOAuth2IdpConfigOptions) error {
	return nil
}

func (drv SBingoIAMDriverFactory) AttributeNames() map[string]string {
	return map[string]string{
		"tenant_name":  "Tenant name in BingoIAM",
		"tenant_id":    "Tenant ID in BingoIAM",
		"name":         "User name in BingoIAM",
		"name_en":      "English user name in BingoIAM",
		"user_id":      "User ID in BingoIAM",
		"display_name": "Display name in BingoIAM",
		"email":        "Email in BingoIAM",
		"mobile":       "Mobile in BingoIAM",
		"org_id":       "Org ID in BingoIAM",
	}
}

func init() {
	oauth2.Register(&SBingoIAMDriverFactory{})
}
