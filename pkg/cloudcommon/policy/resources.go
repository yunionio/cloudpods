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

import "yunion.io/x/pkg/utils"

var (
	computeSystemResources = []string{
		"hosts",
		"zones",
		"storages",
		"wires",
		"vpcs",
		"route_tables",
		"cloudregions",
		"schedtags",
		"serverskus",
		"cachedimages",
		"dynamicschedtags",
		"baremetalagents",
		"schedpolicies",
		"isolated-devices",
		"reservedips",
		"dnsrecords",
		"metadatas",
	}
	computeDomainResources = []string{
		"cloudaccounts",
		"recyclebins",
	}
	computeUserResources = []string{
		"keypairs",
	}

	notifySystemResources = []string{
		"configs",
		"contacts",
	}
	notifyDomainResources = []string{}
	notifyUserResources   = []string{}

	meterSystemResources = []string{
		"rates",
		"res_results",
	}
	meterDomainResources = []string{}
	meterUserResources   = []string{}

	k8sSystemResources = []string{
		"kube_clusters",
		"kube_nodes",
	}
	k8sDomainResources = []string{}
	k8sUserResources   = []string{}

	yunionagentSystemResources = []string{
		"notices",
		"readmarks",
		"infos",
	}
	yunionagentDomainResources = []string{}
	yunionagentUserResources   = []string{}

	yunionconfSystemResources = []string{}
	yunionconfDomainResources = []string{}
	yunionconfUserResources   = []string{
		"parameters",
	}

	logSystemResources = []string{}
	logDomainResources = []string{}
	logUserResources   = []string{}

	identitySystemResources = []string{
		"identity_providers",
		"domains",
		"services",
		"endpoints",
	}
	identityDomainResources = []string{
		"users",
		"groups",
		"projects",
		"roles",
		"policies",
	}
	identityUserResources = []string{}

	itsmSystemResources = []string{
		"process-definitions",
	}
	itsmDomainResources = []string{}
	itsmUserResources   = []string{}

	systemResources = map[string][]string{
		"compute":     computeSystemResources,
		"notify":      notifySystemResources,
		"meter":       meterSystemResources,
		"k8s":         k8sSystemResources,
		"yunionagent": yunionagentSystemResources,
		"yunionconf":  yunionconfSystemResources,
		"log":         logSystemResources,
		"identity":    identitySystemResources,
		"itsm":        itsmSystemResources,
	}

	domainResources = map[string][]string{
		"compute":     computeDomainResources,
		"notify":      notifyDomainResources,
		"meter":       meterDomainResources,
		"k8s":         k8sDomainResources,
		"yunionagent": yunionagentDomainResources,
		"yunionconf":  yunionconfDomainResources,
		"log":         logDomainResources,
		"identity":    identityDomainResources,
		"itsm":        itsmDomainResources,
	}

	userResources = map[string][]string{
		"compute":     computeUserResources,
		"notify":      notifyUserResources,
		"meter":       meterUserResources,
		"k8s":         k8sUserResources,
		"yunionagent": yunionagentUserResources,
		"yunionconf":  yunionconfUserResources,
		"log":         logUserResources,
		"identity":    identityUserResources,
		"itsm":        itsmUserResources,
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
