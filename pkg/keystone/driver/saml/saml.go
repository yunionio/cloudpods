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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/driver"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/keystone/saml"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// SAML 2.0 Service Provider Driver
type SSAMLDriver struct {
	driver.SBaseIdentityDriver

	samlConfig *api.SSAMLIdpConfigOptions

	isDebug bool
}

func NewSAMLDriver(idpId, idpName, template, targetDomainId string, conf api.TConfigs) (driver.IIdentityBackend, error) {
	base, err := driver.NewBaseIdentityDriver(idpId, idpName, template, targetDomainId, conf)
	if err != nil {
		return nil, errors.Wrap(err, "NewBaseIdentityDriver")
	}
	drv := SSAMLDriver{SBaseIdentityDriver: base}
	drv.SetVirtualObject(&drv)
	err = drv.prepareConfig()
	if err != nil {
		return nil, errors.Wrap(err, "prepareConfig")
	}
	return &drv, nil
}

func (self *SSAMLDriver) prepareConfig() error {
	if self.samlConfig == nil {
		confJson := jsonutils.Marshal(self.Config["saml"])
		conf := api.SSAMLIdpConfigOptions{}
		switch self.Template {
		case api.IdpTemplateSAMLTest:
			conf = SAMLTestTemplate
		case api.IdpTemplateAzureADSAML:
			conf = AzureADTemplate
			tenantId, _ := confJson.GetString("tenant_id")
			conf.EntityId = fmt.Sprintf("https://sts.windows.net/%s/", tenantId)
			conf.RedirectSSOUrl = fmt.Sprintf("https://login.microsoftonline.com/%s/saml2", tenantId)
		}
		err := confJson.Unmarshal(&conf)
		if err != nil {
			return errors.Wrap(err, "json.Unmarshal")
		}
		log.Debugf("%s %s %#v", self.Config, confJson, self.samlConfig)
		self.samlConfig = &conf
	}
	return nil
}

func (self *SSAMLDriver) Authenticate(ctx context.Context, ident mcclient.SAuthenticationIdentity) (*api.SUserExtended, error) {
	// no need to base64 decode and decrypt, just unmarshal XML
	/*_, err := samlutils.ValidateXML(ident.SAMLAuth.Response)
	if err != nil {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "ValidateXML fail on SAMLResponse")
	}*/
	resp, err := saml.SAMLInstance().UnmarshalResponse([]byte(ident.SAMLAuth.Response))
	if err != nil {
		return nil, errors.Wrap(err, "SAMLInstance().UnmarshalResponse")
	}

	if !resp.IsSuccess() {
		return nil, errors.Wrap(httperrors.ErrInvalidCredential, "SAML auth unsuccess")
	}

	attrs := resp.FetchAttribtues()

	var usrId, usrName string
	if v, ok := attrs[self.samlConfig.UserIdAttribute]; ok && len(v) > 0 {
		usrId = v[0]
	}
	if v, ok := attrs[self.samlConfig.UserNameAttribute]; ok && len(v) > 0 {
		usrName = v[0]
	}
	if len(usrId) == 0 && len(usrName) == 0 {
		return nil, errors.Wrap(httperrors.ErrUnauthenticated, "empty userId or userName")
	}
	if len(usrId) == 0 {
		usrId = usrName
	} else if len(usrName) == 0 {
		usrName = usrId
	}

	idp, err := models.IdentityProviderManager.FetchIdentityProviderById(self.IdpId)
	if err != nil {
		return nil, errors.Wrap(err, "self.GetIdentityProvider")
	}
	domain, err := idp.GetSingleDomain(ctx, api.DefaultRemoteDomainId, self.IdpName, fmt.Sprintf("cas provider %s", self.IdpName), false)
	if err != nil {
		return nil, errors.Wrap(err, "idp.GetSingleDomain")
	}
	usr, err := idp.SyncOrCreateUser(ctx, usrId, usrName, domain.Id, true, nil)
	if err != nil {
		return nil, errors.Wrap(err, "idp.SyncOrCreateUser")
	}
	extUser, err := models.UserManager.FetchUserExtended(usr.Id, "", "", "")
	if err != nil {
		return nil, errors.Wrap(err, "models.UserManager.FetchUserExtended")
	}

	idp.TryUserJoinProject(self.samlConfig.SIdpAttributeOptions, ctx, usr, domain.Id, attrs)

	return extUser, nil
}

func (self *SSAMLDriver) Sync(ctx context.Context) error {
	return nil
}

func (self *SSAMLDriver) Probe(ctx context.Context) error {
	return nil
}
