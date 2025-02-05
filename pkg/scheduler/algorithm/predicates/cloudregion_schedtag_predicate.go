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

type CloudregionSchedtagPredicate struct {
	*ServerBaseSchedtagPredicate
}

func NewCloudregionSchedtagPredicate() core.FitPredicate {
	p := new(CloudregionSchedtagPredicate)
	p.ServerBaseSchedtagPredicate = NewServerBaseSchedtagPredicate(p)
	return p
}

func (p *CloudregionSchedtagPredicate) Name() string {
	return "cloudregion_schedtag"
}

func (p *CloudregionSchedtagPredicate) Clone() core.FitPredicate {
	return NewCloudregionSchedtagPredicate()
}

type cloudregionSchedtagW struct {
	schedData   *api.SchedInfo
	cloudregion string
	schedtags   []*computeapi.SchedtagConfig
}

func (p *CloudregionSchedtagPredicate) GetInputs(u *core.Unit) []ISchedtagCustomer {
	data := u.SchedData()
	tags := data.Schedtags
	schedtags := GetInputSchedtagByType(tags, computemodels.CloudregionManager.KeywordPlural())
	if len(schedtags) == 0 {
		return nil
	}
	return []ISchedtagCustomer{
		&cloudregionSchedtagW{
			schedData:   data,
			cloudregion: data.PreferRegion,
			schedtags:   schedtags,
		}}
}

func (w *cloudregionSchedtagW) Keyword() string {
	return "server"
}

func (w *cloudregionSchedtagW) ResourceKeyword() string {
	return "cloudregion"
}

func (w *cloudregionSchedtagW) GetDynamicConditionInput() *jsonutils.JSONDict {
	return w.schedData.ToConditionInput()
}

func (w *cloudregionSchedtagW) IsSpecifyResource() bool {
	return w.cloudregion != ""
}

func (w *cloudregionSchedtagW) GetSchedtags() []*computeapi.SchedtagConfig {
	return w.schedtags
}

func (p *CloudregionSchedtagPredicate) GetCandidateResource(c core.Candidater) ISchedtagCandidateResource {
	zone := c.Getter().Zone()
	if zone == nil {
		return nil
	}
	region, _ := zone.GetRegion()
	if region == nil {
		return nil
	}
	return region
}
