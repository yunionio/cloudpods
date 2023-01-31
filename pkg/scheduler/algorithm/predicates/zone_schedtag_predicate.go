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

type ZoneSchedtagPredicate struct {
	*ServerBaseSchedtagPredicate
}

func NewZoneSchedtagPredicate() core.FitPredicate {
	p := new(ZoneSchedtagPredicate)
	p.ServerBaseSchedtagPredicate = NewServerBaseSchedtagPredicate(p)
	return p
}

func (p *ZoneSchedtagPredicate) Name() string {
	return "zone_schedtag"
}

func (p *ZoneSchedtagPredicate) Clone() core.FitPredicate {
	return NewZoneSchedtagPredicate()
}

type zoneSchedtagInputW struct {
	schedData *api.SchedInfo
	zone      string
	schedtags []*computeapi.SchedtagConfig
}

func (p *ZoneSchedtagPredicate) GetInputs(u *core.Unit) []ISchedtagCustomer {
	data := u.SchedData()
	tags := data.Schedtags
	return []ISchedtagCustomer{
		&zoneSchedtagInputW{
			schedData: data,
			zone:      data.PreferZone,
			schedtags: GetInputSchedtagByType(tags, computemodels.ZoneManager.KeywordPlural()),
		},
	}
}

func (w *zoneSchedtagInputW) Keyword() string {
	return "server"
}

func (w *zoneSchedtagInputW) ResourceKeyword() string {
	return "zone"
}

func (w *zoneSchedtagInputW) GetDynamicConditionInput() *jsonutils.JSONDict {
	return w.schedData.ToConditionInput()
}

func (w *zoneSchedtagInputW) IsSpecifyResource() bool {
	return w.zone != ""
}

func (w *zoneSchedtagInputW) GetSchedtags() []*computeapi.SchedtagConfig {
	return w.schedtags
}

type zoneSchedtagResW struct {
	zone *computemodels.SZone
	c    core.Candidater
}

func (r zoneSchedtagResW) GetName() string {
	return r.zone.GetName()
}

func (r zoneSchedtagResW) GetId() string {
	return r.zone.GetId()
}

func (r zoneSchedtagResW) Keyword() string {
	return r.zone.Keyword()
}

func (r zoneSchedtagResW) GetSchedtagJointManager() computemodels.ISchedtagJointManager {
	return r.zone.GetSchedtagJointManager()
}

func (r zoneSchedtagResW) GetDynamicConditionInput() *jsonutils.JSONDict {
	// TODO: use sched input data ???
	return r.zone.GetDynamicConditionInput()
}

func (p *ZoneSchedtagPredicate) GetCandidateResource(c core.Candidater) ISchedtagCandidateResource {
	zone := c.Getter().Zone()
	return zoneSchedtagResW{
		zone: zone,
		c:    c,
	}
}
