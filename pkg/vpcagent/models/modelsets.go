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

	"yunion.io/x/onecloud/pkg/vpcagent/apihelper"
)

// pluralMap maps from KeyPlurals to underscore-separated field names
var pluralMap = map[string]string{}

func init() {
	// XXX drop this
	ss := []string{
		"vpcs",
		"networks",
		"guestnetworks",
	}
	for _, s := range ss {
		k := strings.Replace(s, "_", "", -1)
		pluralMap[k] = s
	}
}

type ModelSetsMaxUpdatedAt struct {
	Vpcs          time.Time
	Networks      time.Time
	Guestnetworks time.Time
}

func NewModelSetsMaxUpdatedAt() *ModelSetsMaxUpdatedAt {
	return &ModelSetsMaxUpdatedAt{
		Vpcs:          apihelper.PseudoZeroTime,
		Networks:      apihelper.PseudoZeroTime,
		Guestnetworks: apihelper.PseudoZeroTime,
	}
}

type ModelSets struct {
	Vpcs          Vpcs
	Networks      Networks
	Guestnetworks Guestnetworks
}

func NewModelSets() *ModelSets {
	return &ModelSets{
		Vpcs:          Vpcs{},
		Networks:      Networks{},
		Guestnetworks: Guestnetworks{},
	}
}

func (mss *ModelSets) ModelSetList() []apihelper.IModelSet {
	// it's ordered this way to favour creation, not deletion
	return []apihelper.IModelSet{
		mss.Vpcs,
		mss.Networks,
		mss.Guestnetworks,
	}
}

func (mss *ModelSets) NewEmpty() apihelper.IModelSets {
	return NewModelSets()
}

func (mss *ModelSets) Copy() apihelper.IModelSets {
	mssCopy := &ModelSets{
		Vpcs:          mss.Vpcs.Copy().(Vpcs),
		Networks:      mss.Networks.Copy().(Networks),
		Guestnetworks: mss.Guestnetworks.Copy().(Guestnetworks),
	}
	mssCopy.join()
	return mssCopy
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
	correct0 := mss.Vpcs.joinNetworks(mss.Networks)
	correct1 := mss.Networks.joinGuestnetworks(mss.Guestnetworks)
	return correct0 && correct1
}
