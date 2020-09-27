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

package mcclient

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SSimpleToken struct {
	Token     string
	Domain    string
	DomainId  string
	User      string
	UserId    string
	Project   string `json:"tenant"`
	ProjectId string `json:"tenant_id"`

	ProjectDomain   string
	ProjectDomainId string

	Roles   string
	RoleIds string
	Expires time.Time

	Context SAuthContext
}

func (self *SSimpleToken) GetTokenString() string {
	return self.Token
}

func (self *SSimpleToken) GetDomainId() string {
	return self.DomainId
}

func (self *SSimpleToken) GetDomainName() string {
	return self.Domain
}

func (self *SSimpleToken) GetTenantId() string {
	return self.ProjectId
}

func (self *SSimpleToken) GetTenantName() string {
	return self.Project
}

func (self *SSimpleToken) GetProjectId() string {
	return self.ProjectId
}

func (self *SSimpleToken) GetProjectName() string {
	return self.Project
}

func (self *SSimpleToken) GetProjectDomainId() string {
	if len(self.ProjectDomainId) == 0 {
		return api.DEFAULT_DOMAIN_ID
	}
	return self.ProjectDomainId
}

func (self *SSimpleToken) GetProjectDomain() string {
	if len(self.ProjectDomain) == 0 {
		return api.DEFAULT_DOMAIN_NAME
	}
	return self.ProjectDomain
}

func (self *SSimpleToken) GetUserId() string {
	return self.UserId
}

func (self *SSimpleToken) GetUserName() string {
	return self.User
}

func (self *SSimpleToken) GetRoles() []string {
	return strings.Split(self.Roles, ",")
}

func (self *SSimpleToken) GetRoleIds() []string {
	return strings.Split(self.RoleIds, ",")
}

func (self *SSimpleToken) GetExpires() time.Time {
	return self.Expires
}

func (self *SSimpleToken) IsValid() bool {
	return self.ValidDuration() > 0
}

func (self *SSimpleToken) ValidDuration() time.Duration {
	return time.Until(self.Expires)
}

func (self *SSimpleToken) IsAdmin() bool {
	roles := self.GetRoles()
	for i := 0; i < len(roles); i++ {
		if roles[i] == "admin" {
			return true
		}
	}
	return false
}

func (self *SSimpleToken) HasSystemAdminPrivilege() bool {
	return self.IsAdmin() && self.Project == "system"
}

func (this *SSimpleToken) IsAllow(scope rbacutils.TRbacScope, service string, resource string, action string, extra ...string) bool {
	if scope == rbacutils.ScopeSystem || scope == rbacutils.ScopeDomain {
		return this.HasSystemAdminPrivilege()
	} else {
		return true
	}
}

func (self *SSimpleToken) GetRegions() []string {
	return nil
}

func (self *SSimpleToken) Len() int {
	return 0
}

func (self *SSimpleToken) GetServiceURL(service, region, zone, endpointType string) (string, error) {
	return "", fmt.Errorf("Not available")
}

func (self *SSimpleToken) GetServiceURLs(service, region, zone, endpointType string) ([]string, error) {
	return nil, fmt.Errorf("Not available")
}

func (this *SSimpleToken) GetServicesByInterface(region string, infType string) []ExternalService {
	return nil
}

func (self *SSimpleToken) GetInternalServices(region string) []string {
	return nil
}

func (self *SSimpleToken) GetExternalServices(region string) []ExternalService {
	return nil
}

func (this *SSimpleToken) GetEndpoints(region string, endpointType string) []Endpoint {
	return nil
}

func (this *SSimpleToken) GetServiceCatalog() IServiceCatalog {
	return nil
}

func (this *SSimpleToken) GetLoginSource() string {
	return this.Context.Source
}

func (this *SSimpleToken) GetLoginIp() string {
	return this.Context.Ip
}

func SimplifyToken(token TokenCredential) TokenCredential {
	simToken, ok := token.(*SSimpleToken)
	if ok {
		return simToken
	}
	return &SSimpleToken{Token: token.GetTokenString(),
		Domain:    token.GetDomainName(),
		DomainId:  token.GetDomainId(),
		User:      token.GetUserName(),
		UserId:    token.GetUserId(),
		Project:   token.GetProjectName(),
		ProjectId: token.GetProjectId(),

		ProjectDomain:   token.GetProjectDomain(),
		ProjectDomainId: token.GetProjectDomainId(),

		Roles:   strings.Join(token.GetRoles(), ","),
		RoleIds: strings.Join(token.GetRoleIds(), ","),
		Expires: token.GetExpires(),
		Context: SAuthContext{
			Source: token.GetLoginSource(),
			Ip:     token.GetLoginIp(),
		},
	}
}

func (self *SSimpleToken) GetCatalogData(serviceTypes []string, region string) jsonutils.JSONObject {
	return nil
}

func (self *SSimpleToken) String() string {
	return self.ToJson().String()
}

func (self *SSimpleToken) IsZero() bool {
	return len(self.UserId) == 0 && len(self.ProjectId) == 0
}

func (self *SSimpleToken) ToJson() jsonutils.JSONObject {
	return jsonutils.Marshal(self)
}

var TokenCredentialType reflect.Type

func init() {
	TokenCredentialType = reflect.TypeOf((*TokenCredential)(nil)).Elem()

	gotypes.RegisterSerializable(TokenCredentialType, func() gotypes.ISerializable {
		return &SSimpleToken{}
	})
}
