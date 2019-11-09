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
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/keystone/driver"
)

type SSQLDriverClass struct{}

func (self *SSQLDriverClass) SingletonInstance() bool {
	return true
}

func (self *SSQLDriverClass) SyncMethod() string {
	return api.IdentityProviderSyncLocal
}

func (self *SSQLDriverClass) NewDriver(idpId, idpName, template, targetDomainId string, autoCreateProject bool, conf api.TConfigs) (driver.IIdentityBackend, error) {
	return NewSQLDriver(idpId, idpName, template, targetDomainId, autoCreateProject, conf)
}

func (self *SSQLDriverClass) Name() string {
	return api.IdentityDriverSQL
}

func init() {
	driver.RegisterDriverClass(&SSQLDriverClass{})
}
