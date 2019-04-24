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
	"yunion.io/x/pkg/utils"
)

const REGION_ZONE_SEP = '-'

type KeystoneEndpointV3 struct {
	Id        string `json:"id"`
	Interface string `json:"interface"`
	Region    string `json:"region"`
	Region_id string `json:"region_id"`
	Url       string `json:"url"`
	Name      string `json:"name"`
}

type KeystoneServiceV3 struct {
	Id        string               `json:"id"`
	Name      string               `json:"name"`
	Type      string               `json:"type"`
	Endpoints []KeystoneEndpointV3 `json:"endpoint"`
}

type KeystoneDomainV3 struct {
	Id   string
	Name string
}

type KeystoneRoleV3 struct {
	Id   string
	Name string
}

type KeystoneProjectV3 struct {
	Id     string
	Name   string
	Domain KeystoneDomainV3
}

type KeystoneUserV3 struct {
	Id                  string
	Name                string
	Domain              KeystoneDomainV3
	Password_expires_at time.Time
}

type KeystoneServiceCatalogV3 []KeystoneServiceV3

type KeystoneTokenV3 struct {
	Audit_ids  []string
	Expires_at time.Time
	Is_domain  bool
	Issued_at  time.Time
	Methods    []string
	Project    KeystoneProjectV3
	Roles      []KeystoneRoleV3
	User       KeystoneUserV3
	Catalog    KeystoneServiceCatalogV3
}

type TokenCredentialV3 struct {
	Token KeystoneTokenV3
	Id    string
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

func (this *TokenCredentialV3) GetExpires() time.Time {
	return this.Token.Expires_at
}

func (this *TokenCredentialV3) IsValid() bool {
	return this.ValidDuration() > 0
}

func (this *TokenCredentialV3) ValidDuration() time.Duration {
	return time.Until(this.Token.Expires_at)
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

func (this *TokenCredentialV3) IsAdminAllow(service string, resource string, action string, extra ...string) bool {
	return this.HasSystemAdminPrivilege()
}

func (this *TokenCredentialV3) GetRegions() []string {
	return this.Token.Catalog.getRegions()
}

func (this *TokenCredentialV3) GetServiceURL(service, region, zone, endpointType string) (string, error) {
	return this.Token.Catalog.GetServiceURL(service, region, zone, endpointType)
}

func (this *TokenCredentialV3) GetServiceURLs(service, region, zone, endpointType string) ([]string, error) {
	return this.Token.Catalog.GetServiceURLs(service, region, zone, endpointType)
}

func (this *TokenCredentialV3) GetInternalServices(region string) []string {
	return this.Token.Catalog.getInternalServices(region)
}

func (this *TokenCredentialV3) GetExternalServices(region string) []ExternalService {
	return this.Token.Catalog.getExternalServices(region)
}

func (this *TokenCredentialV3) GetEndpoints(region string, endpointType string) []Endpoint {
	return this.Token.Catalog.getEndpoints(region, endpointType)
}

func (this *TokenCredentialV3) GetServiceCatalog() IServiceCatalog {
	return this.Token.Catalog
}

func (catalog KeystoneServiceCatalogV3) getInternalServices(region string) []string {
	services := make([]string, 0)
	for i := 0; i < len(catalog); i++ {
		exit := false
		for j := 0; j < len(catalog[i].Endpoints); j++ {
			if catalog[i].Endpoints[j].Region_id == region &&
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

func (catalog KeystoneServiceCatalogV3) getExternalServices(region string) []ExternalService {
	services := make([]ExternalService, 0)
	for i := 0; i < len(catalog); i++ {
		for j := 0; j < len(catalog[i].Endpoints); j++ {
			if catalog[i].Endpoints[j].Region_id == region &&
				catalog[i].Endpoints[j].Interface == "public" &&
				len(catalog[i].Endpoints[j].Name) > 0 {
				srv := ExternalService{Name: catalog[i].Endpoints[j].Name,
					Url: catalog[i].Endpoints[j].Url}
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
			r, _ := Id2RegionZone(catalog[i].Endpoints[j].Region_id)
			if !stringArrayContains(regions, r) {
				regions = append(regions, r)
			}
		}
	}
	fmt.Println("getRegions", regions)
	return regions
}

func (catalog KeystoneServiceCatalogV3) getEndpoints(region string, endpointType string) []Endpoint {
	endpoints := make([]Endpoint, 0)
	for i := 0; i < len(catalog); i++ {
		for j := 0; j < len(catalog[i].Endpoints); j++ {
			endpoint := catalog[i].Endpoints[j]
			if (endpoint.Region_id == region || strings.HasPrefix(endpoint.Region_id, region+"-")) && endpoint.Interface == endpointType {
				endpoints = append(endpoints, Endpoint{
					endpoint.Id,
					endpoint.Region_id,
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
				if strings.HasPrefix(endpointType, ep.Interface) && (ep.Region_id == region ||
					ep.Region_id == regionzone ||
					len(region) == 0) {
					_, ok := regeps[ep.Region_id]
					if !ok {
						regeps[ep.Region_id] = make([]string, 0)
					}
					regeps[ep.Region_id] = append(regeps[ep.Region_id], ep.Url)
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
	return nil, fmt.Errorf("No such service %s", service)
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
