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
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/keystone/driver"
	"yunion.io/x/onecloud/pkg/keystone/driver/utils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SCASDriverClass struct{}

func (self *SCASDriverClass) IsSso() bool {
	return true
}

func (self *SCASDriverClass) ForceSyncUser() bool {
	return false
}

func (self *SCASDriverClass) GetDefaultIconUri(tmpName string) string {
	return "https://www.apereo.org/sites/default/files/styles/project_logo/public/projects/logos/cas_max_logo_0.png"
}

func (self *SCASDriverClass) SingletonInstance() bool {
	return false
}

func (self *SCASDriverClass) SyncMethod() string {
	return api.IdentityProviderSyncOnAuth
}

func (self *SCASDriverClass) NewDriver(idpId, idpName, template, targetDomainId string, conf api.TConfigs) (driver.IIdentityBackend, error) {
	return NewCASDriver(idpId, idpName, template, targetDomainId, conf)
}

func (self *SCASDriverClass) Name() string {
	return api.IdentityDriverCAS
}

func (self *SCASDriverClass) ValidateConfig(ctx context.Context, userCred mcclient.TokenCredential, tconf api.TConfigs) (api.TConfigs, error) {
	conf := api.SCASIdpConfigOptions{}
	confJson := jsonutils.Marshal(tconf[api.IdentityDriverCAS])
	err := confJson.Unmarshal(&conf)
	if err != nil {
		return tconf, errors.Wrap(err, "unmarshal config")
	}
	conf.SIdpAttributeOptions, err = utils.ValidateConfig(conf.SIdpAttributeOptions, userCred)
	if err != nil {
		return tconf, errors.Wrap(err, "ValidateConfig")
	}
	nconf := make(map[string]jsonutils.JSONObject)
	err = jsonutils.Marshal(conf).Unmarshal(&nconf)
	if err != nil {
		return tconf, errors.Wrap(err, "Unmarshal new config")
	}
	tconf[api.IdentityDriverCAS] = nconf
	return tconf, nil
}

func init() {
	driver.RegisterDriverClass(&SCASDriverClass{})
}
