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
	"strings"
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/keystone/keys"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/keystone/options"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	defaultAuthToken *SAuthToken
)

func GetDefaultToken() (string, error) {
	now := time.Now()
	if defaultAuthToken == nil || defaultAuthToken.ExpiresAt.Sub(now) < time.Duration(3600) {
		simpleToken := models.GetDefaultAdminCred()
		defaultAuthToken = &SAuthToken{
			UserId:    simpleToken.GetUserId(),
			Method:    api.AUTH_METHOD_TOKEN,
			ProjectId: simpleToken.GetProjectId(),
			ExpiresAt: now.Add(24 * time.Hour),
			AuditIds:  []string{utils.GenRequestId(16)},
		}
	}
	return defaultAuthToken.EncodeFernetToken()
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

func (t *SAuthToken) ParseFernetToken(tokenStr string) error {
	tk := keys.TokenKeysManager.Decrypt([]byte(tokenStr), time.Duration(options.Options.TokenExpirationSeconds)*time.Second)
	if tk == nil {
		return ErrExpiredToken
	}
	err := t.Decode(tk)
	if err != nil {
		return errors.Wrap(err, "decode error")
	}
	return nil
}

func (t *SAuthToken) EncodeFernetToken() (string, error) {
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
	for i := range roles {
		roleStrs[i] = roles[i].Name
	}
	ret.Roles = strings.Join(roleStrs, ",")
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
	token.Token.Context = t.Context

	tk, err := t.EncodeFernetToken()
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
		extProjs, err := models.ProjectManager.FetchUserProjects(user.Id)
		if err != nil {
			return nil, errors.Wrap(err, "models.ProjectManager.FetchUserProjects")
		}
		token.Token.Projects = make([]mcclient.KeystoneProjectV3, len(extProjs))
		for i := range extProjs {
			token.Token.Projects[i].Id = extProjs[i].Id
			token.Token.Projects[i].Name = extProjs[i].Name
			token.Token.Projects[i].Domain.Id = extProjs[i].DomainId
			token.Token.Projects[i].Domain.Name = extProjs[i].DomainName
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

		endpoints, err := models.EndpointManager.FetchAll()
		if err != nil {
			return nil, errors.Wrap(err, "EndpointManager.FetchAll")
		}
		if endpoints != nil {
			token.Token.Catalog = endpoints.GetKeystoneCatalogV3()
		}
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
	token.Context = t.Context

	tk, err := t.EncodeFernetToken()
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

	return &token, nil
}
