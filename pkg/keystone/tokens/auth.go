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
	"database/sql"
	"fmt"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/driver"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/keystone/options"
	"yunion.io/x/onecloud/pkg/keystone/saml"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/s3auth"
)

func authUserByTokenV2(ctx context.Context, input mcclient.SAuthenticationInputV2) (*api.SUserExtended, error) {
	return authUserByToken(ctx, input.Auth.Token.Id)
}

func authUserByTokenV3(ctx context.Context, input mcclient.SAuthenticationInputV3) (*api.SUserExtended, error) {
	return authUserByToken(ctx, input.Auth.Identity.Token.Id)
}

func authUserByToken(ctx context.Context, tokenStr string) (*api.SUserExtended, error) {
	token := SAuthToken{}
	err := token.ParseFernetToken(tokenStr)
	if err != nil {
		return nil, errors.Wrap(err, "token.ParseFernetToken")
	}
	return models.UserManager.FetchUserExtended(token.UserId, "", "", "")
}

func authUserByPasswordV2(ctx context.Context, input mcclient.SAuthenticationInputV2) (*api.SUserExtended, error) {
	ident := mcclient.SAuthenticationIdentity{}
	ident.Methods = []string{api.AUTH_METHOD_PASSWORD}
	ident.Password.User.Name = input.Auth.PasswordCredentials.Username
	ident.Password.User.Password = input.Auth.PasswordCredentials.Password
	ident.Password.User.Domain.Id = api.DEFAULT_DOMAIN_ID
	return authUserByIdentity(ctx, ident)
}

func authUserByIdentityV3(ctx context.Context, input mcclient.SAuthenticationInputV3) (*api.SUserExtended, error) {
	return authUserByIdentity(ctx, input.Auth.Identity)
}

func authUserByIdentity(ctx context.Context, ident mcclient.SAuthenticationIdentity) (*api.SUserExtended, error) {
	var idpId string

	if len(ident.Password.User.Name) == 0 && len(ident.Password.User.Id) == 0 && len(ident.Password.User.Domain.Id) == 0 && len(ident.Password.User.Domain.Name) == 0 {
		return nil, ErrEmptyAuth
	}
	if len(ident.Password.User.Name) > 0 && len(ident.Password.User.Id) == 0 && len(ident.Password.User.Domain.Id) == 0 && len(ident.Password.User.Domain.Name) == 0 {
		// no use domain specified, try to find use domain
		users := models.UserManager.Query().SubQuery()
		idMappings := models.IdmappingManager.Query().SubQuery()
		q := users.Query()
		q = q.LeftJoin(idMappings, sqlchemy.Equals(idMappings.Field("public_id"), users.Field("id")))
		q = q.Filter(sqlchemy.Equals(users.Field("name"), ident.Password.User.Name))
		q = q.Filter(sqlchemy.OR(
			sqlchemy.IsNull(idMappings.Field("domain_id")),
			sqlchemy.In(idMappings.Field("domain_id"), models.IdentityProviderManager.FetchPasswordProtectedIdpIdsQuery()),
		))
		usrCnt, err := q.CountWithError()
		if err != nil {
			return nil, errors.Wrap(err, "Query user by name")
		}
		if usrCnt > 1 {
			return nil, sqlchemy.ErrDuplicateEntry
		} else if usrCnt == 0 {
			/*idp, err := models.IdentityProviderManager.GetAutoCreateUserProvider()
			if err != nil {
				return nil, errors.Wrap(err, "IdentityProviderManager.GetAutoCreateUserProvider")
			}
			idpId = idp.Id
			*/
			return nil, sqlchemy.ErrEmptyQuery
		} else {
			// userCnt == 1
			usr := models.SUser{}
			usr.SetModelManager(models.UserManager, &usr)
			err := q.First(&usr)
			if err != nil {
				return nil, errors.Wrap(err, "Query user")
			}
			ident.Password.User.Domain.Id = usr.DomainId
			idps, err := models.IdentityProviderManager.FetchIdentityProvidersByUserId(usr.Id, api.PASSWORD_PROTECTED_IDPS)
			if err != nil {
				return nil, errors.Wrap(err, "IdentityProviderManager.FetchIdentityProvidersByUserId")
			}
			if len(idps) == 0 {
				idpId = api.DEFAULT_IDP_ID
			} else if len(idps) == 1 {
				idpId = idps[0].Id
			} else {
				return nil, sqlchemy.ErrDuplicateEntry
			}
		}
	} else {
		usrExt, err := models.UserManager.FetchUserExtended(ident.Password.User.Id, ident.Password.User.Name,
			ident.Password.User.Domain.Id, ident.Password.User.Domain.Name)
		if err != nil && err != sql.ErrNoRows {
			return nil, errors.Wrap(err, "UserManager.FetchUserExtended")
		}

		if err == sql.ErrNoRows {
			// no such user locally, query domain idp
			domain, err := models.DomainManager.FetchDomain(ident.Password.User.Domain.Id, ident.Password.User.Domain.Name)
			if err != nil {
				return nil, errors.Wrap(err, "DomainManager.FetchDomain")
			}
			mapping, err := models.IdmappingManager.FetchFirstEntity(domain.Id, api.IdMappingEntityDomain)
			if err != nil {
				return nil, errors.Wrap(err, "IdmappingManager.FetchEntity")
			}
			idpId = mapping.IdpId
		} else {
			// user exists, query user's idp
			idps, err := models.IdentityProviderManager.FetchIdentityProvidersByUserId(usrExt.Id, api.PASSWORD_PROTECTED_IDPS)
			if err != nil {
				return nil, errors.Wrap(err, "IdentityProviderManager.FetchIdentityProvidersByUserId")
			}
			if len(idps) == 0 {
				idpId = api.DEFAULT_IDP_ID
			} else if len(idps) == 1 {
				idpId = idps[0].Id
			} else {
				return nil, sqlchemy.ErrDuplicateEntry
			}
		}
	}

	if len(idpId) == 0 {
		idpId = api.DEFAULT_IDP_ID
	}
	idpObj, err := models.IdentityProviderManager.FetchById(idpId)
	if err != nil {
		return nil, errors.Wrap(err, "IdentityProviderManager.FetchById")
	}

	idp := idpObj.(*models.SIdentityProvider)

	if idp.Status != api.IdentityDriverStatusConnected && idp.Status != api.IdentityDriverStatusDisconnected {
		return nil, errors.Error(fmt.Sprintf("invalid idp status %s", idp.Status))
	}

	conf, err := models.GetConfigs(idp, true, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "GetConfig")
	}

	backend, err := driver.GetDriver(idp.Driver, idp.Id, idp.Name, idp.Template, idp.TargetDomainId, conf)
	if err != nil {
		return nil, errors.Wrap(err, "driver.GetDriver")
	}

	usr, err := backend.Authenticate(ctx, ident)
	if err != nil {
		return nil, errors.Wrap(err, "Authenticate")
	}

	if idp.Status == api.IdentityDriverStatusDisconnected {
		idp.MarkConnected(ctx, models.GetDefaultAdminCred())
	}

	return usr, nil
}

func authUserByCASV3(ctx context.Context, input mcclient.SAuthenticationInputV3) (*api.SUserExtended, error) {
	var idp *models.SIdentityProvider
	var err error
	if len(input.Auth.Identity.Id) > 0 {
		idp, err = models.IdentityProviderManager.FetchIdentityProviderById(input.Auth.Identity.Id)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "idp %s not found", input.Auth.Identity.Id)
			} else {
				return nil, errors.Wrap(err, "FetchIdentityProviderById")
			}
		}
	} else {
		idps, err := models.IdentityProviderManager.FetchEnabledProviders(api.IdentityDriverCAS)
		if err != nil {
			return nil, errors.Wrap(err, "models.fetchEnabledProviders")
		}
		if len(idps) == 0 {
			return nil, errors.Error("No cas identity provider")
		}
		if len(idps) > 1 {
			return nil, errors.Error("more than 1 cas identity providers?")
		}
		idp = &idps[0]
	}

	conf, err := models.GetConfigs(idp, true, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "idp.GetConfig")
	}

	backend, err := driver.GetDriver(idp.Driver, idp.Id, idp.Name, idp.Template, idp.TargetDomainId, conf)
	if err != nil {
		return nil, errors.Wrap(err, "driver.GetDriver")
	}

	usr, err := backend.Authenticate(ctx, input.Auth.Identity)
	if err != nil {
		return nil, errors.Wrap(err, "Authenticate")
	}

	if idp.Status == api.IdentityDriverStatusDisconnected {
		idp.MarkConnected(ctx, models.GetDefaultAdminCred())
	}

	return usr, nil
}

func authUserBySAML(ctx context.Context, input mcclient.SAuthenticationInputV3) (*api.SUserExtended, error) {
	if !saml.IsSAMLEnabled() {
		return nil, errors.Wrap(httperrors.ErrNotSupported, "unsupported SAML backend")
	}

	idp, err := models.IdentityProviderManager.FetchIdentityProviderById(input.Auth.Identity.Id)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "idp %s not found", input.Auth.Identity.Id)
		} else {
			return nil, errors.Wrap(err, "FetchIdentityProviderById")
		}
	}

	conf, err := models.GetConfigs(idp, true, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "idp.GetConfig")
	}

	backend, err := driver.GetDriver(idp.Driver, idp.Id, idp.Name, idp.Template, idp.TargetDomainId, conf)
	if err != nil {
		return nil, errors.Wrap(err, "driver.GetDriver")
	}

	usr, err := backend.Authenticate(ctx, input.Auth.Identity)
	if err != nil {
		return nil, errors.Wrap(err, "Authenticate")
	}

	if idp.Status == api.IdentityDriverStatusDisconnected {
		idp.MarkConnected(ctx, models.GetDefaultAdminCred())
	}

	return usr, nil
}

func authUserByOIDC(ctx context.Context, input mcclient.SAuthenticationInputV3) (*api.SUserExtended, error) {
	idp, err := models.IdentityProviderManager.FetchIdentityProviderById(input.Auth.Identity.Id)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "idp %s not found", input.Auth.Identity.Id)
		} else {
			return nil, errors.Wrap(err, "FetchIdentityProviderById")
		}
	}

	conf, err := models.GetConfigs(idp, true, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "idp.GetConfig")
	}

	backend, err := driver.GetDriver(idp.Driver, idp.Id, idp.Name, idp.Template, idp.TargetDomainId, conf)
	if err != nil {
		return nil, errors.Wrap(err, "driver.GetDriver")
	}

	usr, err := backend.Authenticate(ctx, input.Auth.Identity)
	if err != nil {
		return nil, errors.Wrap(err, "Authenticate")
	}

	if idp.Status == api.IdentityDriverStatusDisconnected {
		idp.MarkConnected(ctx, models.GetDefaultAdminCred())
	}

	return usr, nil
}

func authUserByOAuth2(ctx context.Context, input mcclient.SAuthenticationInputV3) (*api.SUserExtended, error) {
	idp, err := models.IdentityProviderManager.FetchIdentityProviderById(input.Auth.Identity.Id)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "idp %s not found", input.Auth.Identity.Id)
		} else {
			return nil, errors.Wrap(err, "FetchIdentityProviderById")
		}
	}

	conf, err := models.GetConfigs(idp, true, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "idp.GetConfig")
	}

	backend, err := driver.GetDriver(idp.Driver, idp.Id, idp.Name, idp.Template, idp.TargetDomainId, conf)
	if err != nil {
		return nil, errors.Wrap(err, "driver.GetDriver")
	}

	usr, err := backend.Authenticate(ctx, input.Auth.Identity)
	if err != nil {
		return nil, errors.Wrap(err, "Authenticate")
	}

	if idp.Status == api.IdentityDriverStatusDisconnected {
		idp.MarkConnected(ctx, models.GetDefaultAdminCred())
	}

	return usr, nil
}

func authUserByAccessKeyV3(ctx context.Context, input mcclient.SAuthenticationInputV3) (*api.SUserExtended, string, api.SAccessKeySecretInfo, error) {
	var aksk api.SAccessKeySecretInfo

	akskRequest, err := s3auth.Decode(input.Auth.Identity.AccessKeyRequest)
	if err != nil {
		return nil, "", aksk, errors.Wrap(err, "s3auth.Decode")
	}
	keyId := akskRequest.GetAccessKey()
	obj, err := models.CredentialManager.FetchById(keyId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, "", aksk, ErrInvalidAccessKeyId
		} else {
			return nil, "", aksk, errors.Wrap(err, "CredentialManager.FetchById")
		}
	}
	credential := obj.(*models.SCredential)
	if !credential.Enabled.IsTrue() {
		return nil, "", aksk, errors.Wrap(httperrors.ErrInvalidStatus, "Access Key disabled")
	}
	akBlob, err := credential.GetAccessKeySecret()
	if err != nil {
		return nil, "", aksk, errors.Wrap(err, "credential.GetAccessKeySecret")
	}
	if !akBlob.IsValid() {
		return nil, "", aksk, ErrExpiredAccessKey
	}
	aksk.AccessKey = keyId
	aksk.Secret = akBlob.Secret
	aksk.Expire = akBlob.Expire

	err = akskRequest.Verify(akBlob.Secret)
	if err != nil {
		return nil, "", aksk, errors.Wrap(err, "Verify")
	}
	usrExt, err := models.UserManager.FetchUserExtended(credential.UserId, "", "", "")
	if err != nil {
		return nil, "", aksk, errors.Wrap(err, "UserManager.FetchUserExtended")
	}
	return usrExt, credential.ProjectId, aksk, nil
}

// +onecloud:swagger-gen-route-method=POST
// +onecloud:swagger-gen-route-path=/v3/auth/tokens
// +onecloud:swagger-gen-route-tag=authentication
// +onecloud:swagger-gen-param-body-index=1
// +onecloud:swagger-gen-resp-index=0
// +onecloud:swagger-gen-resp-header=X-Subject-Token
// +onecloud:swagger-gen-resp-header=验证成功的keystone V3 token

// keystone v3认证API
func AuthenticateV3(ctx context.Context, input mcclient.SAuthenticationInputV3) (*mcclient.TokenCredentialV3, error) {
	var akskInfo api.SAccessKeySecretInfo
	var user *api.SUserExtended
	var err error
	if len(input.Auth.Identity.Methods) != 1 {
		return nil, ErrInvalidAuthMethod
	}
	method := input.Auth.Identity.Methods[0]
	switch method {
	case api.AUTH_METHOD_TOKEN:
		// auth by token
		user, err = authUserByTokenV3(ctx, input)
		if err != nil {
			return nil, errors.Wrap(err, "authUserByTokenV3")
		}
	case api.AUTH_METHOD_AKSK:
		// auth by aksk
		user, input.Auth.Scope.Project.Id, akskInfo, err = authUserByAccessKeyV3(ctx, input)
		if err != nil {
			return nil, errors.Wrap(err, "authUserByAccessKeyV3")
		}
	case api.AUTH_METHOD_CAS:
		// auth by apereo CAS
		user, err = authUserByCASV3(ctx, input)
		if err != nil {
			return nil, errors.Wrap(err, "authUserByCASV3")
		}
	case api.AUTH_METHOD_SAML:
		// auth by SAML 2.0 IDP, keystone acts as a SAML SP
		user, err = authUserBySAML(ctx, input)
		if err != nil {
			return nil, errors.Wrap(err, "authUserBySAML")
		}
	case api.AUTH_METHOD_OIDC:
		// auth by OpenID Connect, keystone acts as an OpenID Connect client
		user, err = authUserByOIDC(ctx, input)
		if err != nil {
			return nil, errors.Wrap(err, "authUserByOIDC")
		}
	case api.AUTH_METHOD_OAuth2:
		// auth by customized OAuth2.0 provider, keystone acts as an OAuth2.0 app
		user, err = authUserByOAuth2(ctx, input)
		if err != nil {
			return nil, errors.Wrap(err, "authUserByOAuth2")
		}
	default:
		// auth by other methods, e.g. password , etc...
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

	if !user.DomainEnabled {
		return nil, ErrDomainDisabled
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
		return token.getTokenV3(ctx, user, nil, nil, akskInfo)
	}
	var projExt *models.SProjectExtended
	var domain *models.SDomain
	if len(input.Auth.Scope.Project.Id) > 0 || len(input.Auth.Scope.Project.Name) > 0 {
		project, err := models.ProjectManager.FetchProject(
			input.Auth.Scope.Project.Id,
			input.Auth.Scope.Project.Name,
			input.Auth.Scope.Project.Domain.Id,
			input.Auth.Scope.Project.Domain.Name,
		)
		if err != nil {
			return nil, errors.Wrap(err, "ProjectManager.FetchProject")
		}
		// if project.Enabled.IsFalse() {
		// 	return nil, ErrProjectDisabled
		// }
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
	tokenV3, err := token.getTokenV3(ctx, user, projExt, domain, akskInfo)
	if err != nil {
		return nil, errors.Wrap(err, "getTokenV3")
	}
	return tokenV3, nil
}

type SAuthenticateV2ResponseBody struct {
	Access mcclient.TokenCredentialV2 `json:"access"`
}

// +onecloud:swagger-gen-route-method=POST
// +onecloud:swagger-gen-route-path=/v2.0/tokens
// +onecloud:swagger-gen-route-tag=authentication
// +onecloud:swagger-gen-param-body-index=1
// +onecloud:swagger-gen-resp-index=0

// keystone v2 认证接口，通过用户名/密码或者 token 认证
func AuthenticateV2(ctx context.Context, input mcclient.SAuthenticationInputV2) (*SAuthenticateV2ResponseBody, error) {
	token, err := _authenticateV2(ctx, input)
	if err != nil {
		return nil, errors.Wrap(err, "_authenticateV2")
	}
	body := SAuthenticateV2ResponseBody{
		Access: *token,
	}
	return &body, nil
}

func _authenticateV2(ctx context.Context, input mcclient.SAuthenticationInputV2) (*mcclient.TokenCredentialV2, error) {
	var user *api.SUserExtended
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

	if !user.DomainEnabled {
		return nil, ErrDomainDisabled
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
	// if project.Enabled.IsFalse() {
	// 	return nil, ErrProjectDisabled
	// }
	token.ProjectId = project.Id
	projExt, err := project.FetchExtend()
	if err != nil {
		return nil, errors.Wrap(err, "project.FetchExtend")
	}

	return token.getTokenV2(ctx, user, projExt)
}
