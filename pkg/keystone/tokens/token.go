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
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/cache"
	"yunion.io/x/onecloud/pkg/keystone/keys"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/keystone/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	defaultAuthToken    *SAuthToken
	defaultAuthTokenStr string
)

func GetDefaultToken() string {
	now := time.Now()
	if defaultAuthToken == nil || defaultAuthToken.ExpiresAt.Sub(now) < time.Duration(3600) {
		simpleToken := models.GetDefaultAdminCred()
		defaultAuthToken = &SAuthToken{
			UserId:    simpleToken.GetUserId(),
			Method:    api.AUTH_METHOD_TOKEN,
			ProjectId: simpleToken.GetProjectId(),
			ExpiresAt: now.Add(time.Duration(options.Options.TokenExpirationSeconds) * time.Second),
			AuditIds:  []string{utils.GenRequestId(16)},
		}
		var err error
		defaultAuthTokenStr, err = defaultAuthToken.encodeFernetToken()
		if err != nil {
			log.Fatalf("defaultAuthToken.EncodeFernetToken fail: %s", err)
		}
		if simpleToken.(*mcclient.SSimpleToken).Token != defaultAuthTokenStr {
			simpleToken.(*mcclient.SSimpleToken).Token = defaultAuthTokenStr
		}
	}
	return defaultAuthTokenStr
}

func GetDefaultAdminCredToken() mcclient.TokenCredential {
	GetDefaultToken()
	return models.GetDefaultAdminCred()
}

type SAuthToken struct {
	UserId    string
	Method    string
	ProjectId string
	DomainId  string
	ExpiresAt time.Time
	AuditIds  []string

	Context mcclient.SAuthContext
}

func (t *SAuthToken) Decode(tk []byte) error {
	for _, payload := range []ITokenPayload{
		&SProjectScopedPayloadWithContext{},
		&SDomainScopedPayloadWithContext{},
		&SUnscopedPayloadWithContext{},
		&SProjectScopedPayload{},
		&SDomainScopedPayload{},
		&SUnscopedPayload{},
	} {
		err := payload.Unmarshal(tk)
		if err == nil {
			payload.Decode(t)
			return nil
		}
	}
	return ErrInvalidFernetToken
}

func (t *SAuthToken) getProjectScopedPayload() ITokenPayload {
	p := SProjectScopedPayload{}
	p.Version = SProjectScopedPayloadVersion
	p.UserId.parse(t.UserId)
	p.ProjectId.parse(t.ProjectId)
	p.Method = authMethodStr2Id(t.Method)
	p.ExpiresAt = float64(t.ExpiresAt.Unix())
	p.AuditIds = auditStrings2Bytes(t.AuditIds)
	return &p
}

func (t *SAuthToken) getProjectScopedPayloadWithContext() ITokenPayload {
	p := SProjectScopedPayloadWithContext{}
	p.Version = SProjectScopedPayloadWithContextVersion
	p.UserId.parse(t.UserId)
	p.ProjectId.parse(t.ProjectId)
	p.Method = authMethodStr2Id(t.Method)
	p.ExpiresAt = float64(t.ExpiresAt.Unix())
	p.AuditIds = auditStrings2Bytes(t.AuditIds)
	p.Context = authContext2Payload(t.Context)
	return &p
}

func (t *SAuthToken) getDomainScopedPayload() ITokenPayload {
	p := SDomainScopedPayload{}
	p.Version = SDomainScopedPayloadVersion
	p.UserId.parse(t.UserId)
	p.DomainId.parse(t.DomainId)
	p.Method = authMethodStr2Id(t.Method)
	p.ExpiresAt = float64(t.ExpiresAt.Unix())
	p.AuditIds = auditStrings2Bytes(t.AuditIds)
	return &p
}

func (t *SAuthToken) getDomainScopedPayloadWithContext() ITokenPayload {
	p := SDomainScopedPayloadWithContext{}
	p.Version = SDomainScopedPayloadVersion
	p.UserId.parse(t.UserId)
	p.DomainId.parse(t.DomainId)
	p.Method = authMethodStr2Id(t.Method)
	p.ExpiresAt = float64(t.ExpiresAt.Unix())
	p.AuditIds = auditStrings2Bytes(t.AuditIds)
	p.Context = authContext2Payload(t.Context)
	return &p
}

func (t *SAuthToken) getUnscopedPayload() ITokenPayload {
	p := SUnscopedPayload{}
	p.Version = SUnscopedPayloadVersion
	p.UserId.parse(t.UserId)
	p.Method = authMethodStr2Id(t.Method)
	p.ExpiresAt = float64(t.ExpiresAt.Unix())
	p.AuditIds = auditStrings2Bytes(t.AuditIds)
	return &p
}

func (t *SAuthToken) getUnscopedPayloadWithContext() ITokenPayload {
	p := SUnscopedPayloadWithContext{}
	p.Version = SUnscopedPayloadVersion
	p.UserId.parse(t.UserId)
	p.Method = authMethodStr2Id(t.Method)
	p.ExpiresAt = float64(t.ExpiresAt.Unix())
	p.AuditIds = auditStrings2Bytes(t.AuditIds)
	p.Context = authContext2Payload(t.Context)
	return &p
}

func (t *SAuthToken) getPayload() ITokenPayload {
	if len(t.ProjectId) > 0 {
		return t.getProjectScopedPayloadWithContext()
	}
	if len(t.DomainId) > 0 {
		return t.getDomainScopedPayloadWithContext()
	}
	return t.getUnscopedPayloadWithContext()
}

func (t *SAuthToken) Encode() ([]byte, error) {
	return t.getPayload().Encode()
}

func TokenStrDecode(tokenStr string) (*SAuthToken, error) {
	token, err := models.TokenCacheManager.FetchToken(tokenStr)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			// not found
			token := &SAuthToken{}
			err := token.parseFernetToken(tokenStr)
			if err != nil {
				return nil, errors.Wrap(err, "parseFernetToken")
			}
			return token, nil
		} else {
			return nil, errors.Wrap(err, "FetchToken")
		}
	} else {
		if !token.Valid {
			return nil, errors.Wrap(httperrors.ErrInvalidCredential, "invalid token")
		}
		return &SAuthToken{
			UserId:    token.UserId,
			ProjectId: token.ProjectId,
			DomainId:  token.DomainId,

			Method:    token.Method,
			ExpiresAt: token.ExpiredAt,
			AuditIds:  strings.Split(token.AuditIds, ","),

			Context: mcclient.SAuthContext{
				Source: token.Source,
				Ip:     token.Ip,
			},
		}, nil
	}
}

func (t *SAuthToken) parseFernetToken(tokenStr string) error {
	tk := keys.TokenKeysManager.Decrypt([]byte(tokenStr)) // , time.Duration(options.Options.TokenExpirationSeconds)*time.Second)
	if tk == nil {
		return errors.Wrapf(ErrInvalidToken, tokenStr)
	}
	err := t.Decode(tk)
	if err != nil {
		return errors.Wrap(err, "decode error")
	}
	if t.ExpiresAt.Before(time.Now()) {
		return errors.Wrapf(ErrExpiredToken, "expires_at %s", t.ExpiresAt.String())
	}
	return nil
}

func (t *SAuthToken) encodeFernetToken() (string, error) {
	tk, err := t.Encode()
	if err != nil {
		return "", errors.Wrap(err, "encode error")
	}
	ftk, err := keys.TokenKeysManager.Encrypt(tk)
	if err != nil {
		return "", errors.Wrap(err, "TokenKeysManager.Encrypt")
	}
	return string(ftk), nil
}

func (t *SAuthToken) encodeShortToken() (string, error) {
	tk, err := t.encodeFernetToken()
	if err != nil {
		return "", errors.Wrap(err, "encodeFernetToken")
	}
	return stringutils2.GenId(tk), nil
}

func (t *SAuthToken) GetSimpleUserCred(token string) (mcclient.TokenCredential, error) {
	userExt, err := models.UserManager.FetchUserExtended(t.UserId, "", "", "")
	if err != nil {
		return nil, errors.Wrap(err, "UserManager.FetchUserExtended")
	}
	ret := mcclient.SSimpleToken{
		Token:    token,
		UserId:   t.UserId,
		User:     userExt.Name,
		Domain:   userExt.DomainName,
		DomainId: userExt.DomainId,
		Expires:  t.ExpiresAt,
		Context:  t.Context,
	}
	var roles []models.SRole
	if len(t.ProjectId) > 0 {
		proj, err := models.ProjectManager.FetchProjectById(t.ProjectId)
		if err != nil {
			return nil, errors.Wrap(err, "ProjectManager.FetchProjectById")
		}
		ret.ProjectId = t.ProjectId
		ret.Project = proj.Name
		ret.ProjectDomainId = proj.DomainId
		ret.ProjectDomain = proj.GetDomain().Name
		roles, err = models.AssignmentManager.FetchUserProjectRoles(t.UserId, t.ProjectId)
	} else if len(t.DomainId) > 0 {
		domain, err := models.DomainManager.FetchDomainById(t.DomainId)
		if err != nil {
			return nil, errors.Wrap(err, "DomainManager.FetchDomainById")
		}
		ret.ProjectDomainId = t.DomainId
		ret.ProjectDomain = domain.Name
		roles, err = models.AssignmentManager.FetchUserProjectRoles(t.UserId, t.DomainId)
	}
	roleStrs := make([]string, len(roles))
	roleIdStrs := make([]string, len(roles))
	for i := range roles {
		roleStrs[i] = roles[i].Name
		roleIdStrs[i] = roles[i].Id
	}
	ret.Roles = strings.Join(roleStrs, ",")
	ret.RoleIds = strings.Join(roleIdStrs, ",")
	return &ret, nil
}

func (t *SAuthToken) getRoles() ([]models.SRole, error) {
	var roleProjectId string
	if len(t.ProjectId) > 0 {
		roleProjectId = t.ProjectId
	} else if len(t.DomainId) > 0 {
		roleProjectId = t.DomainId
	}
	if len(roleProjectId) > 0 {
		return models.AssignmentManager.FetchUserProjectRoles(t.UserId, roleProjectId)
	}
	return nil, nil
}

func (t *SAuthToken) getTokenV3(
	ctx context.Context,
	user *api.SUserExtended,
	project *models.SProjectExtended,
	domain *models.SDomain,
	akskInfo api.SAccessKeySecretInfo,
) (*mcclient.TokenCredentialV3, error) {
	token := mcclient.TokenCredentialV3{}
	token.Token.AccessKey = akskInfo
	token.Token.ExpiresAt = t.ExpiresAt
	token.Token.IssuedAt = t.ExpiresAt.Add(-time.Duration(options.Options.TokenExpirationSeconds) * time.Second)
	token.Token.AuditIds = t.AuditIds
	token.Token.Methods = []string{t.Method}
	token.Token.User.Id = user.Id
	token.Token.User.Name = user.Name
	token.Token.User.Domain.Id = user.DomainId
	token.Token.User.Domain.Name = user.DomainName
	lastPass, _ := models.PasswordManager.FetchLastPassword(user.LocalId)
	if lastPass != nil && !lastPass.ExpiresAt.IsZero() {
		token.Token.User.PasswordExpiresAt = lastPass.ExpiresAt
	}
	token.Token.User.Displayname = user.Displayname
	token.Token.User.Email = user.Email
	token.Token.User.Mobile = user.Mobile
	token.Token.User.IsSystemAccount = user.IsSystemAccount
	token.Token.Context = t.Context

	tk, err := t.encodeShortToken()
	if err != nil {
		return nil, errors.Wrap(err, "EncodeFernetToken")
	}
	token.Id = tk

	roles, err := t.getRoles()
	if err != nil {
		return nil, errors.Wrap(err, "getRoles")
	}

	if len(roles) == 0 {
		if project != nil || domain != nil {
			return nil, ErrUserNotInProject
		}
		/*extProjs, err := models.ProjectManager.FetchUserProjects(user.Id)
		if err != nil {
			return nil, errors.Wrap(err, "models.ProjectManager.FetchUserProjects")
		}
		token.Token.Projects = make([]mcclient.KeystoneProjectV3, len(extProjs))
		for i := range extProjs {
			token.Token.Projects[i].Id = extProjs[i].Id
			token.Token.Projects[i].Name = extProjs[i].Name
			token.Token.Projects[i].Domain.Id = extProjs[i].DomainId
			token.Token.Projects[i].Domain.Name = extProjs[i].DomainName
		}*/
		assigns, _, err := models.AssignmentManager.FetchAll(user.Id, "", "", "", "", "",
			nil, nil, nil, nil, nil, nil,
			true, true, true, true, true, 0, 0)
		if err != nil {
			return nil, errors.Wrap(err, "models.AssignmentManager.FetchAll")
		}
		token.Token.RoleAssignments = assigns
		token.Token.Projects = make([]mcclient.KeystoneProjectV3, 0)
		projMap := make(map[string]bool)
		for i := range assigns {
			if _, ok := projMap[assigns[i].Scope.Project.Id]; ok {
				continue
			}
			projMap[assigns[i].Scope.Project.Id] = true
			p := mcclient.KeystoneProjectV3{
				Id:   assigns[i].Scope.Project.Id,
				Name: assigns[i].Scope.Project.Name,
				Domain: mcclient.KeystoneDomainV3{
					Id:   assigns[i].Scope.Project.Domain.Id,
					Name: assigns[i].Scope.Project.Domain.Name,
				},
			}
			token.Token.Projects = append(token.Token.Projects, p)
		}
	} else {
		if project != nil {
			token.Token.IsDomain = false
			token.Token.Project.Id = project.Id
			token.Token.Project.Name = project.Name
			token.Token.Project.Domain.Id = project.DomainId
			token.Token.Project.Domain.Name = project.DomainName
		} else if domain != nil {
			token.Token.IsDomain = true
			token.Token.Project.Id = domain.Id
			token.Token.Project.Name = domain.Name
		}

		token.Token.Roles = make([]mcclient.KeystoneRoleV3, len(roles))
		for i := range roles {
			token.Token.Roles[i].Id = roles[i].Id
			token.Token.Roles[i].Name = roles[i].Name
		}

		policyNames, _, _ := models.RolePolicyManager.GetMatchPolicyGroup(&token, time.Time{}, true)
		token.Token.Policies.Project = policyNames[rbacscope.ScopeProject]
		token.Token.Policies.Domain = policyNames[rbacscope.ScopeDomain]
		token.Token.Policies.System = policyNames[rbacscope.ScopeSystem]

		endpoints, err := models.EndpointManager.FetchAll()
		if err != nil {
			return nil, errors.Wrap(err, "EndpointManager.FetchAll")
		}
		if endpoints != nil {
			token.Token.Catalog = endpoints.GetKeystoneCatalogV3()
		}
	}

	{
		err := models.TokenCacheManager.Save(ctx, token.Id, t.ExpiresAt, t.Method, t.AuditIds, t.UserId, t.ProjectId, t.DomainId, t.Context.Source, t.Context.Ip)
		if err != nil {
			return nil, errors.Wrap(err, "Save Token")
		}
		cache.Save(token.Id, &token)
	}

	return &token, nil
}

func (t *SAuthToken) getTokenV2(
	ctx context.Context,
	user *api.SUserExtended,
	project *models.SProjectExtended,
) (*mcclient.TokenCredentialV2, error) {
	token := mcclient.TokenCredentialV2{}
	token.User.Name = user.Name
	token.User.Id = user.Id
	token.User.Username = user.Name
	token.User.IsSystemAccount = user.IsSystemAccount
	token.Context = t.Context

	tk, err := t.encodeShortToken()
	if err != nil {
		return nil, errors.Wrap(err, "EncodeFernetToken")
	}
	token.Token.Id = tk
	token.Token.Expires = t.ExpiresAt

	roles, err := t.getRoles()
	if err != nil {
		return nil, errors.Wrap(err, "getRoles")
	}

	if len(roles) == 0 {
		if project != nil {
			return nil, ErrUserNotInProject
		}
		extProjs, err := models.ProjectManager.FetchUserProjects(user.Id)
		if err != nil {
			return nil, errors.Wrap(err, "models.ProjectManager.FetchUserProjects")
		}
		token.Tenants = make([]mcclient.KeystoneTenantV2, len(extProjs))
		for i := range extProjs {
			token.Tenants[i].Id = extProjs[i].Id
			token.Tenants[i].Name = extProjs[i].Name
			token.Tenants[i].Domain.Id = extProjs[i].DomainId
			token.Tenants[i].Domain.Name = extProjs[i].DomainName
			token.Tenants[i].Enabled = true
		}
	} else {
		token.Token.Tenant.Id = project.Id
		token.Token.Tenant.Name = project.Name
		// token.Token.Tenant.Enabled = project.Enabled.Bool()
		token.Token.Tenant.Description = project.Description
		token.Token.Tenant.Domain.Id = project.DomainId
		token.Token.Tenant.Domain.Name = project.DomainName

		token.User.Roles = make([]mcclient.KeystoneRoleV2, len(roles))
		token.Metadata.Roles = make([]string, len(roles))
		for i := range roles {
			token.User.Roles[i].Name = roles[i].Name
			token.User.Roles[i].Id = roles[i].Id
			token.Metadata.Roles[i] = roles[i].Name
		}
		endpoints, err := models.EndpointManager.FetchAll()
		if err != nil {
			return nil, errors.Wrap(err, "EndpointManager.FetchAll")
		}
		if endpoints != nil {
			token.ServiceCatalog = endpoints.GetKeystoneCatalogV2()
		}
	}

	{
		err := models.TokenCacheManager.Save(ctx, token.Token.Id, t.ExpiresAt, t.Method, t.AuditIds, t.UserId, t.ProjectId, t.DomainId, t.Context.Source, t.Context.Ip)
		if err != nil {
			return nil, errors.Wrap(err, "Save Token")
		}
		cache.Save(token.Token.Id, &token)
	}

	return &token, nil
}
