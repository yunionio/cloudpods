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

package models

import (
	"sort"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"

	"yunion.io/x/onecloud/pkg/mcclient/models"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type IModelSet interface {
	//InitializeFromJSON([]jsonutils.JSONObject) error
	ModelManager() modulebase.Manager
	NewModel() models.IVirtualResource
	//GetModel(id string) models.IVirtualResource
	addModelCallback(models.IVirtualResource) error
}

type Loadbalancers map[string]*Loadbalancer
type LoadbalancerListeners map[string]*LoadbalancerListener
type LoadbalancerListenerRules map[string]*LoadbalancerListenerRule
type LoadbalancerBackendGroups map[string]*LoadbalancerBackendGroup
type LoadbalancerBackends map[string]*LoadbalancerBackend
type LoadbalancerAcls map[string]*LoadbalancerAcl
type LoadbalancerCertificates map[string]*LoadbalancerCertificate

func (set Loadbalancers) ModelManager() modulebase.Manager {
	return &modules.Loadbalancers
}

func (set Loadbalancers) NewModel() models.IVirtualResource {
	return &models.Loadbalancer{}
}

func (set Loadbalancers) addModelCallback(i models.IVirtualResource) error {
	m, _ := i.(*models.Loadbalancer)
	set[m.Id] = &Loadbalancer{
		Loadbalancer:  m,
		listeners:     LoadbalancerListeners{},
		backendGroups: LoadbalancerBackendGroups{},
	}
	return nil
}

func (ms Loadbalancers) JoinListeners(subEntries LoadbalancerListeners) bool {
	for _, m := range ms {
		m.listeners = LoadbalancerListeners{}
	}
	correct := true
	for subId, subEntry := range subEntries {
		id := subEntry.LoadbalancerId
		m, ok := ms[id]
		if !ok {
			log.Warningf("loadbalancer id %s not found", id)
			correct = false
			continue
		}
		if _, ok := m.listeners[subId]; ok {
			log.Warningf("loadbalancer listener id %s already joined", subId)
			continue
		}
		subEntry.loadbalancer = m
		m.listeners[subId] = subEntry
	}
	return correct
}

func (ms Loadbalancers) JoinBackendGroups(subEntries LoadbalancerBackendGroups) bool {
	for _, m := range ms {
		m.backendGroups = LoadbalancerBackendGroups{}
	}
	correct := true
	for subId, subEntry := range subEntries {
		id := subEntry.LoadbalancerId
		m, ok := ms[id]
		if !ok {
			log.Warningf("loadbalancer id %s not found", id)
			correct = false
			continue
		}
		if _, ok := m.backendGroups[subId]; ok {
			log.Warningf("loadbalancer backendgroup id %s already joined", subId)
			continue
		}
		subEntry.loadbalancer = m
		m.backendGroups[subId] = subEntry
	}
	return correct
}

func (set LoadbalancerListeners) ModelManager() modulebase.Manager {
	return &modules.LoadbalancerListeners
}

func (set LoadbalancerListeners) NewModel() models.IVirtualResource {
	return &models.LoadbalancerListener{}
}

func (set LoadbalancerListeners) addModelCallback(i models.IVirtualResource) error {
	m, _ := i.(*models.LoadbalancerListener)
	set[m.Id] = &LoadbalancerListener{
		LoadbalancerListener: m,
		rules:                LoadbalancerListenerRules{},
	}
	return nil
}

func (ms LoadbalancerListeners) JoinListenerRules(subEntries LoadbalancerListenerRules) bool {
	for _, m := range ms {
		m.rules = LoadbalancerListenerRules{}
	}
	correct := true
	for subId, subEntry := range subEntries {
		id := subEntry.ListenerId
		m, ok := ms[id]
		if !ok {
			log.Warningf("loadbalancer listener id %s not found", id)
			correct = false
			continue
		}
		if _, ok := m.rules[subId]; ok {
			log.Warningf("loadbalancer rule id %s already joined", subId)
			continue
		}
		subEntry.listener = m
		m.rules[subId] = subEntry
	}
	return correct
}

func (ms LoadbalancerListeners) JoinCertificates(subEntries LoadbalancerCertificates) bool {
	correct := true
	for _, m := range ms {
		m.certificate = nil
		if m.CertificateId != "" {
			subEntry, ok := subEntries[m.CertificateId]
			if !ok {
				log.Warningf("loadbalancerlistener id %s: cannot find certificate id %s",
					m.Id, m.CertificateId)
				correct = false
				continue
			}
			m.certificate = subEntry
		}
	}
	return correct
}

func (set LoadbalancerListenerRules) ModelManager() modulebase.Manager {
	return &modules.LoadbalancerListenerRules
}

func (set LoadbalancerListenerRules) NewModel() models.IVirtualResource {
	return &models.LoadbalancerListenerRule{}
}

func (set LoadbalancerListenerRules) addModelCallback(i models.IVirtualResource) error {
	m, _ := i.(*models.LoadbalancerListenerRule)
	set[m.Id] = &LoadbalancerListenerRule{
		LoadbalancerListenerRule: m,
	}
	return nil
}

type OrderedLoadbalancerListenerRuleList []*LoadbalancerListenerRule

func (lst OrderedLoadbalancerListenerRuleList) Len() int {
	return len(lst)
}

func (lst OrderedLoadbalancerListenerRuleList) Less(i, j int) bool {
	ldi := len(lst[i].Domain)
	ldj := len(lst[j].Domain)
	if ldi < ldj {
		return true
	} else if ldi == ldj {
		lpi := len(lst[i].Path)
		lpj := len(lst[j].Path)
		if lpi < lpj {
			return true
		}
	}
	return false
}

func (lst OrderedLoadbalancerListenerRuleList) Swap(i, j int) {
	lst[i], lst[j] = lst[j], lst[i]
}

func (set LoadbalancerListenerRules) OrderedEnabledList() OrderedLoadbalancerListenerRuleList {
	rules := OrderedLoadbalancerListenerRuleList{}
	for _, rule := range set {
		if rule.Status == "enabled" {
			rules = append(rules, rule)
		}
	}
	// more specific rules come first
	sort.Sort(sort.Reverse(rules))
	return rules
}

func (set LoadbalancerBackendGroups) ModelManager() modulebase.Manager {
	return &modules.LoadbalancerBackendGroups
}

func (set LoadbalancerBackendGroups) NewModel() models.IVirtualResource {
	return &models.LoadbalancerBackendGroup{}
}

func (set LoadbalancerBackendGroups) addModelCallback(i models.IVirtualResource) error {
	m, _ := i.(*models.LoadbalancerBackendGroup)
	set[m.Id] = &LoadbalancerBackendGroup{
		LoadbalancerBackendGroup: m,
		backends:                 LoadbalancerBackends{},
	}
	return nil
}

func (ms LoadbalancerBackendGroups) JoinBackends(subEntries LoadbalancerBackends) bool {
	for _, m := range ms {
		m.backends = LoadbalancerBackends{}
	}
	correct := true
	for subId, subEntry := range subEntries {
		id := subEntry.BackendGroupId
		m, ok := ms[id]
		if !ok {
			log.Warningf("loadbalancer backend group id %s not found", id)
			correct = false
			continue
		}
		if _, ok := m.backends[subId]; ok {
			log.Warningf("loadbalancer backend id %s already joined", subId)
			continue
		}
		m.backends[subId] = subEntry
	}
	return correct
}

func (set LoadbalancerBackends) ModelManager() modulebase.Manager {
	return &modules.LoadbalancerBackends
}

func (set LoadbalancerBackends) NewModel() models.IVirtualResource {
	return &models.LoadbalancerBackend{}
}

func (set LoadbalancerBackends) addModelCallback(i models.IVirtualResource) error {
	m, _ := i.(*models.LoadbalancerBackend)
	set[m.Id] = &LoadbalancerBackend{
		LoadbalancerBackend: m,
	}
	return nil
}

func (set LoadbalancerAcls) ModelManager() modulebase.Manager {
	return &modules.LoadbalancerAcls
}

func (set LoadbalancerAcls) NewModel() models.IVirtualResource {
	return &models.LoadbalancerAcl{}
}

func (set LoadbalancerAcls) addModelCallback(i models.IVirtualResource) error {
	m, _ := i.(*models.LoadbalancerAcl)
	set[m.Id] = &LoadbalancerAcl{
		LoadbalancerAcl: m,
	}
	return nil
}

func (set LoadbalancerCertificates) ModelManager() modulebase.Manager {
	return &modules.LoadbalancerCertificates
}

func (set LoadbalancerCertificates) NewModel() models.IVirtualResource {
	return &models.LoadbalancerCertificate{}
}

func (set LoadbalancerCertificates) addModelCallback(i models.IVirtualResource) error {
	m, _ := i.(*models.LoadbalancerCertificate)
	set[m.Id] = &LoadbalancerCertificate{
		LoadbalancerCertificate: m,
	}
	return nil
}
