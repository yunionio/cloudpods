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
)

var (
	COMMON_SERVICE = "common"

	globalRegion = ""

	globalServiceType = ""

	tenantCacheExpireSeconds = 900

	nonDefaultDomainProjects = false
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

func SetNonDefaultDomainProjects(val bool) {
	nonDefaultDomainProjects = val
}

func GetNonDefaultDomainProjects() bool {
	return nonDefaultDomainProjects
}
