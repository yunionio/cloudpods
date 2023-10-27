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
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/keystone/driver"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/keystone/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/oidcutils"
	"yunion.io/x/onecloud/pkg/util/oidcutils/client"
)

// OpenID Connect client driver
// https://openid.net/specs/openid-connect-basic-1_0.html
// https://tools.ietf.org/html/rfc6749
type SOIDCDriver struct {
	driver.SBaseIdentityDriver

	oidcConfig *api.SOIDCIdpConfigOptions

	isDebug bool
}

func NewOIDCDriver(idpId, idpName, template, targetDomainId string, conf api.TConfigs) (driver.IIdentityBackend, error) {
	base, err := driver.NewBaseIdentityDriver(idpId, idpName, template, targetDomainId, conf)
	if err != nil {
		return nil, errors.Wrap(err, "NewBaseIdentityDriver")
	}
	isDebug := false
	if options.Options.LogLevel == "debug" {
		isDebug = true
	}
	drv := SOIDCDriver{
		SBaseIdentityDriver: base,
		isDebug:             isDebug,
	}
	drv.SetVirtualObject(&drv)
	err = drv.prepareConfig()
	if err != nil {
		return nil, errors.Wrap(err, "prepareConfig")
	}
	return &drv, nil
}

func (self *SOIDCDriver) prepareConfig() error {
	if self.oidcConfig == nil {
		confJson := jsonutils.Marshal(self.Config[api.IdentityDriverOIDC])
		conf := api.SOIDCIdpConfigOptions{}
		switch self.Template {
		case api.IdpTemplateDex:
			conf = DexOIDCTemplate
		case api.IdpTemplateGithub:
			conf = GithubOIDCTemplate
		case api.IdpTemplateGoogle:
			conf = GoogleOIDCTemplate
		case api.IdpTemplateAzureOAuth2:
			conf = AzureADTemplate
			tenantId, _ := confJson.GetString("tenant_id")
			if len(tenantId) == 0 {
				tenantId = "common"
			}
			cloudEnv, _ := confJson.GetString("cloud_env")
			loginUrl := "https://login.microsoftonline.com"
			graphUrl := "https://graph.microsoft.com"
			switch cloudEnv {
			case api.AZURE_CLOUD_ENV_CHINA:
				loginUrl = "https://login.partner.microsoftonline.cn"
				graphUrl = "https://microsoftgraph.chinacloudapi.cn"
			}
			conf.AuthUrl = fmt.Sprintf("%s/%s/oauth2/v2.0/authorize", loginUrl, tenantId)
			conf.TokenUrl = fmt.Sprintf("%s/%s/oauth2/v2.0/token", loginUrl, tenantId)
			conf.UserinfoUrl = fmt.Sprintf("%s/oidc/userinfo", graphUrl)
		}
		err := confJson.Unmarshal(&conf)
		if err != nil {
			return errors.Wrap(err, "json.Unmarshal")
		}
		log.Debugf("%s %s %#v", self.Config, confJson, self.oidcConfig)
		self.oidcConfig = &conf
	}
	return nil
}

func (self *SOIDCDriver) getOIDCClient(ctx context.Context) (*client.SOIDCClient, error) {
	timeout := self.oidcConfig.TimeoutSecs
	if timeout <= 0 {
		timeout = 30
	}
	cli := client.NewOIDCClient(self.oidcConfig.ClientId, self.oidcConfig.ClientSecret, timeout, self.isDebug)
	if len(self.oidcConfig.Endpoint) > 0 {
		err := cli.FetchConfiguration(ctx, self.oidcConfig.Endpoint)
		if err != nil {
			return nil, errors.Wrap(err, "FetchConfiguration")
		}
	} else {
		cli.SetConfig(self.oidcConfig.AuthUrl, self.oidcConfig.TokenUrl, self.oidcConfig.UserinfoUrl, self.oidcConfig.Scopes)
	}
	log.Debugf("Userinfo url: %s", cli.GetConfig().UserinfoEndpoint)
	return cli, nil
}

func (oidc *SOIDCDriver) GetSsoRedirectUri(ctx context.Context, callbackUrl, state string) (string, error) {
	cli, err := oidc.getOIDCClient(ctx)
	if err != nil {
		return "", errors.Wrap(err, "getOIDCClient")
	}
	conf := cli.GetConfig()
	qs := oidcutils.SOIDCAuthRequest{
		ResponseType: oidcutils.OIDC_RESPONSE_TYPE_CODE,
		ClientId:     oidc.oidcConfig.ClientId,
		RedirectUri:  callbackUrl,
		State:        state,
		Scope:        strings.Join(conf.ScopesSupported, " "),
	}
	urlstr := fmt.Sprintf("%s?%s", conf.AuthorizationEndpoint, jsonutils.Marshal(qs).QueryString())
	return urlstr, nil
}

func (self *SOIDCDriver) Authenticate(ctx context.Context, ident mcclient.SAuthenticationIdentity) (*api.SUserExtended, error) {
	cli, err := self.getOIDCClient(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "getOIDCClient")
	}
	token, err := cli.FetchToken(ctx, ident.OIDCAuth.Code, ident.OIDCAuth.RedirectUri)
	if err != nil {
		return nil, errors.Wrapf(err, "OIDCClient.FetchToken %s", self.oidcConfig.TokenUrl)
	}
	userAttrs, err := cli.FetchUserInfo(ctx, token.AccessToken)
	if err != nil {
		return nil, errors.Wrap(err, "OIDCClient.FetchUserInfo")
	}

	log.Debugf("attrs: %s", userAttrs)

	attrs := make(map[string][]string)
	for k, v := range userAttrs {
		attrs[k] = []string{v}
	}

	var domainId, domainName, usrId, usrName string
	if v, ok := attrs[self.oidcConfig.DomainIdAttribute]; ok && len(v) > 0 {
		domainId = v[0]
	}
	if v, ok := attrs[self.oidcConfig.DomainNameAttribute]; ok && len(v) > 0 {
		domainName = v[0]
	}
	if v, ok := attrs[self.oidcConfig.UserIdAttribute]; ok && len(v) > 0 {
		usrId = v[0]
	}
	if v, ok := attrs[self.oidcConfig.UserNameAttribute]; ok && len(v) > 0 {
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

	idp.TryUserJoinProject(self.oidcConfig.SIdpAttributeOptions, ctx, usr, domain.Id, attrs)

	extUser.AuditIds = []string{ident.OIDCAuth.Code}

	return extUser, nil
}

func (self *SOIDCDriver) Sync(ctx context.Context) error {
	return nil
}

func (self *SOIDCDriver) Probe(ctx context.Context) error {
	return nil
}
