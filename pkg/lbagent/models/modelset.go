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

	"yunion.io/x/onecloud/pkg/apihelper"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type Networks map[string]*Network
type LoadbalancerNetworks map[string]*LoadbalancerNetwork // key: networkId/loadbalancerId
type Loadbalancers map[string]*Loadbalancer
type LoadbalancerListeners map[string]*LoadbalancerListener
type LoadbalancerListenerRules map[string]*LoadbalancerListenerRule
type LoadbalancerBackendGroups map[string]*LoadbalancerBackendGroup
type LoadbalancerBackends map[string]*LoadbalancerBackend
type LoadbalancerAcls map[string]*LoadbalancerAcl
type LoadbalancerCertificates map[string]*LoadbalancerCertificate

func (set Networks) ModelManager() modulebase.IBaseManager {
	return &modules.Networks
}

func (set Networks) NewModel() db.IModel {
	return &models.SNetwork{}
}

func (set Networks) AddModel(i db.IModel) {
	m, _ := i.(*models.SNetwork)
	set[m.Id] = &Network{
		SNetwork: m,
	}
}

func (set Networks) Copy() apihelper.IModelSet {
	setCopy := Networks{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (set LoadbalancerNetworks) ModelManager() modulebase.IBaseManager {
	return &modules.Loadbalancernetworks
}

func (set LoadbalancerNetworks) NewModel() db.IModel {
	return &models.SLoadbalancerNetwork{}
}

func (set LoadbalancerNetworks) AddModel(i db.IModel) {
	m, _ := i.(*models.SLoadbalancerNetwork)
	set[m.NetworkId+"/"+m.LoadbalancerId] = &LoadbalancerNetwork{
		SLoadbalancerNetwork: m,
	}
}

func (set LoadbalancerNetworks) Copy() apihelper.IModelSet {
	setCopy := LoadbalancerNetworks{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (set LoadbalancerNetworks) JoinLoadbalancers(entries Loadbalancers) bool {
	for mKey, m := range set {
		lbId := m.LoadbalancerId
		netId := m.NetworkId
		entry, ok := entries[lbId]
		if !ok {
			// this can happen for external loadbalancers
			log.Warningf("lb for loadbalancer network %s %s not found", lbId, netId)
			delete(set, mKey)
			continue
		}
		m.Loadbalancer = entry
		entry.LoadbalancerNetwork = m
	}
	for _, entry := range entries {
		if entry.LoadbalancerNetwork == nil {
			log.Warningf("loadbalancer network for loadbalancer %s(%s) not found ", entry.Name, entry.Id)
			return false
		}
	}
	return true
}

func (set LoadbalancerNetworks) JoinNetworks(entries Networks) bool {
	for mKey, m := range set {
		lbId := m.LoadbalancerId
		netId := m.NetworkId
		entry, ok := entries[netId]
		if !ok {
			// this can happen for external loadbalancers
			log.Warningf("network for loadbalancer network %s/%s not found", lbId, netId)
			delete(set, mKey)
			continue
		}
		m.Network = entry
	}
	return true
}

func (set Loadbalancers) ModelManager() modulebase.IBaseManager {
	return &modules.Loadbalancers
}

func (set Loadbalancers) NewModel() db.IModel {
	return &models.SLoadbalancer{}
}

func (set Loadbalancers) AddModel(i db.IModel) {
	m, _ := i.(*models.SLoadbalancer)
	if m.ManagerId != "" || m.ExternalId != "" {
		log.Errorf("unexpected lb: %#v", m)
		return
	}
	set[m.Id] = &Loadbalancer{
		SLoadbalancer: m,
		Listeners:     LoadbalancerListeners{},
		BackendGroups: LoadbalancerBackendGroups{},
	}
}

func (set Loadbalancers) Copy() apihelper.IModelSet {
	setCopy := Loadbalancers{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (ms Loadbalancers) JoinListeners(subEntries LoadbalancerListeners) bool {
	for _, m := range ms {
		m.Listeners = LoadbalancerListeners{}
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
		if _, ok := m.Listeners[subId]; ok {
			log.Warningf("loadbalancer listener id %s already joined", subId)
			continue
		}
		subEntry.loadbalancer = m
		m.Listeners[subId] = subEntry
	}
	return correct
}

func (ms Loadbalancers) JoinBackendGroups(subEntries LoadbalancerBackendGroups) bool {
	for _, m := range ms {
		m.BackendGroups = LoadbalancerBackendGroups{}
	}
	correct := true
	for subId, subEntry := range subEntries {
		id := subEntry.LoadbalancerId
		m, ok := ms[id]
		if !ok {
			// external_id of AWS backendgroup can be empty
			log.Warningf("backendgroup %s(%s): loadbalancer id %s not found",
				subEntry.Name, subEntry.Id, id)
			continue
		}
		if _, ok := m.BackendGroups[subId]; ok {
			log.Warningf("loadbalancer backendgroup id %s already joined", subId)
			continue
		}
		subEntry.loadbalancer = m
		m.BackendGroups[subId] = subEntry
	}
	return correct
}

func (set LoadbalancerListeners) ModelManager() modulebase.IBaseManager {
	return &modules.LoadbalancerListeners
}

func (set LoadbalancerListeners) NewModel() db.IModel {
	return &models.SLoadbalancerListener{}
}

func (set LoadbalancerListeners) AddModel(i db.IModel) {
	m, _ := i.(*models.SLoadbalancerListener)
	if m.ExternalId != "" || m.LoadbalancerId == "" {
		log.Errorf("unexpected lblistener: %#v", m)
		return
	}
	set[m.Id] = &LoadbalancerListener{
		SLoadbalancerListener: m,
		rules:                 LoadbalancerListenerRules{},
	}
}

func (set LoadbalancerListeners) Copy() apihelper.IModelSet {
	setCopy := LoadbalancerListeners{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
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

func (set LoadbalancerListenerRules) ModelManager() modulebase.IBaseManager {
	return &modules.LoadbalancerListenerRules
}

func (set LoadbalancerListenerRules) NewModel() db.IModel {
	return &models.SLoadbalancerListenerRule{}
}

func (set LoadbalancerListenerRules) AddModel(i db.IModel) {
	m, _ := i.(*models.SLoadbalancerListenerRule)
	if m.ExternalId != "" || m.ListenerId == "" {
		log.Errorf("unexpected lblistenerrule: %#v", m)
		return
	}
	set[m.Id] = &LoadbalancerListenerRule{
		SLoadbalancerListenerRule: m,
	}
}

func (set LoadbalancerListenerRules) Copy() apihelper.IModelSet {
	setCopy := LoadbalancerListenerRules{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
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

func (set LoadbalancerBackendGroups) ModelManager() modulebase.IBaseManager {
	return &modules.LoadbalancerBackendGroups
}

func (set LoadbalancerBackendGroups) NewModel() db.IModel {
	return &models.SLoadbalancerBackendGroup{}
}

func (set LoadbalancerBackendGroups) AddModel(i db.IModel) {
	m, _ := i.(*models.SLoadbalancerBackendGroup)
	if m.ExternalId != "" || m.LoadbalancerId == "" {
		log.Errorf("unexpected lbbg: %#v", m)
	}
	set[m.Id] = &LoadbalancerBackendGroup{
		SLoadbalancerBackendGroup: m,
		Backends:                  LoadbalancerBackends{},
	}
}

func (set LoadbalancerBackendGroups) Copy() apihelper.IModelSet {
	setCopy := LoadbalancerBackendGroups{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (ms LoadbalancerBackendGroups) JoinBackends(subEntries LoadbalancerBackends) bool {
	for _, m := range ms {
		m.Backends = LoadbalancerBackends{}
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
		if _, ok := m.Backends[subId]; ok {
			log.Warningf("loadbalancer backend id %s already joined", subId)
			continue
		}
		m.Backends[subId] = subEntry
		subEntry.backendGroup = m
	}
	return correct
}

func (set LoadbalancerBackends) ModelManager() modulebase.IBaseManager {
	return &modules.LoadbalancerBackends
}

func (set LoadbalancerBackends) NewModel() db.IModel {
	return &models.SLoadbalancerBackend{}
}

func (set LoadbalancerBackends) AddModel(i db.IModel) {
	m, _ := i.(*models.SLoadbalancerBackend)
	if m.ExternalId != "" || m.BackendGroupId == "" {
		log.Errorf("unexpected lbb: %#v", m)
		return
	}
	set[m.Id] = &LoadbalancerBackend{
		SLoadbalancerBackend: m,
	}
}

func (set LoadbalancerBackends) Copy() apihelper.IModelSet {
	setCopy := LoadbalancerBackends{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (set LoadbalancerAcls) ModelManager() modulebase.IBaseManager {
	return &modules.LoadbalancerAcls
}

func (set LoadbalancerAcls) NewModel() db.IModel {
	return &models.SLoadbalancerAcl{}
}

func (set LoadbalancerAcls) AddModel(i db.IModel) {
	m, _ := i.(*models.SLoadbalancerAcl)
	if m.ManagerId != "" || m.ExternalId != "" {
		return
	}
	set[m.Id] = &LoadbalancerAcl{
		SLoadbalancerAcl: m,
	}
}

func (set LoadbalancerAcls) Copy() apihelper.IModelSet {
	setCopy := LoadbalancerAcls{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (set LoadbalancerCertificates) ModelManager() modulebase.IBaseManager {
	return &modules.LoadbalancerCertificates
}

func (set LoadbalancerCertificates) NewModel() db.IModel {
	return &models.SLoadbalancerCertificate{}
}

func (set LoadbalancerCertificates) AddModel(i db.IModel) {
	m, _ := i.(*models.SLoadbalancerCertificate)
	if m.ExternalId != "" {
		return
	}
	set[m.Id] = &LoadbalancerCertificate{
		SLoadbalancerCertificate: m,
	}
}

func (set LoadbalancerCertificates) Copy() apihelper.IModelSet {
	setCopy := LoadbalancerCertificates{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}
