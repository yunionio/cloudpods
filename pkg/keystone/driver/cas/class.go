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
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/driver"
	"yunion.io/x/onecloud/pkg/keystone/driver/utils"
	"yunion.io/x/onecloud/pkg/keystone/models"
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

func (self *SCASDriverClass) ValidateConfig(ctx context.Context, userCred mcclient.TokenCredential, template string, tconf api.TConfigs, idpId, domainId string) (api.TConfigs, error) {
	conf := api.SCASIdpConfigOptions{}
	confJson := jsonutils.Marshal(tconf[api.IdentityDriverCAS])
	err := confJson.Unmarshal(&conf)
	if err != nil {
		return tconf, errors.Wrap(err, "unmarshal config")
	}
	if len(conf.CASServerURL) == 0 {
		return tconf, errors.Wrap(httperrors.ErrInputParameter, "empty cas_server_url")
	}
	_, err = url.Parse(conf.CASServerURL)
	if err != nil {
		return tconf, errors.Wrap(httperrors.ErrInputParameter, "invalid cas_server_url")
	}
	// validate uniqueness
	unique, err := models.IdentityProviderManager.CheckUniqueness(idpId, domainId, api.IdentityDriverCAS, template, api.IdentityDriverCAS, "cas_server_url", jsonutils.NewString(conf.CASServerURL))
	if err != nil {
		return tconf, errors.Wrap(err, "IdentityProviderManager.CheckUniqueness")
	}
	if !unique {
		return tconf, errors.Wrapf(httperrors.ErrDuplicateResource, "cas_server_url %s has been registered", conf.CASServerURL)
	}
	conf.SIdpAttributeOptions, err = utils.ValidateConfig(ctx, conf.SIdpAttributeOptions, userCred)
	if err != nil {
		return tconf, errors.Wrap(err, "ValidateConfig")
	}
	nconf := make(map[string]jsonutils.JSONObject)
	err = confJson.Unmarshal(&nconf)
	if err != nil {
		return tconf, errors.Wrap(err, "Unmarshal old config")
	}
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
