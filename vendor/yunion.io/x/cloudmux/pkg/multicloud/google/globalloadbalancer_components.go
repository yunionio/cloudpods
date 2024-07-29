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

package google

import (
	"fmt"
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SGlobalInstanceGroup struct {
	SResourceBase
	region    *SGlobalRegion
	instances []SGlobalInstanceGroupInstance

	CreationTimestamp string      `json:"creationTimestamp"`
	Description       string      `json:"description"`
	NamedPorts        []NamedPort `json:"namedPorts"`
	Network           string      `json:"network"`
	Fingerprint       string      `json:"fingerprint"`
	Zone              string      `json:"zone"`
	Size              int64       `json:"size"`
	Region            string      `json:"region"`
	Subnetwork        string      `json:"subnetwork"`
	Kind              string      `json:"kind"`
}

type SGlobalInstanceGroupInstance struct {
	instanceGroup *SGlobalInstanceGroup
	Instance      string      `json:"instance"`
	Status        string      `json:"status"`
	NamedPorts    []NamedPort `json:"namedPorts"`
}

func (self *SGlobalLoadbalancer) isNameMatch(target interface{}) (bool, error) {
	var name string
	switch t := target.(type) {
	case *STargetHttpProxy, *STargetHttpsProxy:
		var URLMapUrl string
		switch t.(type) {
		case *STargetHttpProxy:
			URLMapUrl = t.(*STargetHttpProxy).URLMap
		case *STargetHttpsProxy:
			URLMapUrl = t.(*STargetHttpsProxy).URLMap
		}
		urlMapResponse, err := _jsonRequest(self.region.client.client, "GET", URLMapUrl, nil, false)
		if err != nil {
			return false, err
		}
		var urlMap SUrlMap
		urlMapResponse.Unmarshal(&urlMap)
		name = urlMap.Name
		if name != self.GetName() {
			parts := strings.Split(urlMap.DefaultService, "/")
			name = parts[len(parts)-1]
		}
	case *STargetTcpProxy:
		parts := strings.Split(t.Service, "/")
		name = parts[len(parts)-1]
	}
	return name == self.GetName(), nil
}

func (self *SGlobalLoadbalancer) GetForwardingRules() ([]SForwardingRule, error) {
	var ret []SForwardingRule
	if err := self.region.getGlobalLoadbalancerComponents("forwardingRules", "", &ret); err != nil {
		return nil, errors.Wrap(err, "getGlobalAddress.forwardingRules")
	}

	var _ret []SForwardingRule
	for _, item := range ret {
		targetResponse, err := _jsonRequest(self.region.client.client, "GET", item.Target, nil, false)
		if err != nil {
			return nil, errors.Wrap(err, "getGlobalAddress.GetProxy")
		}

		var target interface{}
		switch {
		case strings.Contains(item.Target, "targetHttpProxies"):
			target = new(STargetHttpProxy)
		case strings.Contains(item.Target, "targetHttpsProxies"):
			target = new(STargetHttpsProxy)
		case strings.Contains(item.Target, "targetTcpProxies"):
			target = new(STargetTcpProxy)
		default:
			continue
		}

		if err := targetResponse.Unmarshal(target); err != nil {
			return nil, errors.Wrap(err, "getGlobalAddress.target.Unmarshal")
		}

		match, err := self.isNameMatch(target)
		if err != nil {
			return nil, errors.Wrapf(err, "isNameMatch")
		}
		if match {
			_ret = append(_ret, item)
		}
	}

	return _ret, nil
}

func (self *SGlobalRegion) GetGlobalInstanceGroups(filter string) ([]SGlobalInstanceGroup, error) {
	ret := make([]SGlobalInstanceGroup, 0)
	err := self.getGlobalLoadbalancerComponents("instanceGroups", filter, &ret)
	for i := range ret {
		ret[i].region = self
	}
	return ret, err
}

func (self *SGlobalLoadbalancer) GetInstanceGroups() ([]SGlobalInstanceGroup, error) {
	if self.instanceGroups != nil {
		return self.instanceGroups, nil
	}

	if self.backendServices == nil {
		bss, err := self.GetBackendServices()
		if err != nil {
			return nil, errors.Wrap(err, "GetBackendServices")
		}
		self.backendServices = bss
	}

	bgs := []string{}
	for i := range self.backendServices {
		_bgs := self.backendServices[i].Backends
		for j := range _bgs {
			if !utils.IsInStringArray(_bgs[j].Group, bgs) {
				bgs = append(bgs, _bgs[j].Group)
			}
		}
	}

	if len(bgs) == 0 {
		return []SGlobalInstanceGroup{}, nil
	}

	regionFilters := []string{}
	zonesFilter := map[string][]string{}
	for i := range bgs {
		if !strings.Contains(bgs[i], "/zones/") {
			regionFilters = append(regionFilters, fmt.Sprintf(`(selfLink="%s")`, bgs[i]))
		} else {
			ig := bgs[i]
			index := strings.Index(ig, "/zones/")
			zoneId := strings.Split(ig[index:], "/")[2]
			if fs, ok := zonesFilter[zoneId]; ok {
				f := fmt.Sprintf(`(selfLink="%s")`, ig)
				if !utils.IsInStringArray(f, fs) {
					zonesFilter[zoneId] = append(fs, f)
				}
			} else {
				zonesFilter[zoneId] = []string{fmt.Sprintf(`(selfLink="%s")`, ig)}
			}
		}
	}

	igs := make([]SGlobalInstanceGroup, 0)
	// regional instance groups
	if len(regionFilters) > 0 {
		_igs, err := self.region.GetGlobalInstanceGroups(strings.Join(regionFilters, " OR "))
		if err != nil {
			return nil, errors.Wrap(err, "GetRegionalInstanceGroups")
		}

		igs = append(igs, _igs...)
	}

	for z, fs := range zonesFilter {
		_igs := make([]SGlobalInstanceGroup, 0)
		err := self.region.getInstanceGroups(z, "instanceGroups", strings.Join(fs, " OR "), &_igs)
		if err != nil {
			return nil, errors.Wrap(err, "getInstanceGroups")
		}

		igs = append(igs, _igs...)
	}

	self.instanceGroups = igs
	return igs, nil
}

func (self *SGlobalLoadbalancer) GetInstanceGroupsMap() (map[string]SGlobalInstanceGroup, error) {
	igs, err := self.GetInstanceGroups()
	if err != nil {
		return nil, errors.Wrap(err, "SGlobalInstanceGroup")
	}

	ret := make(map[string]SGlobalInstanceGroup, 0)
	for i := range igs {
		ig := igs[i]
		ig.region = self.region
		ret[ig.SelfLink] = ig
	}

	return ret, nil
}

func (self *SGlobalLoadbalancer) GetBackendServices() ([]SBackendServices, error) {
	if self.isHttpLb && self.urlMap != nil {
		ret := make([]SBackendServices, 0)
		ids := []string{self.urlMap.DefaultService}
		for i := range self.urlMap.PathMatchers {
			ps := self.urlMap.PathMatchers[i]
			if len(ps.DefaultService) > 0 && !utils.IsInStringArray(ps.DefaultService, ids) {
				ids = append(ids, ps.DefaultService)
			}

			for j := range ps.PathRules {
				if len(ps.PathRules[j].Service) > 0 && !utils.IsInStringArray(ps.PathRules[j].Service, ids) {
					ids = append(ids, ps.PathRules[j].Service)
				}
			}
		}

		filters := []string{}
		for i := range ids {
			filters = append(filters, fmt.Sprintf(`(selfLink="%s")`, ids[i]))
		}

		if len(filters) == 0 {
			return []SBackendServices{}, nil
		}
		err := self.region.getGlobalLoadbalancerComponents("backendServices", strings.Join(filters, " OR "), &ret)
		self.backendServices = ret
		return ret, err
	}

	return self.backendServices, nil
}

func (self *SGlobalLoadbalancer) GetTargetHttpProxies() ([]STargetHttpProxy, error) {
	ret := make([]STargetHttpProxy, 0)
	filter := fmt.Sprintf("urlMap eq %s", self.GetId())
	err := self.region.getGlobalLoadbalancerComponents("targetHttpProxies", filter, &ret)
	return ret, err
}

func (self *SGlobalLoadbalancer) GetTargetHttpsProxies() ([]STargetHttpsProxy, error) {
	ret := make([]STargetHttpsProxy, 0)
	filter := fmt.Sprintf("urlMap eq %s", self.GetId())
	err := self.region.getGlobalLoadbalancerComponents("targetHttpsProxies", filter, &ret)
	return ret, err
}

func (self *SGlobalRegion) GetGlobalUrlMaps(filter string) ([]SUrlMap, error) {
	ret := make([]SUrlMap, 0)
	err := self.getGlobalLoadbalancerComponents("urlMaps", filter, &ret)
	return ret, err
}

func (self *SGlobalRegion) GetGlobalBackendServices(filter string) ([]SBackendServices, error) {
	ret := make([]SBackendServices, 0)
	err := self.getGlobalLoadbalancerComponents("backendServices", filter, &ret)
	return ret, err
}

func (self *SGlobalInstanceGroup) GetInstances() ([]SGlobalInstanceGroupInstance, error) {
	if self.instances != nil {
		return self.instances, nil
	}

	ret := make([]SGlobalInstanceGroupInstance, 0)
	resourceId := strings.TrimPrefix(getGlobalId(self.SelfLink), fmt.Sprintf("projects/%s/", self.region.GetProjectId()))
	err := self.region.listAll("POST", resourceId+"/listInstances", nil, &ret)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return nil, nil
		}

		return nil, errors.Wrap(err, "ListAll")
	}

	for i := range ret {
		ret[i].instanceGroup = self
	}

	self.instances = ret
	return ret, nil
}

func (self *SGlobalLoadbalancer) GetHealthCheckMaps() (map[string]HealthChecks, error) {
	hcs, err := self.GetHealthChecks()
	if err != nil {
		return nil, errors.Wrap(err, "GetHealthChecks")
	}

	ret := map[string]HealthChecks{}
	for i := range hcs {
		ret[hcs[i].SelfLink] = hcs[i]
	}
	return ret, err
}

func (self *SGlobalLoadbalancer) GetHealthChecks() ([]HealthChecks, error) {
	if self.healthChecks != nil {
		return self.healthChecks, nil
	}

	hcs, err := self.region.GetRegionalHealthChecks("")
	if err != nil {
		return nil, errors.Wrap(err, "GetRegionalHealthChecks")
	}

	ghcs, err := self.region.GetGlobalHealthChecks("")
	if err != nil {
		return nil, errors.Wrap(err, "GetGlobalHealthChecks")
	}

	self.healthChecks = append(self.healthChecks, ghcs...)
	self.healthChecks = append(self.healthChecks, hcs...)
	return self.healthChecks, err
}

func (self *SGlobalRegion) GetRegionalHealthChecks(filter string) ([]HealthChecks, error) {
	ret := make([]HealthChecks, 0)
	err := self.getLoadbalancerComponents("healthChecks", filter, &ret)
	return ret, err
}

func (self *SGlobalRegion) GetGlobalHealthChecks(filter string) ([]HealthChecks, error) {
	ret := make([]HealthChecks, 0)
	params := map[string]string{}
	if len(filter) > 0 {
		params["filter"] = filter
	}

	err := self.ListAll("global/healthChecks", params, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "ListAll")
	}

	return ret, err
}

func (self *SGlobalRegion) getInstanceGroups(zoneId, resource string, filter string, result interface{}) error {
	url := fmt.Sprintf("zones/%s/%s", zoneId, resource)

	params := map[string]string{}
	if len(filter) > 0 {
		params["filter"] = filter
	}

	err := self.ListAll(url, params, result)
	if err != nil {
		return errors.Wrap(err, "ListAll")
	}

	return nil
}
