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

package consts

import (
	"time"

	"yunion.io/x/log"
)

var (
	COMMON_SERVICE = "common"

	globalRegion = ""

	globalServiceType = ""

	tenantCacheExpireSeconds = 900

	roleCacheExpireHours = 24

	nonDefaultDomainProjects = false

	defaultPagingLimit int64 = 2048
	maxPagingLimit     int64 = 2048

	domainizedNamespace = true

	historicalUniqueName = false

	enableQuotaCheck = false
)

func SetRegion(region string) {
	globalRegion = region
}

func GetRegion() string {
	return globalRegion
}

func SetServiceType(srvType string) {
	globalServiceType = srvType
}

func GetServiceType() string {
	return globalServiceType
}

func SetTenantCacheExpireSeconds(sec int) {
	tenantCacheExpireSeconds = sec
}

func GetTenantCacheExpireSeconds() time.Duration {
	return time.Duration(tenantCacheExpireSeconds) * time.Second
}

func SetRoleCacheExpireHours(h int) {
	roleCacheExpireHours = h
}

func GetRoleCacheExpireHours() time.Duration {
	return time.Duration(roleCacheExpireHours) * time.Hour
}

func SetNonDefaultDomainProjects(val bool) {
	log.Infof("set non_default_domain_projects to %v", val)
	nonDefaultDomainProjects = val
}

func GetNonDefaultDomainProjects() bool {
	return nonDefaultDomainProjects
}

func GetDefaultPagingLimit() int64 {
	return defaultPagingLimit
}

func GetMaxPagingLimit() int64 {
	return maxPagingLimit
}

func SetDomainizedNamespace(domainNS bool) {
	domainizedNamespace = domainNS
}

func IsDomainizedNamespace() bool {
	return nonDefaultDomainProjects && domainizedNamespace
}

func EnableHistoricalUniqueName() {
	historicalUniqueName = true
}

func DisableHistoricalUniqueName() {
	historicalUniqueName = false
}

func IsHistoricalUniqueName() bool {
	return historicalUniqueName
}

func SetEnableQuotaCheck(val bool) {
	enableQuotaCheck = val
}

func EnableQuotaCheck() bool {
	return enableQuotaCheck
}
