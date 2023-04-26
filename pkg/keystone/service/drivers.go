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

package service

import (
	_ "yunion.io/x/onecloud/pkg/keystone/driver/cas"
	_ "yunion.io/x/onecloud/pkg/keystone/driver/ldap"
	_ "yunion.io/x/onecloud/pkg/keystone/driver/oauth2"
	_ "yunion.io/x/onecloud/pkg/keystone/driver/oauth2/alipay"
	_ "yunion.io/x/onecloud/pkg/keystone/driver/oauth2/bingoiam"
	_ "yunion.io/x/onecloud/pkg/keystone/driver/oauth2/dingtalk"
	_ "yunion.io/x/onecloud/pkg/keystone/driver/oauth2/feishu"
	_ "yunion.io/x/onecloud/pkg/keystone/driver/oauth2/qywechat"
	_ "yunion.io/x/onecloud/pkg/keystone/driver/oauth2/wechat"
	_ "yunion.io/x/onecloud/pkg/keystone/driver/oidc"
	_ "yunion.io/x/onecloud/pkg/keystone/driver/saml"
	_ "yunion.io/x/onecloud/pkg/keystone/driver/sql"
)
