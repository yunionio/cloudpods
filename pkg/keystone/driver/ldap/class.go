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

package ldap

import (
	"context"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/driver"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SLDAPDriverClass struct{}

func (self *SLDAPDriverClass) IsSso() bool {
	return false
}

func (self *SLDAPDriverClass) ForceSyncUser() bool {
	return true
}

func (self *SLDAPDriverClass) GetDefaultIconUri(tmpName string) string {
	return ""
}

func (self *SLDAPDriverClass) SingletonInstance() bool {
	return false
}

func (self *SLDAPDriverClass) SyncMethod() string {
	return api.IdentityProviderSyncFull
}

func (self *SLDAPDriverClass) NewDriver(idpId, idpName, template, targetDomainId string, conf api.TConfigs) (driver.IIdentityBackend, error) {
	return NewLDAPDriver(idpId, idpName, template, targetDomainId, conf)
}

func (self *SLDAPDriverClass) Name() string {
	return api.IdentityDriverLDAP
}

func (self *SLDAPDriverClass) ValidateConfig(ctx context.Context, userCred mcclient.TokenCredential, template string, tconf api.TConfigs, idpId, domainId string) (api.TConfigs, error) {
	conf := api.SLDAPIdpConfigOptions{}
	confJson := jsonutils.Marshal(tconf[api.IdentityDriverLDAP])
	err := confJson.Unmarshal(&conf)
	if err != nil {
		return tconf, errors.Wrap(err, "unmarshal config")
	}
	if len(conf.Url) == 0 {
		return tconf, errors.Wrap(httperrors.ErrInputParameter, "empty url")
	}
	_, err = url.Parse(conf.Url)
	if err != nil {
		return tconf, errors.Wrap(httperrors.ErrInputParameter, "invalid url")
	}
	if len(conf.Suffix) == 0 {
		return tconf, errors.Wrap(httperrors.ErrInputParameter, "empty suffix")
	}
	// validate uniqueness
	unique, err := models.IdentityProviderManager.CheckUniqueness(idpId, domainId, api.IdentityDriverLDAP, template, api.IdentityDriverLDAP, "url", jsonutils.NewString(conf.Url))
	if err != nil {
		return tconf, errors.Wrap(err, "IdentityProviderManager.CheckUniqueness")
	}
	if !unique {
		return tconf, errors.Wrapf(httperrors.ErrDuplicateResource, "ldap URL %s has been registered", conf.Url)
	}
	return tconf, nil
}

func init() {
	driver.RegisterDriverClass(&SLDAPDriverClass{})
}
