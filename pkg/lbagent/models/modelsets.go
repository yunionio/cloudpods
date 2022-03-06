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
	"strings"
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apihelper"
)

// pluralMap maps from KeyPlurals to underscore-separated field names
var pluralMap = map[string]string{}

func init() {
	ss := []string{
		"networks",
		"loadbalancer_networks",
		"loadbalancers",
		"loadbalancer_listeners",
		"loadbalancer_listener_rules",
		"loadbalancer_backend_groups",
		"loadbalancer_backends",
		"loadbalancer_acls",
		"loadbalancer_certificates",
	}
	for _, s := range ss {
		k := strings.Replace(s, "_", "", -1)
		pluralMap[k] = s
	}
}

type ModelSetsMaxUpdatedAt struct {
	Networks                  time.Time
	LoadbalancerNetworks      time.Time
	Loadbalancers             time.Time
	LoadbalancerListeners     time.Time
	LoadbalancerListenerRules time.Time
	LoadbalancerBackendGroups time.Time
	LoadbalancerBackends      time.Time
	LoadbalancerAcls          time.Time
	LoadbalancerCertificates  time.Time
}

func NewModelSetsMaxUpdatedAt() *ModelSetsMaxUpdatedAt {
	return &ModelSetsMaxUpdatedAt{
		Networks:                  apihelper.PseudoZeroTime,
		LoadbalancerNetworks:      apihelper.PseudoZeroTime,
		Loadbalancers:             apihelper.PseudoZeroTime,
		LoadbalancerListeners:     apihelper.PseudoZeroTime,
		LoadbalancerListenerRules: apihelper.PseudoZeroTime,
		LoadbalancerBackendGroups: apihelper.PseudoZeroTime,
		LoadbalancerBackends:      apihelper.PseudoZeroTime,
		LoadbalancerAcls:          apihelper.PseudoZeroTime,
		LoadbalancerCertificates:  apihelper.PseudoZeroTime,
	}
}

type ModelSets struct {
	Networks                  Networks
	LoadbalancerNetworks      LoadbalancerNetworks
	Loadbalancers             Loadbalancers
	LoadbalancerListeners     LoadbalancerListeners
	LoadbalancerListenerRules LoadbalancerListenerRules
	LoadbalancerBackendGroups LoadbalancerBackendGroups
	LoadbalancerBackends      LoadbalancerBackends
	LoadbalancerAcls          LoadbalancerAcls
	LoadbalancerCertificates  LoadbalancerCertificates
}

func NewModelSets() *ModelSets {
	return &ModelSets{
		Networks:                  Networks{},
		LoadbalancerNetworks:      LoadbalancerNetworks{},
		Loadbalancers:             Loadbalancers{},
		LoadbalancerListeners:     LoadbalancerListeners{},
		LoadbalancerListenerRules: LoadbalancerListenerRules{},
		LoadbalancerBackendGroups: LoadbalancerBackendGroups{},
		LoadbalancerBackends:      LoadbalancerBackends{},
		LoadbalancerAcls:          LoadbalancerAcls{},
		LoadbalancerCertificates:  LoadbalancerCertificates{},
	}
}

func (mss *ModelSets) NewEmpty() apihelper.IModelSets {
	return NewModelSets()
}

func (mss *ModelSets) ModelSetList() []apihelper.IModelSet {
	// it's ordered this way to favour creation, not deletion
	return []apihelper.IModelSet{
		mss.LoadbalancerListenerRules,
		mss.LoadbalancerListeners,
		mss.LoadbalancerBackends,
		mss.LoadbalancerBackendGroups,
		mss.Loadbalancers,
		mss.LoadbalancerAcls,
		mss.LoadbalancerCertificates,
		mss.LoadbalancerNetworks,
		mss.Networks,
	}
}

func (mss *ModelSets) copy_() *ModelSets {
	mssCopy := &ModelSets{
		Networks:                  mss.Networks.Copy().(Networks),
		LoadbalancerNetworks:      mss.LoadbalancerNetworks.Copy().(LoadbalancerNetworks),
		Loadbalancers:             mss.Loadbalancers.Copy().(Loadbalancers),
		LoadbalancerListeners:     mss.LoadbalancerListeners.Copy().(LoadbalancerListeners),
		LoadbalancerListenerRules: mss.LoadbalancerListenerRules.Copy().(LoadbalancerListenerRules),
		LoadbalancerBackendGroups: mss.LoadbalancerBackendGroups.Copy().(LoadbalancerBackendGroups),
		LoadbalancerBackends:      mss.LoadbalancerBackends.Copy().(LoadbalancerBackends),
		LoadbalancerAcls:          mss.LoadbalancerAcls.Copy().(LoadbalancerAcls),
		LoadbalancerCertificates:  mss.LoadbalancerCertificates.Copy().(LoadbalancerCertificates),
	}
	return mssCopy
}

func (mss *ModelSets) Copy() apihelper.IModelSets {
	return mss.copy_()
}

func (mss *ModelSets) CopyJoined() apihelper.IModelSets {
	mssCopy := mss.copy_()
	mssCopy.join()
	return mssCopy
}

func (mss *ModelSets) MaxSeenUpdatedAtParams() *jsonutils.JSONDict {
	d := jsonutils.NewDict()
	for _, ms := range mss.ModelSetList() {
		k := ms.ModelManager().KeyString()
		k = pluralMap[k]
		t := apihelper.ModelSetMaxUpdatedAt(ms)
		if !t.Equal(apihelper.PseudoZeroTime) {
			d.Set(k, jsonutils.NewTimeString(t))
		}
	}
	return d
}

func (mss *ModelSets) ApplyUpdates(mssNews apihelper.IModelSets) apihelper.ModelSetsUpdateResult {
	r := apihelper.ModelSetsUpdateResult{
		Changed: false,
		Correct: true,
	}
	mssList := mss.ModelSetList()
	mssNewsList := mssNews.ModelSetList()
	for i, mss := range mssList {
		mssNews := mssNewsList[i]
		msR := apihelper.ModelSetApplyUpdates(mss, mssNews)
		if !r.Changed && msR.Changed {
			r.Changed = true
		}
	}
	if r.Changed {
		r.Correct = mss.join()
	}
	return r
}

func (mss *ModelSets) join() bool {
	var p []bool
	p = append(p, mss.LoadbalancerBackendGroups.JoinBackends(mss.LoadbalancerBackends))
	p = append(p, mss.LoadbalancerListeners.JoinListenerRules(mss.LoadbalancerListenerRules))
	p = append(p, mss.LoadbalancerListeners.JoinCertificates(mss.LoadbalancerCertificates))
	p = append(p, mss.Loadbalancers.JoinListeners(mss.LoadbalancerListeners))
	p = append(p, mss.Loadbalancers.JoinBackendGroups(mss.LoadbalancerBackendGroups))

	p = append(p, mss.LoadbalancerNetworks.JoinLoadbalancers(mss.Loadbalancers))
	p = append(p, mss.LoadbalancerNetworks.JoinNetworks(mss.Networks))

	for _, b := range p {
		if !b {
			return false
		}
	}
	return true
}
