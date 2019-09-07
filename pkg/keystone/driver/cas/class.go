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

package cas

import (
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/keystone/driver"
)

type SCASDriverClass struct{}

func (self *SCASDriverClass) SingletonInstance() bool {
	return true
}

func (self *SCASDriverClass) SyncMethod() string {
	return api.IdentityProviderSyncOnAuth
}

func (self *SCASDriverClass) NewDriver(idpId, idpName, template, targetDomainId string, autoCreateProject bool, conf api.TIdentityProviderConfigs) (driver.IIdentityBackend, error) {
	return NewCASDriver(idpId, idpName, template, targetDomainId, autoCreateProject, conf)
}

func (self *SCASDriverClass) Name() string {
	return api.IdentityDriverCAS
}

func init() {
	driver.RegisterDriverClass(&SCASDriverClass{})
}
