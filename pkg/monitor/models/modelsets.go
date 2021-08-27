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

import "yunion.io/x/onecloud/pkg/apihelper"

type MonitorResModelSets struct {
	Servers  Servers
	Hosts    Hosts
	Rds      Rds
	Redis    Redis
	Oss      Oss
	Accounts Accounts
	Storages Storages
	Domains  Domains
	Projects Projects
}

func (m *MonitorResModelSets) NewEmpty() apihelper.IModelSets {
	return NewModelSets()
}

func (m *MonitorResModelSets) ModelSetList() []apihelper.IModelSet {
	return []apihelper.IModelSet{
		m.Servers,
		m.Hosts,
		m.Rds,
		m.Redis,
		m.Oss,
		m.Accounts,
		m.Storages,
		m.Domains,
		m.Projects,
	}
}

func (m *MonitorResModelSets) ApplyUpdates(mssNews apihelper.IModelSets) apihelper.ModelSetsUpdateResult {
	r := apihelper.ModelSetsUpdateResult{
		Changed: false,
		Correct: true,
	}
	mssList := m.ModelSetList()
	mssNewsList := mssNews.ModelSetList()
	for i, mss := range mssList {
		mssNews := mssNewsList[i]
		msR := apihelper.ModelSetApplyUpdates(mss, mssNews)
		if !r.Changed && msR.Changed {
			r.Changed = true
		}
	}
	return r
}

func (m *MonitorResModelSets) Copy() apihelper.IModelSets {
	//TODO CHECKE
	return m
}

func (m *MonitorResModelSets) CopyJoined() apihelper.IModelSets {
	//TODO CHECKE
	return m
}

func NewModelSets() *MonitorResModelSets {
	return &MonitorResModelSets{
		Servers:  Servers{},
		Hosts:    Hosts{},
		Rds:      Rds{},
		Redis:    Redis{},
		Oss:      Oss{},
		Accounts: Accounts{},
		Storages: Storages{},
		Domains:  Domains{},
		Projects: Projects{},
	}
}
