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

package oauth2

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/driver"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// OAuth2.0
type SOAuth2Driver struct {
	driver.SBaseIdentityDriver

	oauth2Config *api.SOAuth2IdpConfigOptions

	isDebug bool
}

func NewOAuth2Driver(idpId, idpName, template, targetDomainId string, conf api.TConfigs) (driver.IIdentityBackend, error) {
	base, err := driver.NewBaseIdentityDriver(idpId, idpName, template, targetDomainId, conf)
	if err != nil {
		return nil, errors.Wrap(err, "NewBaseIdentityDriver")
	}
	drv := SOAuth2Driver{SBaseIdentityDriver: base}
	drv.SetVirtualObject(&drv)
	err = drv.prepareConfig()
	if err != nil {
		return nil, errors.Wrap(err, "prepareConfig")
	}
	return &drv, nil
}

func (self *SOAuth2Driver) prepareConfig() error {
	if self.oauth2Config == nil {
		confJson := jsonutils.Marshal(self.Config[api.IdentityDriverOAuth2])
		conf := api.SOAuth2IdpConfigOptions{}
		err := confJson.Unmarshal(&conf)
		if err != nil {
			return errors.Wrap(err, "json.Unmarshal")
		}
		log.Debugf("%s %s %#v", self.Config, confJson, self.oauth2Config)
		self.oauth2Config = &conf
	}
	return nil
}

func (self *SOAuth2Driver) GetSsoRedirectUri(ctx context.Context, callbackUrl, state string) (string, error) {
	factory := findDriverFactory(self.Template)
	if factory == nil {
		return "", errors.Wrapf(httperrors.ErrNotSupported, "template %s not supported", self.Template)
	}
	driver := factory.NewDriver(self.oauth2Config.AppId, self.oauth2Config.Secret)
	ctx = context.WithValue(ctx, "config", self.SBaseIdentityDriver.Config)
	return driver.GetSsoRedirectUri(ctx, callbackUrl, state)
}

func (self *SOAuth2Driver) Authenticate(ctx context.Context, ident mcclient.SAuthenticationIdentity) (*api.SUserExtended, error) {
	factory := findDriverFactory(self.Template)
	if factory == nil {
		return nil, errors.Wrapf(httperrors.ErrNotSupported, "template %s not supported", self.Template)
	}
	options := self.oauth2Config.SIdpAttributeOptions
	options.Update(factory.IdpAttributeOptions())
	driver := factory.NewDriver(self.oauth2Config.AppId, self.oauth2Config.Secret)
	ctx = context.WithValue(ctx, "config", self.SBaseIdentityDriver.Config)
	attrs, err := driver.Authenticate(ctx, ident.OAuth2.Code)
	if err != nil {
		return nil, errors.Wrapf(err, "driver %s Authenticate", self.Template)
	}

	var domainId, domainName, usrId, usrName string
	if v, ok := attrs[options.DomainIdAttribute]; ok && len(v) > 0 {
		domainId = v[0]
	}
	if v, ok := attrs[options.DomainNameAttribute]; ok && len(v) > 0 {
		domainName = v[0]
	}
	if v, ok := attrs[options.UserIdAttribute]; ok && len(v) > 0 {
		usrId = v[0]
	}
	if v, ok := attrs[options.UserNameAttribute]; ok && len(v) > 0 {
		usrName = v[0]
	}

	idp, err := models.IdentityProviderManager.FetchIdentityProviderById(self.IdpId)
	if err != nil {
		return nil, errors.Wrap(err, "self.GetIdentityProvider")
	}
	domain, usr, err := idp.SyncOrCreateDomainAndUser(ctx, domainId, domainName, usrId, usrName)
	if err != nil {
		return nil, errors.Wrap(err, "idp.SyncOrCreateDomainAndUser")
	}
	extUser, err := models.UserManager.FetchUserExtended(usr.Id, "", "", "")
	if err != nil {
		return nil, errors.Wrap(err, "models.UserManager.FetchUserExtended")
	}

	idp.TryUserJoinProject(options, ctx, usr, domain.Id, attrs)

	extUser.AuditIds = []string{ident.OAuth2.Code}

	return extUser, nil
}

func (self *SOAuth2Driver) Sync(ctx context.Context) error {
	factory := findDriverFactory(self.Template)
	if factory == nil {
		return nil
	}
	if driver, isOk := factory.NewDriver(self.oauth2Config.AppId, self.oauth2Config.Secret).(IOAuth2Synchronizer); isOk {
		ctx = context.WithValue(ctx, "config", self.SBaseIdentityDriver.Config)
		return driver.Sync(ctx, self.IdpId)
	}
	return nil
}

func (self *SOAuth2Driver) Probe(ctx context.Context) error {
	return nil
}
