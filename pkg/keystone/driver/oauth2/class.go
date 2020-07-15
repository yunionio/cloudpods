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

package oauth2

import (
	"context"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/keystone/driver"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SOAuth2DriverClass struct{}

func (self *SOAuth2DriverClass) IsSso() bool {
	return true
}

func (self *SOAuth2DriverClass) ForceSyncUser() bool {
	return false
}

func (self *SOAuth2DriverClass) GetDefaultIconUri(tmpName string) string {
	switch tmpName {
	case api.IdpTemplateDingtalk:
		return "https://img.alicdn.com/tfs/TB13Bxnd3oQMeJjy0FoXXcShVXa-80-80.png"
	case api.IdpTemplateFeishu:
		return "https://sf1-ttcdn-tos.pstatp.com/obj/suite-public-file-cn/feishu-share-icon.png"
	case api.IdpTemplateAlipay:
		return "https://gw.alipayobjects.com/mdn/member_frontWeb/afts/img/A*h7o9Q4g2KiUAAAAAAAAAAABkARQnAQ"
	case api.IdpTemplateWechat:
		return "https://open.weixin.qq.com/zh_CN/htmledition/res/assets/res-design-download/icon64_appwx_logo.png"
	case api.IdpTemplateQywechat:
		return "http://yunioniso.oss-cn-beijing.aliyuncs.com/icons/qywechat_logo.png"
	}
	return "https://st.fbk.eu/sites/st.fbk.eu/files/styles/threshold-1382/public/oauth2-logo.jpg"
}

func (self *SOAuth2DriverClass) SingletonInstance() bool {
	return false
}

func (self *SOAuth2DriverClass) SyncMethod() string {
	return api.IdentityProviderSyncOnAuth
}

func (self *SOAuth2DriverClass) NewDriver(idpId, idpName, template, targetDomainId string, conf api.TConfigs) (driver.IIdentityBackend, error) {
	return NewOAuth2Driver(idpId, idpName, template, targetDomainId, conf)
}

func (self *SOAuth2DriverClass) Name() string {
	return api.IdentityDriverOAuth2
}

func (self *SOAuth2DriverClass) ValidateConfig(ctx context.Context, userCred mcclient.TokenCredential, tconf api.TConfigs) (api.TConfigs, error) {
	return tconf, nil
}

func init() {
	driver.RegisterDriverClass(&SOAuth2DriverClass{})
}
