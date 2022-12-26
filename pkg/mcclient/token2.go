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
	"math/rand"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type KeystoneEndpointV2 struct {
	// 接口ID
	Id string `json:"id"`
	// 内部URL
	InternalURL string `json:"internal_url"`
	// 外部URL
	PublicURL string `json:"public_url"`
	// 管理URL
	AdminURL string `json:"admin_url"`
	// 区域ID
	Region string `json:"region"`
}

type KeystoneServiceV2 struct {
	// 服务名称
	Name string `json:"name"`
	// 服务类型
	Type string `json:"type"`
	// 服务接口地址列表
	Endpoints []KeystoneEndpointV2 `json:"endpoints"`
}

type KeystoneRoleV2 struct {
	// 角色名称
	Name string `json:"name"`
	// 角色ID
	Id string `json:"id"`
}

type KeystoneUserV2 struct {
	// 用户ID
	Id string `json:"id"`
	// 用户名
	Name string `json:"name"`
	// 用户username
	Username string `json:"username"`
	// 用户角色列表
	Roles []KeystoneRoleV2 `json:"roles"`
}

type KeystoneTenantV2 struct {
	// 项目ID
	Id string `json:"id"`
	// 项目名称
	Name string `json:"name"`
	// 是否启用
	Enabled bool `json:"enabled"`
	// 描述
	Description string `json:"description"`
	// 项目归属域信息
	Domain struct {
		// 域ID
		Id string `json:"id"`
		// 域名称
		Name string `json:"name"`
	} `json:"domain"`
}

type KeystoneTokenV2 struct {
	// token
	Id string `json:"id"`
	// 过期时间(UTC)
	Expires time.Time `json:"expires"`
	// token有效的项目信息
	Tenant KeystoneTenantV2 `json:"tenant"`
}

type KeystoneMetadataV2 struct {
	// 是否为管理员
	IsAdmin int `json:"is_admin"`
	// 角色
	Roles []string `json:"roles"`
}

type KeystoneServiceCatalogV2 []KeystoneServiceV2

// Keystone token信息V2
type TokenCredentialV2 struct {
	// token信息
	Token KeystoneTokenV2 `json:"token"`
	// 服务目录
	ServiceCatalog KeystoneServiceCatalogV2 `json:"service_catalog"`
	// 认证用户信息
	User KeystoneUserV2 `json:"user"`
	// 用户所属项目列表
	Tenants []KeystoneTenantV2 `json:"tenants"`
	// 认证元数据
	Metadata KeystoneMetadataV2 `json:"metadata"`
	// 认证上下文
	Context SAuthContext `json:"context"`
}

func (token *TokenCredentialV2) GetTokenString() string {
	return token.Token.Id
}

func (token *TokenCredentialV2) GetDomainId() string {
	return api.DEFAULT_DOMAIN_ID
}

func (token *TokenCredentialV2) GetDomainName() string {
	return api.DEFAULT_DOMAIN_NAME
}

func (token *TokenCredentialV2) GetTenantId() string {
	return token.Token.Tenant.Id
}

func (token *TokenCredentialV2) GetTenantName() string {
	return token.Token.Tenant.Name
}

func (token *TokenCredentialV2) GetProjectId() string {
	return token.GetTenantId()
}

func (token *TokenCredentialV2) GetProjectName() string {
	return token.GetTenantName()
}

func (token *TokenCredentialV2) GetProjectDomainId() string {
	return api.DEFAULT_DOMAIN_ID
}

func (token *TokenCredentialV2) GetProjectDomain() string {
	return api.DEFAULT_DOMAIN_NAME
}

func (token *TokenCredentialV2) GetUserName() string {
	return token.User.Username
}

func (token *TokenCredentialV2) GetUserId() string {
	return token.User.Id
}

func (token *TokenCredentialV2) GetRoles() []string {
	roles := make([]string, 0)
	for i := 0; i < len(token.User.Roles); i++ {
		roles = append(roles, token.User.Roles[i].Name)
	}
	return roles
}

func (token *TokenCredentialV2) GetRoleIds() []string {
	roles := make([]string, 0)
	for i := 0; i < len(token.User.Roles); i++ {
		roles = append(roles, token.User.Roles[i].Id)
	}
	return roles
}

func (this *TokenCredentialV2) GetExpires() time.Time {
	return this.Token.Expires
}

func (this *TokenCredentialV2) IsValid() bool {
	return this.ValidDuration() > 0
}

func (this *TokenCredentialV2) ValidDuration() time.Duration {
	return time.Until(this.Token.Expires)
}

func (this *TokenCredentialV2) IsAdmin() bool {
	for i := 0; i < len(this.User.Roles); i++ {
		if this.User.Roles[i].Name == "admin" {
			return true
		}
	}
	return false
}

func (this *TokenCredentialV2) GetRegions() []string {
	return this.ServiceCatalog.getRegions()
}

func (this *TokenCredentialV2) HasSystemAdminPrivilege() bool {
	return this.IsAdmin() && this.GetTenantName() == "system"
}

func (this *TokenCredentialV2) IsAllow(scope rbacscope.TRbacScope, service string, resource string, action string, extra ...string) rbacutils.SPolicyResult {
	if this.isAllow(scope, service, resource, action, extra...) {
		return rbacutils.PolicyAllow
	} else {
		return rbacutils.PolicyDeny
	}
}

func (this *TokenCredentialV2) isAllow(scope rbacscope.TRbacScope, service string, resource string, action string, extra ...string) bool {
	if scope == rbacscope.ScopeSystem || scope == rbacscope.ScopeDomain {
		return this.HasSystemAdminPrivilege()
	} else {
		return true
	}
}

func (this *TokenCredentialV2) Len() int {
	return this.ServiceCatalog.Len()
}

func (this *TokenCredentialV2) getServiceURL(service, region, zone, endpointType string) (string, error) {
	return this.ServiceCatalog.getServiceURL(service, region, zone, endpointType)
}

func (this *TokenCredentialV2) getServiceURLs(service, region, zone, endpointType string) ([]string, error) {
	return this.ServiceCatalog.getServiceURLs(service, region, zone, endpointType)
}

func (this *TokenCredentialV2) GetInternalServices(region string) []string {
	return nil
}

func (this *TokenCredentialV2) GetExternalServices(region string) []ExternalService {
	return nil
}

func (this *TokenCredentialV2) GetServicesByInterface(region string, infType string) []ExternalService {
	return nil
}

func (this *TokenCredentialV2) GetEndpoints(region string, endpointType string) []Endpoint {
	return nil
}

func (this *TokenCredentialV2) GetServiceCatalog() IServiceCatalog {
	return this.ServiceCatalog
}

func (this *TokenCredentialV2) GetLoginSource() string {
	return this.Context.Source
}

func (this *TokenCredentialV2) GetLoginIp() string {
	return this.Context.Ip
}

func stringArrayContains(arr []string, needle string) bool {
	for i := 0; i < len(arr); i++ {
		if arr[i] == needle {
			return true
		}
	}
	return false
}

func (catalog KeystoneServiceCatalogV2) getRegions() []string {
	regions := make([]string, 0)
	for i := 0; i < len(catalog); i++ {
		for j := 0; j < len(catalog[i].Endpoints); j++ {
			r := catalog[i].Endpoints[j].Region
			slash := strings.IndexByte(r, '/')
			if slash > 0 {
				r = r[:slash]
			}
			if !stringArrayContains(regions, r) {
				regions = append(regions, r)
			}
		}
	}
	return regions
}

func (catalog KeystoneServiceCatalogV2) getServiceEndpoint(service, region, zone string) (KeystoneEndpointV2, error) {
	var selected KeystoneEndpointV2
	var findService bool
	for i := 0; i < len(catalog); i++ {
		if service == catalog[i].Type {
			findService = true
			if len(catalog[i].Endpoints) == 0 {
				continue
			}
			if len(region) == 0 {
				selected = catalog[i].Endpoints[0]
			} else {
				regionEps := make([]KeystoneEndpointV2, 0)
				zoneEps := make([]KeystoneEndpointV2, 0)
				if len(zone) > 0 {
					zone = fmt.Sprintf("%s/%s", region, zone)
				}
				for j := 0; j < len(catalog[i].Endpoints); j++ {
					if catalog[i].Endpoints[j].Region == region {
						regionEps = append(regionEps, catalog[i].Endpoints[j])
					} else if len(zone) > 0 && catalog[i].Endpoints[j].Region == zone {
						zoneEps = append(zoneEps, catalog[i].Endpoints[j])
					}
				}
				if len(zone) > 0 && len(zoneEps) > 0 {
					selected = zoneEps[rand.Intn(len(zoneEps))]
				} else if len(regionEps) > 0 {
					selected = regionEps[rand.Intn(len(regionEps))]
				} else {
					return selected, fmt.Errorf("No such region %s", region)
				}
			}
			return selected, nil
		}
	}
	if findService {
		return selected, fmt.Errorf("No default region")
	} else {
		return selected, fmt.Errorf("No such service %s", service)
	}
}

func (catalog KeystoneServiceCatalogV2) Len() int {
	return len(catalog)
}

func (catalog KeystoneServiceCatalogV2) getServiceURL(service, region, zone, endpointType string) (string, error) {
	ep, err := catalog.getServiceEndpoint(service, region, zone)
	if err != nil {
		return "", err
	}
	return ep.getURL(endpointType), nil
}

func (catalog KeystoneServiceCatalogV2) getServiceURLs(service, region, zone, endpointType string) ([]string, error) {
	url, err := catalog.getServiceURL(service, region, zone, endpointType)
	if err != nil {
		return nil, err
	}
	return []string{url}, nil
}

func (catalog KeystoneServiceCatalogV2) GetInternalServices(region string) []string {
	return nil
}

func (catalog KeystoneServiceCatalogV2) GetExternalServices(region string) []ExternalService {
	return nil
}

func (catalog KeystoneServiceCatalogV2) GetServicesByInterface(region string, infType string) []ExternalService {
	return nil
}

func (ep KeystoneEndpointV2) getURL(epType string) string {
	switch epType {
	case "publicURL":
		return ep.PublicURL
	case "adminURL":
		return ep.AdminURL
	default:
		return ep.InternalURL
	}
}

func (self *TokenCredentialV2) GetCatalogData(serviceTypes []string, region string) jsonutils.JSONObject {
	return jsonutils.Marshal(self.GetServiceCatalog())
}

func (self *TokenCredentialV2) String() string {
	token := SimplifyToken(self)
	return token.String()
}

func (self *TokenCredentialV2) IsZero() bool {
	return len(self.GetUserId()) == 0 && len(self.GetProjectId()) == 0
}

func (self *TokenCredentialV2) ToJson() jsonutils.JSONObject {
	return SimplifyToken(self).ToJson()
}
