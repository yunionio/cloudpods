package models

import (
	"strings"
	"time"

	"yunion.io/x/jsonutils"
)

// pluralMap maps from KeyPlurals to underscore-separated field names
var pluralMap = map[string]string{}

func init() {
	ss := []string{
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
		Loadbalancers:             PseudoZeroTime,
		LoadbalancerListeners:     PseudoZeroTime,
		LoadbalancerListenerRules: PseudoZeroTime,
		LoadbalancerBackendGroups: PseudoZeroTime,
		LoadbalancerBackends:      PseudoZeroTime,
		LoadbalancerAcls:          PseudoZeroTime,
		LoadbalancerCertificates:  PseudoZeroTime,
	}
}

type ModelSets struct {
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
		Loadbalancers:             Loadbalancers{},
		LoadbalancerListeners:     LoadbalancerListeners{},
		LoadbalancerListenerRules: LoadbalancerListenerRules{},
		LoadbalancerBackendGroups: LoadbalancerBackendGroups{},
		LoadbalancerBackends:      LoadbalancerBackends{},
		LoadbalancerAcls:          LoadbalancerAcls{},
		LoadbalancerCertificates:  LoadbalancerCertificates{},
	}
}

func (mss *ModelSets) ModelSetList() []IModelSet {
	// it's ordered this way to favour creation, not deletion
	return []IModelSet{
		mss.LoadbalancerListenerRules,
		mss.LoadbalancerListeners,
		mss.LoadbalancerBackends,
		mss.LoadbalancerBackendGroups,
		mss.Loadbalancers,
		mss.LoadbalancerAcls,
		mss.LoadbalancerCertificates,
	}
}

func (mss *ModelSets) MaxSeenUpdatedAtParams() *jsonutils.JSONDict {
	d := jsonutils.NewDict()
	for _, ms := range mss.ModelSetList() {
		k := ms.ModelManager().KeyString()
		k = pluralMap[k]
		t := ModelSetMaxUpdatedAt(ms)
		if !t.Equal(PseudoZeroTime) {
			d.Set(k, jsonutils.NewTimeString(t))
		}
	}
	return d
}

type ModelSetsUpdateResult struct {
	Correct               bool // all elements referenced are present
	Changed               bool // any thing changed in the corpus
	ModelSetsMaxUpdatedAt *ModelSetsMaxUpdatedAt
}

func (mss *ModelSets) ApplyUpdates(mssNews *ModelSets) *ModelSetsUpdateResult {
	r := &ModelSetsUpdateResult{
		Changed: false,
		Correct: true,
	}
	mssmua := NewModelSetsMaxUpdatedAt()
	mssList := mss.ModelSetList()
	mssNewsList := mssNews.ModelSetList()
	for i, mss := range mssList {
		mssNews := mssNewsList[i]
		msR := ModelSetApplyUpdates(mss, mssNews)
		if !r.Changed && msR.Changed {
			r.Changed = true
		}
		{
			keyPlural := mss.ModelManager().KeyString()
			ModelSetsMaxUpdatedAtSetField(mssmua, keyPlural, msR.MaxUpdatedAt)
		}
	}
	if r.Changed {
		r.Correct = mss.join()
	}
	r.ModelSetsMaxUpdatedAt = mssmua
	return r
}

func (mss *ModelSets) join() bool {
	correct0 := mss.LoadbalancerBackendGroups.JoinBackends(mss.LoadbalancerBackends)
	correct1 := mss.LoadbalancerListeners.JoinListenerRules(mss.LoadbalancerListenerRules)
	correct2 := mss.LoadbalancerListeners.JoinCertificates(mss.LoadbalancerCertificates)
	correct3 := mss.Loadbalancers.JoinListeners(mss.LoadbalancerListeners)
	correct4 := mss.Loadbalancers.JoinBackendGroups(mss.LoadbalancerBackendGroups)
	return correct0 && correct1 && correct2 && correct3 && correct4
}
