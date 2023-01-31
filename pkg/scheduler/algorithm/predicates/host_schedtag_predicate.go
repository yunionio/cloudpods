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

package predicates

import (
	"yunion.io/x/jsonutils"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	computemodels "yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

type HostSchedtagPredicate struct {
	*ServerBaseSchedtagPredicate
}

func NewHostSchedtagPredicate() core.FitPredicate {
	p := new(HostSchedtagPredicate)
	p.ServerBaseSchedtagPredicate = NewServerBaseSchedtagPredicate(p)
	return p
}

func (p *HostSchedtagPredicate) Name() string {
	return "host_schedtag"
}

func (p *HostSchedtagPredicate) Clone() core.FitPredicate {
	return NewHostSchedtagPredicate()
}

type hostSchedtagInputW struct {
	schedData  *api.SchedInfo
	host       string
	backupHost string
	schedtags  []*computeapi.SchedtagConfig
}

func (p *HostSchedtagPredicate) GetInputs(u *core.Unit) []ISchedtagCustomer {
	data := u.SchedData()
	tags := data.Schedtags
	return []ISchedtagCustomer{
		&hostSchedtagInputW{
			schedData:  data,
			host:       data.PreferHost,
			backupHost: data.PreferBackupHost,
			schedtags:  GetInputSchedtagByType(tags, "", computemodels.HostManager.KeywordPlural()),
		}}
}

func (w *hostSchedtagInputW) Keyword() string {
	return "server"
}

func (w *hostSchedtagInputW) ResourceKeyword() string {
	return "host"
}

func (w *hostSchedtagInputW) GetDynamicConditionInput() *jsonutils.JSONDict {
	return w.schedData.ToConditionInput()
}

func (w *hostSchedtagInputW) IsSpecifyResource() bool {
	return w.host != "" || w.backupHost != ""
}

func (w *hostSchedtagInputW) GetSchedtags() []*computeapi.SchedtagConfig {
	return w.schedtags
}

type hostSchedtagResW struct {
	core.Candidater
}

func (r hostSchedtagResW) GetName() string {
	return r.Candidater.Getter().Name()
}

func (r hostSchedtagResW) GetId() string {
	return r.Candidater.Getter().Id()
}

func (r hostSchedtagResW) Keyword() string {
	return r.Candidater.Getter().Host().Keyword()
}

func (r hostSchedtagResW) GetSchedtagJointManager() computemodels.ISchedtagJointManager {
	return r.Candidater.Getter().Host().GetSchedtagJointManager()
}

func (r hostSchedtagResW) GetDynamicConditionInput() *jsonutils.JSONDict {
	return r.GetSchedDesc()
}

func (p *HostSchedtagPredicate) GetCandidateResource(c core.Candidater) ISchedtagCandidateResource {
	return hostSchedtagResW{c}
}
