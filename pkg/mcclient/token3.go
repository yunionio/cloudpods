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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

const REGION_ZONE_SEP = '-'

type KeystoneEndpointV3 struct {
	// endpoint ID
	// example: 75f4e36100184a5a8a3e36cb0f12aa87
	Id string `json:"id"`
	// endpoint接口类型，目前定义了一下集中类型
	//
	// | interface | 说明                                                   |
	// |-----------|--------------------------------------------------------|
	// | internal  | 内部接口，访问服务时默认用inernal类型的接口            |
	// | public    | 外部接口                                               |
	// | admin     | 管理类型接口，deprecated                               |
	// | console   | web控制台接口，指定显示在web控制台的外部服务的接口地址 |
	//
	Interface string `json:"interface"`
	// 区域名称
	Region string `json:"region"`
	// 区域ID
	RegionId string `json:"region_id"`
	// 接口URL
	Url string `json:"url"`
	// 接口名称
	Name string `json:"name"`
}

type KeystoneServiceV3 struct {
	// service ID
	Id string `json:"id,omitempty"`
	// service Name
	Name string `json:"name,omitempty"`
	// service Type，例如identity, compute等
	Type string `json:"type,omitempty"`
	// service的访问endpoints
	Endpoints []KeystoneEndpointV3 `json:"endpoints,omitempty"`
}

type KeystoneDomainV3 api.SIdentityObject

type KeystoneRoleV3 api.SIdentityObject

type KeystoneProjectV3 struct {
	// 项目ID
	Id string
	// 项目名称
	Name string
	// 项目归属域
	Domain KeystoneDomainV3
}

type KeystoneUserV3 struct {
	// 用户ID
	Id string
	// 用户名称
	Name string
	// 用户归属域
	Domain KeystoneDomainV3
	// 用户密码过期时间
	PasswordExpiresAt time.Time

	// 用户的显式名称，通常为中文名
	Displayname string
	// 用户Email
	Email string
	// 用户手机号
	Mobile string
}

type KeystoneServiceCatalogV3 []KeystoneServiceV3

type KeystonePolicy struct {
	// 项目范围的权限
	Project []string
	// 域范围的权限
	Domain []string
	// 系统范围的权限
	System []string
}

type KeystoneTokenV3 struct {
	// AutdiIds, 没有什么用
	// swagger:ignore
	AuditIds []string `json:"audit_ids"`
	// token过期时间
	ExpiresAt time.Time `json:"expires_at"`
	// 是否为域的token
	IsDomain bool `json:"is_domain,allowfalse"`
	// token颁发时间
	IssuedAt time.Time `json:"issued_at"`
	// 获取token的认证方式
	Methods []string `json:"methods"`
	// token的关联项目，如果用户认证时scope为项目，则为改指定项目的信息
	Project KeystoneProjectV3 `json:"project"`
	// token的关联用户在关联项目的权限信息，只有项目scope的token才有这个属性
	Policies KeystonePolicy `json:"policies"`
	// token的关联用户在关联项目的角色列表，只有项目scope的token才有这个属性
	Roles []KeystoneRoleV3 `json:"roles"`
	// token的关联用户信息
	User KeystoneUserV3 `json:"user"`
	// 服务目录
	Catalog KeystoneServiceCatalogV3 `json:"catalog"`
	// 认证上下文
	Context SAuthContext `json:"context"`

	// 当用户认证时未指定scope时，会返回该用户所有的项目
	Projects []KeystoneProjectV3 `json:"projects"`
	// 返回用户在所有项目的所有角色信息
	RoleAssignments []api.SRoleAssignment `json:"role_assignments"`

	// 如果时AK/SK认证，返回用户的AccessKey/Secret信息，用于客户端后续的AK/SK认证，避免频繁访问keystone进行AK/SK认证
	AccessKey api.SAccessKeySecretInfo `json:"access_key"`
}

type TokenCredentialV3 struct {
	// keystone V3 token
	Token KeystoneTokenV3 `json:"token"`

	// swagger:ignore
	Id string `json:"id"`
}

func (token *TokenCredentialV3) GetTokenString() string {
	return token.Id
}

func (token *TokenCredentialV3) GetDomainId() string {
	return token.Token.User.Domain.Id
}

func (token *TokenCredentialV3) GetDomainName() string {
	return token.Token.User.Domain.Name
}

func (token *TokenCredentialV3) GetProjectId() string {
	return token.Token.Project.Id
}

func (token *TokenCredentialV3) GetProjectName() string {
	return token.Token.Project.Name
}

func (token *TokenCredentialV3) GetTenantId() string {
	return token.Token.Project.Id
}

func (token *TokenCredentialV3) GetTenantName() string {
	return token.Token.Project.Name
}

func (token *TokenCredentialV3) GetProjectDomainId() string {
	return token.Token.Project.Domain.Id
}

func (token *TokenCredentialV3) GetProjectDomain() string {
	return token.Token.Project.Domain.Name
}

func (token *TokenCredentialV3) GetUserName() string {
	return token.Token.User.Name
}

func (token *TokenCredentialV3) GetUserId() string {
	return token.Token.User.Id
}

func (token *TokenCredentialV3) GetRoles() []string {
	roles := make([]string, 0)
	for i := 0; i < len(token.Token.Roles); i++ {
		roles = append(roles, token.Token.Roles[i].Name)
	}
	return roles
}

func (token *TokenCredentialV3) GetRoleIds() []string {
	roles := make([]string, 0)
	for i := 0; i < len(token.Token.Roles); i++ {
		roles = append(roles, token.Token.Roles[i].Id)
	}
	return roles
}

func (this *TokenCredentialV3) GetExpires() time.Time {
	return this.Token.ExpiresAt
}

func (this *TokenCredentialV3) IsValid() bool {
	return len(this.Id) > 0 && this.ValidDuration() > 0
}

func (this *TokenCredentialV3) ValidDuration() time.Duration {
	return time.Until(this.Token.ExpiresAt)
}

func (this *TokenCredentialV3) IsAdmin() bool {
	for i := 0; i < len(this.Token.Roles); i++ {
		if this.Token.Roles[i].Name == "admin" {
			return true
		}
	}
	return false
}

func (this *TokenCredentialV3) HasSystemAdminPrivilege() bool {
	return this.IsAdmin() && this.GetTenantName() == "system"
}

func (this *TokenCredentialV3) IsAllow(scope rbacutils.TRbacScope, service string, resource string, action string, extra ...string) bool {
	if scope == rbacutils.ScopeSystem || scope == rbacutils.ScopeDomain {
		return this.HasSystemAdminPrivilege()
	} else {
		return true
	}
}

func (this *TokenCredentialV3) GetRegions() []string {
	return this.Token.Catalog.getRegions()
}

func (this *TokenCredentialV3) Len() int {
	return this.Token.Catalog.Len()
}

func (this *TokenCredentialV3) GetServiceURL(service, region, zone, endpointType string) (string, error) {
	return this.Token.Catalog.GetServiceURL(service, region, zone, endpointType)
}

func (this *TokenCredentialV3) GetServiceURLs(service, region, zone, endpointType string) ([]string, error) {
	return this.Token.Catalog.GetServiceURLs(service, region, zone, endpointType)
}

func (this *TokenCredentialV3) GetInternalServices(region string) []string {
	return this.Token.Catalog.GetInternalServices(region)
}

func (this *TokenCredentialV3) GetExternalServices(region string) []ExternalService {
	return this.Token.Catalog.GetExternalServices(region)
}

func (this *TokenCredentialV3) GetServicesByInterface(region string, infType string) []ExternalService {
	return this.Token.Catalog.GetServicesByInterface(region, infType)
}

func (this *TokenCredentialV3) GetEndpoints(region string, endpointType string) []Endpoint {
	return this.Token.Catalog.getEndpoints(region, endpointType)
}

func (this *TokenCredentialV3) GetServiceCatalog() IServiceCatalog {
	return this.Token.Catalog
}

func (this *TokenCredentialV3) GetLoginSource() string {
	return this.Token.Context.Source
}

func (this *TokenCredentialV3) GetLoginIp() string {
	return this.Token.Context.Ip
}

func (catalog KeystoneServiceCatalogV3) GetInternalServices(region string) []string {
	services := make([]string, 0)
	for i := 0; i < len(catalog); i++ {
		exit := false
		for j := 0; j < len(catalog[i].Endpoints); j++ {
			if catalog[i].Endpoints[j].RegionId == region &&
				catalog[i].Endpoints[j].Interface == "internal" {
				exit = true
				break
			}
		}
		if exit {
			services = append(services, catalog[i].Type)
		}
	}
	return services
}

func (catalog KeystoneServiceCatalogV3) GetExternalServices(region string) []ExternalService {
	return catalog.GetServicesByInterface(region, "console")
}

func (catalog KeystoneServiceCatalogV3) GetServicesByInterface(region string, infType string) []ExternalService {
	services := make([]ExternalService, 0)
	for i := 0; i < len(catalog); i++ {
		for j := 0; j < len(catalog[i].Endpoints); j++ {
			if catalog[i].Endpoints[j].RegionId == region &&
				catalog[i].Endpoints[j].Interface == infType &&
				len(catalog[i].Endpoints[j].Name) > 0 {
				srv := ExternalService{
					Name:    catalog[i].Endpoints[j].Name,
					Url:     catalog[i].Endpoints[j].Url,
					Service: catalog[i].Type,
				}
				services = append(services, srv)
			}
		}
	}
	return services
}

func (catalog KeystoneServiceCatalogV3) getRegions() []string {
	regions := make([]string, 0)
	for i := 0; i < len(catalog); i++ {
		for j := 0; j < len(catalog[i].Endpoints); j++ {
			if len(catalog[i].Endpoints[j].RegionId) > 0 && !stringArrayContains(regions, catalog[i].Endpoints[j].RegionId) {
				regions = append(regions, catalog[i].Endpoints[j].RegionId)
			}
		}
	}
	return regions
}

func (catalog KeystoneServiceCatalogV3) getEndpoints(region string, endpointType string) []Endpoint {
	endpoints := make([]Endpoint, 0)
	for i := 0; i < len(catalog); i++ {
		for j := 0; j < len(catalog[i].Endpoints); j++ {
			endpoint := catalog[i].Endpoints[j]
			if (endpoint.RegionId == region || strings.HasPrefix(endpoint.RegionId, region+"-")) && endpoint.Interface == endpointType {
				endpoints = append(endpoints, Endpoint{
					endpoint.Id,
					endpoint.RegionId,
					catalog[i].Id,
					catalog[i].Name,
					endpoint.Url,
					endpoint.Interface,
				})
			}
		}
	}

	return endpoints
}

func RegionID(region, zone string) string {
	if len(zone) > 0 {
		return fmt.Sprintf("%s%c%s", region, REGION_ZONE_SEP, zone)
	} else {
		return region
	}
}

func Id2RegionZone(id string) (string, string) {
	pos := strings.IndexByte(id, REGION_ZONE_SEP)
	if pos > 0 {
		return id[:pos], id[pos+1:]
	} else {
		return id, ""
	}
}

func (catalog KeystoneServiceCatalogV3) Len() int {
	return len(catalog)
}

func (catalog KeystoneServiceCatalogV3) GetServiceURL(service, region, zone, endpointType string) (string, error) {
	urls, err := catalog.GetServiceURLs(service, region, zone, endpointType)
	if err != nil {
		return "", err
	}
	return urls[rand.Intn(len(urls))], nil
}

func (catalog KeystoneServiceCatalogV3) GetServiceURLs(service, region, zone, endpointType string) ([]string, error) {
	if endpointType == "" {
		endpointType = "internalURL"
	}
	for i := 0; i < len(catalog); i++ {
		if service == catalog[i].Type {
			if len(catalog[i].Endpoints) == 0 {
				continue
			}
			var selected []string
			regeps := make(map[string][]string)
			regionzone := ""
			if len(zone) > 0 {
				regionzone = RegionID(region, zone)
			}
			for j := 0; j < len(catalog[i].Endpoints); j++ {
				ep := catalog[i].Endpoints[j]
				if strings.HasPrefix(endpointType, ep.Interface) && (ep.RegionId == region ||
					ep.RegionId == regionzone ||
					len(region) == 0) {
					_, ok := regeps[ep.RegionId]
					if !ok {
						regeps[ep.RegionId] = make([]string, 0)
					}
					regeps[ep.RegionId] = append(regeps[ep.RegionId], ep.Url)
				}
			}
			if len(region) == 0 {
				if len(regeps) >= 1 {
					for _, v := range regeps {
						selected = v
						break
					}
				} else {
					return nil, fmt.Errorf("No default region")
				}
			} else {
				_, ok := regeps[regionzone]
				if ok {
					selected = regeps[regionzone]
				} else {
					selected, ok = regeps[region]
					if !ok {
						return nil, fmt.Errorf("No valid %s endpoints for %s in region %s", endpointType, service, RegionID(region, zone))
					}
				}
			}
			return selected, nil
		}
	}
	return nil, errors.Wrapf(httperrors.ErrNotFound, "No such service %s", service)
}

func (self *TokenCredentialV3) GetCatalogData(serviceTypes []string, region string) jsonutils.JSONObject {
	catalog := self.Token.Catalog
	ret := make([]map[string]interface{}, 0)
	for i := 0; i < len(catalog); i++ {
		if !utils.IsInStringArray(catalog[i].Type, serviceTypes) {
			continue
		}
		neps := make([]KeystoneEndpointV3, 0)
		for j := 0; j < len(catalog[i].Endpoints); j++ {
			if catalog[i].Endpoints[j].Region != region {
				continue
			}
			neps = append(neps, catalog[i].Endpoints[j])
		}
		if len(neps) > 0 {
			data := map[string]interface{}{
				"type":      catalog[i].Type,
				"endpoints": neps,
			}
			ret = append(ret, data)
		}
	}
	return jsonutils.Marshal(ret)
}

func (self *TokenCredentialV3) String() string {
	token := SimplifyToken(self)
	return token.String()
}

func (self *TokenCredentialV3) IsZero() bool {
	return len(self.GetUserId()) == 0 && len(self.GetProjectId()) == 0
}

func (self *TokenCredentialV3) ToJson() jsonutils.JSONObject {
	return SimplifyToken(self).ToJson()
}
