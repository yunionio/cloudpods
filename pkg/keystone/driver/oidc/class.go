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

package oidc

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/driver"
	"yunion.io/x/onecloud/pkg/keystone/driver/utils"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/oidcutils/client"
)

type SOIDCDriverClass struct{}

func (self *SOIDCDriverClass) IsSso() bool {
	return true
}

func (self *SOIDCDriverClass) ForceSyncUser() bool {
	return false
}

func (self *SOIDCDriverClass) GetDefaultIconUri(tmpName string) string {
	switch tmpName {
	case api.IdpTemplateDex:
		return "https://raw.githubusercontent.com/dexidp/dex/master/Documentation/logos/dex-glyph-color.png"
	case api.IdpTemplateGithub:
		return "https://github.githubassets.com/images/modules/logos_page/GitHub-Mark.png"
	case api.IdpTemplateAzureOAuth2:
		return "https://upload.wikimedia.org/wikipedia/commons/a/a8/Microsoft_Azure_Logo.svg"
	}
	return "https://openid.net/wordpress-content/uploads/2011/09/OPENID_CONNECT_NEW-Logo-1024x474.jpg"
}

func (self *SOIDCDriverClass) SingletonInstance() bool {
	return false
}

func (self *SOIDCDriverClass) SyncMethod() string {
	return api.IdentityProviderSyncOnAuth
}

func (self *SOIDCDriverClass) NewDriver(idpId, idpName, template, targetDomainId string, conf api.TConfigs) (driver.IIdentityBackend, error) {
	return NewOIDCDriver(idpId, idpName, template, targetDomainId, conf)
}

func (self *SOIDCDriverClass) Name() string {
	return api.IdentityDriverOIDC
}

func (self *SOIDCDriverClass) ValidateConfig(ctx context.Context, userCred mcclient.TokenCredential, template string, tconf api.TConfigs, idpId, domainId string) (api.TConfigs, error) {
	conf := api.SOIDCIdpConfigOptions{}
	confJson := jsonutils.Marshal(tconf[api.IdentityDriverOIDC])
	err := confJson.Unmarshal(&conf)
	if err != nil {
		return tconf, errors.Wrap(err, "unmarshal config")
	}
	switch template {
	case api.IdpTemplateAzureOAuth2:
		if tid, ok := tconf[api.IdentityDriverOIDC]["tenant_id"]; !ok || tid == nil {
			return tconf, errors.Wrap(httperrors.ErrInputParameter, "empty tenant_id")
		} else {
			tidStr, _ := tid.GetString()
			if len(tidStr) == 0 {
				return tconf, errors.Wrap(httperrors.ErrInputParameter, "empty tenant_id")
			}
		}
	}
	if len(conf.ClientId) == 0 {
		return tconf, errors.Wrap(httperrors.ErrInputParameter, "empty client_id")
	}
	if len(conf.ClientSecret) == 0 {
		return tconf, errors.Wrap(httperrors.ErrInputParameter, "empty client_secret")
	}
	if len(conf.Endpoint) > 0 {
		cli := client.NewOIDCClient(conf.ClientId, conf.ClientSecret, 30, false)
		err := cli.FetchConfiguration(ctx, conf.Endpoint)
		if err != nil {
			return tconf, errors.Wrapf(err, "invalid endpoint %s", conf.Endpoint)
		}
	}
	// validate uniqueness
	unique, err := models.IdentityProviderManager.CheckUniqueness(idpId, domainId, api.IdentityDriverOIDC, template, api.IdentityDriverOIDC, "client_id", jsonutils.NewString(conf.ClientId))
	if err != nil {
		return tconf, errors.Wrap(err, "IdentityProviderManager.CheckUniqueness")
	}
	if !unique {
		return tconf, errors.Wrapf(httperrors.ErrDuplicateResource, "client_id %s has been registered", conf.ClientId)
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
	tconf[api.IdentityDriverOIDC] = nconf
	return tconf, nil
}

func init() {
	driver.RegisterDriverClass(&SOIDCDriverClass{})
}
