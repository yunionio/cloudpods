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

	"yunion.io/x/onecloud/pkg/apihelper"
)

type ModelSetsMaxUpdatedAt struct {
	ProxyEndpoints time.Time
	Forwards       time.Time
}

func NewModelSetsMaxUpdatedAt() *ModelSetsMaxUpdatedAt {
	return &ModelSetsMaxUpdatedAt{
		ProxyEndpoints: apihelper.PseudoZeroTime,
		Forwards:       apihelper.PseudoZeroTime,
	}
}

type ModelSets struct {
	ProxyEndpoints ProxyEndpoints
	Forwards       Forwards
}

func NewModelSets() *ModelSets {
	return &ModelSets{
		ProxyEndpoints: ProxyEndpoints{},
		Forwards:       Forwards{},
	}
}

func (mss *ModelSets) ModelSetList() []apihelper.IModelSet {
	// it's ordered this way to favour creation, not deletion
	return []apihelper.IModelSet{
		mss.ProxyEndpoints,
		mss.Forwards,
	}
}

func (mss *ModelSets) NewEmpty() apihelper.IModelSets {
	return NewModelSets()
}

func (mss *ModelSets) copy_() *ModelSets {
	mssCopy := &ModelSets{
		ProxyEndpoints: mss.ProxyEndpoints.Copy().(ProxyEndpoints),
		Forwards:       mss.Forwards.Copy().(Forwards),
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
	p = append(p, mss.ProxyEndpoints.joinForwards(mss.Forwards))
	for _, b := range p {
		if !b {
			return false
		}
	}
	return true
}
