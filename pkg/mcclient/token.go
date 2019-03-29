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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
)

type ExternalService struct {
	Name string
	Url  string
}

type Endpoint struct {
	Id          string
	RegionId    string
	ServiceId   string
	ServiceName string
	Url         string
	Interface   string
}

type IIdentityProvider interface {
	GetProjectId() string
	GetUserId() string
	GetTenantId() string
}

type TokenCredential interface {
	gotypes.ISerializable

	IServiceCatalog

	IIdentityProvider

	GetTokenString() string
	GetDomainId() string
	GetDomainName() string
	GetTenantName() string
	GetProjectName() string
	GetUserName() string
	GetRoles() []string
	GetExpires() time.Time
	IsValid() bool
	ValidDuration() time.Duration
	// IsAdmin() bool
	HasSystemAdminPrivilege() bool
	IsAdminAllow(service string, resource string, action string, extra ...string) bool
	GetRegions() []string

	GetServiceCatalog() IServiceCatalog
	GetCatalogData(serviceTypes []string, region string) jsonutils.JSONObject

	GetInternalServices(region string) []string
	GetExternalServices(region string) []ExternalService
	GetEndpoints(region string, endpointType string) []Endpoint

	ToJson() jsonutils.JSONObject
}
