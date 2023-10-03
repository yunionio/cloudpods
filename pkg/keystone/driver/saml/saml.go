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
	"encoding/base64"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/samlutils"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/driver"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/keystone/saml"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/samlutils/sp"
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

func (self *SSAMLDriver) GetSsoCallbackUri(callbackUrl string) string {
	if self.samlConfig.AllowIdpInit != nil && *self.samlConfig.AllowIdpInit {
		callbackUrl = httputils.JoinPath(callbackUrl, self.IdpId)
	}
	return callbackUrl
}

func (self *SSAMLDriver) GetSsoRedirectUri(ctx context.Context, callbackUrl, state string) (string, error) {
	spLoginFunc := func(ctx context.Context, idp *sp.SSAMLIdentityProvider) (sp.SSAMLSpInitiatedLoginRequest, error) {
		result := sp.SSAMLSpInitiatedLoginRequest{}
		result.RequestID = samlutils.GenerateSAMLId()
		result.RelayState = state
		return result, nil
	}
	spInst := sp.NewSpInstance(saml.SAMLInstance(), self.IdpName, nil, spLoginFunc)
	spInst.SetAssertionConsumerUri(callbackUrl)
	err := spInst.AddIdp(self.samlConfig.EntityId, self.samlConfig.RedirectSSOUrl)
	if err != nil {
		return "", errors.Wrap(err, "Invalid SAMLIdentityProvider")
	}
	input := samlutils.SSpInitiatedLoginInput{
		EntityID: self.samlConfig.EntityId,
	}
	redir, err := spInst.ProcessSpInitiatedLogin(ctx, input)
	if err != nil {
		return "", errors.Wrap(err, "ProcessSpInitiatedLogin")
	}
	return redir, nil
}

func (self *SSAMLDriver) Authenticate(ctx context.Context, ident mcclient.SAuthenticationIdentity) (*api.SUserExtended, error) {
	samlRespBytes, err := base64.StdEncoding.DecodeString(ident.SAMLAuth.Response)
	if err != nil {
		return nil, errors.Wrap(err, "base64.StdEncoding.DecodeString")
	}

	resp, err := saml.SAMLInstance().UnmarshalResponse(samlRespBytes)
	if err != nil {
		return nil, errors.Wrap(err, "decode SAMLResponse error")
	}

	if !resp.IsSuccess() {
		return nil, errors.Wrap(httperrors.ErrInvalidCredential, "SAML auth unsuccess")
	}

	attrs := resp.FetchAttribtues()

	var domainId, domainName, usrId, usrName string
	if v, ok := attrs[self.samlConfig.DomainIdAttribute]; ok && len(v) > 0 {
		domainId = v[0]
	}
	if v, ok := attrs[self.samlConfig.DomainNameAttribute]; ok && len(v) > 0 {
		domainName = v[0]
	}
	if v, ok := attrs[self.samlConfig.UserIdAttribute]; ok && len(v) > 0 {
		usrId = v[0]
	}
	if v, ok := attrs[self.samlConfig.UserNameAttribute]; ok && len(v) > 0 {
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

	idp.TryUserJoinProject(self.samlConfig.SIdpAttributeOptions, ctx, usr, domain.Id, attrs)

	extUser.AuditIds = []string{resp.ID}

	return extUser, nil
}

func (self *SSAMLDriver) Sync(ctx context.Context) error {
	return nil
}

func (self *SSAMLDriver) Probe(ctx context.Context) error {
	return nil
}
