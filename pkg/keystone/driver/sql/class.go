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

package sql

import (
	"context"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/keystone/driver"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SSQLDriverClass struct{}

func (self *SSQLDriverClass) IsSso() bool {
	return false
}

func (self *SSQLDriverClass) ForceSyncUser() bool {
	return true
}

func (self *SSQLDriverClass) GetDefaultIconUri(tmpName string) string {
	return ""
}

func (self *SSQLDriverClass) SingletonInstance() bool {
	return true
}

func (self *SSQLDriverClass) SyncMethod() string {
	return api.IdentityProviderSyncLocal
}

func (self *SSQLDriverClass) NewDriver(idpId, idpName, template, targetDomainId string, conf api.TConfigs) (driver.IIdentityBackend, error) {
	return NewSQLDriver(idpId, idpName, template, targetDomainId, conf)
}

func (self *SSQLDriverClass) Name() string {
	return api.IdentityDriverSQL
}

func (self *SSQLDriverClass) ValidateConfig(ctx context.Context, userCred mcclient.TokenCredential, template string, conf api.TConfigs) (api.TConfigs, error) {
	return conf, nil
}

func init() {
	driver.RegisterDriverClass(&SSQLDriverClass{})
}
