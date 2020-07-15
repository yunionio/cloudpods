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

package saml

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/keystone/driver"
	"yunion.io/x/onecloud/pkg/keystone/driver/utils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SSAMLDriverClass struct{}

func (self *SSAMLDriverClass) IsSso() bool {
	return true
}

func (self *SSAMLDriverClass) GetDefaultIconUri(tmpName string) string {
	switch tmpName {
	case api.IdpTemplateAzureADSAML:
		return "https://upload.wikimedia.org/wikipedia/commons/a/a8/Microsoft_Azure_Logo.svg"
	}
	return "https://www.oasis-open.org/committees/download.php/29723/draft-saml-logo-03.png"
}

func (self *SSAMLDriverClass) ForceSyncUser() bool {
	return false
}

func (self *SSAMLDriverClass) SingletonInstance() bool {
	return false
}

func (self *SSAMLDriverClass) SyncMethod() string {
	return api.IdentityProviderSyncOnAuth
}

func (self *SSAMLDriverClass) NewDriver(idpId, idpName, template, targetDomainId string, conf api.TConfigs) (driver.IIdentityBackend, error) {
	return NewSAMLDriver(idpId, idpName, template, targetDomainId, conf)
}

func (self *SSAMLDriverClass) Name() string {
	return api.IdentityDriverSAML
}

func (self *SSAMLDriverClass) ValidateConfig(ctx context.Context, userCred mcclient.TokenCredential, tconf api.TConfigs) (api.TConfigs, error) {
	conf := api.SSAMLIdpConfigOptions{}
	confJson := jsonutils.Marshal(tconf[api.IdentityDriverSAML])
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
	tconf[api.IdentityDriverSAML] = nconf
	return tconf, nil
}

func init() {
	driver.RegisterDriverClass(&SSAMLDriverClass{})
}
