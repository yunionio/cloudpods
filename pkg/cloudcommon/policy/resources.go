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

package policy

import (
	"yunion.io/x/pkg/utils"
)

var (
	itsmSystemResources = []string{
		"process-definitions",
	}
	itsmDomainResources = []string{}
	itsmUserResources   = []string{}

	systemResources = map[string][]string{
		"itsm": itsmSystemResources,
	}

	domainResources = map[string][]string{
		"itsm": itsmDomainResources,
	}

	userResources = map[string][]string{
		"itsm": itsmUserResources,
	}
)

func GetSystemResources() map[string][]string {
	return systemResources
}

func GetResources() map[string]map[string][]string {
	return map[string]map[string][]string{
		"system": systemResources,
		"domain": domainResources,
		"user":   userResources,
	}
}

func isSystemResource(service string, resource string) bool {
	resList, ok := systemResources[service]
	if ok {
		if utils.IsInStringArray(resource, resList) {
			return true
		}
	}
	return false
}

func isDomainResource(service string, resource string) bool {
	resList, ok := domainResources[service]
	if ok {
		if utils.IsInStringArray(resource, resList) {
			return true
		}
	}
	return false
}

func isUserResource(service string, resource string) bool {
	resList, ok := userResources[service]
	if ok {
		if utils.IsInStringArray(resource, resList) {
			return true
		}
	}
	return false
}

func isProjectResource(service string, resource string) bool {
	if isSystemResource(service, resource) {
		return false
	}
	if isDomainResource(service, resource) {
		return false
	}
	if isUserResource(service, resource) {
		return false
	}
	return true
}

func RegisterSystemResources(service string, resources []string) {
	systemResources[service] = resources
}

func RegisterDomainResources(service string, resources []string) {
	domainResources[service] = resources
}

func RegisterUserResources(service string, resources []string) {
	userResources[service] = resources
}
