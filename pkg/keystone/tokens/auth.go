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

package tokens

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/keystone/driver"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/keystone/options"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func authUserByTokenV2(ctx context.Context, input mcclient.SAuthenticationInputV2) (*models.SUserExtended, error) {
	return authUserByToken(ctx, input.Auth.Token.Id)
}

func authUserByTokenV3(ctx context.Context, input mcclient.SAuthenticationInputV3) (*models.SUserExtended, error) {
	return authUserByToken(ctx, input.Auth.Identity.Token.Id)
}

func authUserByToken(ctx context.Context, tokenStr string) (*models.SUserExtended, error) {
	token := SAuthToken{}
	err := token.ParseFernetToken(tokenStr)
	if err != nil {
		return nil, errors.Wrap(err, "token.ParseFernetToken")
	}
	return models.UserManager.FetchUserExtended(token.UserId, "", "", "")
}

func authUserByPasswordV2(ctx context.Context, input mcclient.SAuthenticationInputV2) (*models.SUserExtended, error) {
	ident := mcclient.SAuthenticationIdentity{}
	ident.Methods = []string{api.AUTH_METHOD_PASSWORD}
	ident.Password.User.Name = input.Auth.PasswordCredentials.Username
	ident.Password.User.Password = input.Auth.PasswordCredentials.Password
	ident.Password.User.Domain.Id = api.DEFAULT_DOMAIN_ID
	return authUserByIdentity(ctx, ident)
}

func authUserByIdentityV3(ctx context.Context, input mcclient.SAuthenticationInputV3) (*models.SUserExtended, error) {
	return authUserByIdentity(ctx, input.Auth.Identity)
}

func authUserByIdentity(ctx context.Context, ident mcclient.SAuthenticationIdentity) (*models.SUserExtended, error) {
	domain, err := models.DomainManager.FetchDomain(ident.Password.User.Domain.Id, ident.Password.User.Domain.Name)
	if err != nil {
		return nil, errors.Wrap(err, "DomainManager.FetchDomain")
	}
	conf, err := domain.GetConfig(true)
	if err != nil {
		return nil, errors.Wrap(err, "domain.GetConfig")
	}
	domainDriver, err := driver.GetDriver(domain.Id, conf)
	if err != nil {
		return nil, errors.Wrapf(err, "driver.GetDriver")
	}
	usrExt, err := domainDriver.Authenticate(ctx, ident)
	if err != nil {
		return nil, errors.Wrap(err, "domainDriver.Authenticate")
	}
	return usrExt, nil
}

func AuthenticateV3(ctx context.Context, input mcclient.SAuthenticationInputV3) (*mcclient.TokenCredentialV3, error) {
	var user *models.SUserExtended
	var err error
	if len(input.Auth.Identity.Methods) != 1 {
		return nil, ErrInvalidAuthMethod
	}
	method := input.Auth.Identity.Methods[0]
	if method == api.AUTH_METHOD_TOKEN {
		// auth by token
		user, err = authUserByTokenV3(ctx, input)
		if err != nil {
			return nil, errors.Wrap(err, "authUserByTokenV3")
		}
	} else {
		// auth by other methods, password, openid, saml, etc...
		user, err = authUserByIdentityV3(ctx, input)
		if err != nil {
			return nil, errors.Wrap(err, "authUserByIdentityV3")
		}
	}
	// user not found
	if user == nil {
		return nil, ErrUserNotFound
	}
	// user is not enabled
	if !user.Enabled {
		return nil, ErrUserDisabled
	}
	token := SAuthToken{}
	token.UserId = user.Id
	token.Method = method
	token.AuditIds = []string{utils.GenRequestId(16)}
	now := time.Now().UTC()
	token.ExpiresAt = now.Add(time.Duration(options.Options.TokenExpirationSeconds) * time.Second)
	token.Context = input.Auth.Context

	if len(input.Auth.Scope.Project.Id) == 0 && len(input.Auth.Scope.Project.Name) == 0 && len(input.Auth.Scope.Domain.Id) == 0 && len(input.Auth.Scope.Domain.Name) == 0 {
		// unscoped auth
		return token.getTokenV3(ctx, user, nil, nil)
	}
	var projExt *models.SProjectExtended
	var domain *models.SDomain
	if len(input.Auth.Scope.Project.Id) > 0 || len(input.Auth.Scope.Project.Name) > 0 {
		project, err := models.ProjectManager.FetchProject(input.Auth.Scope.Project.Id,
			input.Auth.Scope.Project.Name,
			input.Auth.Scope.Project.Domain.Id,
			input.Auth.Scope.Project.Domain.Name)
		if err != nil {
			return nil, errors.Wrap(err, "ProjectManager.FetchProject")
		}
		if project.Enabled.IsFalse() {
			return nil, ErrProjectDisabled
		}
		projExt, err = project.FetchExtend()
		if err != nil {
			return nil, errors.Wrap(err, "project.FetchExtend")
		}
		token.ProjectId = project.Id
	} else {
		domain, err = models.DomainManager.FetchDomain(input.Auth.Scope.Domain.Id,
			input.Auth.Scope.Domain.Name)
		if err != nil {
			return nil, errors.Wrap(err, "DomainManager.FetchDomain")
		}
		if domain.Enabled.IsFalse() {
			return nil, ErrDomainDisabled
		}
		token.DomainId = domain.Id
	}
	return token.getTokenV3(ctx, user, projExt, domain)
}

func AuthenticateV2(ctx context.Context, input mcclient.SAuthenticationInputV2) (*mcclient.TokenCredentialV2, error) {
	var user *models.SUserExtended
	var err error
	var method string
	if len(input.Auth.Token.Id) > 0 {
		// auth by token
		user, err = authUserByTokenV2(ctx, input)
		if err != nil {
			return nil, errors.Wrap(err, "authUserByTokenV2")
		}
		method = api.AUTH_METHOD_TOKEN
	} else {
		// auth by password
		user, err = authUserByPasswordV2(ctx, input)
		if err != nil {
			return nil, errors.Wrap(err, "authUserByPasswordV2")
		}
		method = api.AUTH_METHOD_PASSWORD
	}
	// user not found
	if user == nil {
		return nil, ErrUserNotFound
	}
	// user is not enabled
	if !user.Enabled {
		return nil, ErrUserDisabled
	}
	token := SAuthToken{}
	token.UserId = user.Id
	token.Method = method
	token.AuditIds = []string{utils.GenRequestId(16)}
	now := time.Now().UTC()
	token.ExpiresAt = now.Add(time.Duration(options.Options.TokenExpirationSeconds) * time.Second)
	token.Context = input.Auth.Context

	if len(input.Auth.TenantId) == 0 && len(input.Auth.TenantName) == 0 {
		// unscoped auth
		return token.getTokenV2(ctx, user, nil)
	}
	project, err := models.ProjectManager.FetchProject(
		input.Auth.TenantId,
		input.Auth.TenantName,
		api.DEFAULT_DOMAIN_ID, "")
	if err != nil {
		return nil, errors.Wrap(err, "ProjectManager.FetchProject")
	}
	if project.Enabled.IsFalse() {
		return nil, ErrProjectDisabled
	}
	token.ProjectId = project.Id
	projExt, err := project.FetchExtend()
	if err != nil {
		return nil, errors.Wrap(err, "project.FetchExtend")
	}

	return token.getTokenV2(ctx, user, projExt)
}
