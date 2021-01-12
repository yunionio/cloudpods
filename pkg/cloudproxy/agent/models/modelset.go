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
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apihelper"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	mcclient_modulebase "yunion.io/x/onecloud/pkg/mcclient/modulebase"
	mcclient_modules "yunion.io/x/onecloud/pkg/mcclient/modules/cloudproxy"
)

type (
	ProxyEndpoints map[string]*ProxyEndpoint
	Forwards       map[string]*Forward
)

func (set ProxyEndpoints) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.ProxyEndpoints
}

func (set ProxyEndpoints) NewModel() db.IModel {
	return &ProxyEndpoint{}
}

func (set ProxyEndpoints) AddModel(i db.IModel) {
	m := i.(*ProxyEndpoint)
	set[m.Id] = m
}

func (set ProxyEndpoints) Copy() apihelper.IModelSet {
	setCopy := ProxyEndpoints{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}

func (ms ProxyEndpoints) joinForwards(subEntries Forwards) bool {
	correct := true
	for _, subEntry := range subEntries {
		epId := subEntry.ProxyEndpointId
		m, ok := ms[epId]
		if !ok {
			log.Warningf("proxy_endpoint_id %s of forward %s(%s) is not present", epId, subEntry.Name, subEntry.Id)
			correct = false
			continue
		}
		subEntry.ProxyEndpoint = m
		if m.Forwards == nil {
			m.Forwards = Forwards{}
		}
		m.Forwards[subEntry.Id] = subEntry
	}
	return correct
}

func (set Forwards) ModelManager() mcclient_modulebase.IBaseManager {
	return &mcclient_modules.Forwards
}

func (set Forwards) NewModel() db.IModel {
	return &Forward{}
}

func (set Forwards) AddModel(i db.IModel) {
	m := i.(*Forward)
	set[m.Id] = m
}

func (set Forwards) Copy() apihelper.IModelSet {
	setCopy := Forwards{}
	for id, el := range set {
		setCopy[id] = el.Copy()
	}
	return setCopy
}
