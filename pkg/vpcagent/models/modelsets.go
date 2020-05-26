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
	"time"

	"yunion.io/x/onecloud/pkg/vpcagent/apihelper"
)

type ModelSetsMaxUpdatedAt struct {
	Vpcs          time.Time
	Networks      time.Time
	Guests        time.Time
	Hosts         time.Time
	Guestnetworks time.Time
}

func NewModelSetsMaxUpdatedAt() *ModelSetsMaxUpdatedAt {
	return &ModelSetsMaxUpdatedAt{
		Vpcs:          apihelper.PseudoZeroTime,
		Networks:      apihelper.PseudoZeroTime,
		Guests:        apihelper.PseudoZeroTime,
		Hosts:         apihelper.PseudoZeroTime,
		Guestnetworks: apihelper.PseudoZeroTime,
	}
}

type ModelSets struct {
	Vpcs          Vpcs
	Networks      Networks
	Guests        Guests
	Hosts         Hosts
	Guestnetworks Guestnetworks
}

func NewModelSets() *ModelSets {
	return &ModelSets{
		Vpcs:          Vpcs{},
		Networks:      Networks{},
		Guests:        Guests{},
		Hosts:         Hosts{},
		Guestnetworks: Guestnetworks{},
	}
}

func (mss *ModelSets) ModelSetList() []apihelper.IModelSet {
	// it's ordered this way to favour creation, not deletion
	return []apihelper.IModelSet{
		mss.Vpcs,
		mss.Networks,
		mss.Guests,
		mss.Hosts,
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
		Guests:        mss.Guests.Copy().(Guests),
		Hosts:         mss.Hosts.Copy().(Hosts),
		Guestnetworks: mss.Guestnetworks.Copy().(Guestnetworks),
	}
	mssCopy.join()
	return mssCopy
}

func (mss *ModelSets) CopyJoined() apihelper.IModelSets {
	return mss.Copy()
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
	p = append(p, mss.Vpcs.joinNetworks(mss.Networks))
	p = append(p, mss.Networks.joinGuestnetworks(mss.Guestnetworks))
	p = append(p, mss.Guests.joinHosts(mss.Hosts))
	p = append(p, mss.Guestnetworks.joinGuests(mss.Guests))
	for _, b := range p {
		if !b {
			return false
		}
	}
	return true
}
