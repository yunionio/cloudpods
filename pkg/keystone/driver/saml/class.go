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

func (self *SSAMLDriverClass) ValidateConfig(ctx context.Context, userCred mcclient.TokenCredential, template string, tconf api.TConfigs, idpId, domainId string) (api.TConfigs, error) {
	conf := api.SSAMLIdpConfigOptions{}
	confJson := jsonutils.Marshal(tconf[api.IdentityDriverSAML])
	err := confJson.Unmarshal(&conf)
	if err != nil {
		return tconf, errors.Wrap(err, "unmarshal config")
	}
	switch template {
	case api.IdpTemplateAzureADSAML:
		if tid, ok := tconf[api.IdentityDriverSAML]["tenant_id"]; !ok || tid == nil {
			return tconf, errors.Wrap(httperrors.ErrInputParameter, "empty tenant_id")
		} else {
			tidStr, _ := tid.GetString()
			if len(tidStr) == 0 {
				return tconf, errors.Wrap(httperrors.ErrInputParameter, "empty tenant_id")
			}
			// validate uniqueness
			unique, err := models.IdentityProviderManager.CheckUniqueness(idpId, domainId, api.IdentityDriverSAML, template, api.IdentityDriverSAML, "tenant_id", jsonutils.NewString(tidStr))
			if err != nil {
				return tconf, errors.Wrap(err, "IdentityProviderManager.CheckUniqueness")
			}
			if !unique {
				return tconf, errors.Wrapf(httperrors.ErrDuplicateResource, "tenant_id %s has been registered", tidStr)
			}
		}
	case api.IdpTemplateSAMLTest:
		// validate uniqueness
		unique, err := models.IdentityProviderManager.CheckUniqueness(idpId, domainId, api.IdentityDriverSAML, template, "", "", nil)
		if err != nil {
			return tconf, errors.Wrap(err, "IdentityProviderManager.CheckUniqueness")
		}
		if !unique {
			return tconf, errors.Wrap(httperrors.ErrDuplicateResource, "SAMLTest has been registered")
		}
	default:
		if len(conf.EntityId) == 0 {
			return tconf, errors.Wrap(httperrors.ErrInputParameter, "empty entity_id")
		}
		if len(conf.RedirectSSOUrl) == 0 {
			return tconf, errors.Wrap(httperrors.ErrInputParameter, "empty redirect_sso_url")
		}
		_, err = url.Parse(conf.RedirectSSOUrl)
		if err != nil {
			return tconf, errors.Wrap(httperrors.ErrInputParameter, "invalid redirect_sso_url")
		}
		// validate uniqueness
		unique, err := models.IdentityProviderManager.CheckUniqueness(idpId, domainId, api.IdentityDriverSAML, template, api.IdentityDriverSAML, "entity_id", jsonutils.NewString(conf.EntityId))
		if err != nil {
			return tconf, errors.Wrap(err, "IdentityProviderManager.CheckUniqueness")
		}
		if !unique {
			return tconf, errors.Wrapf(httperrors.ErrDuplicateResource, "entity_id %s has been registered", conf.EntityId)
		}
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
	nconf["allow_idp_init"] = jsonutils.JSONTrue
	tconf[api.IdentityDriverSAML] = nconf
	return tconf, nil
}

func init() {
	driver.RegisterDriverClass(&SSAMLDriverClass{})
}
