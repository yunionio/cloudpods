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
	"context"

	"yunion.io/x/jsonutils"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	computemodels "yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

type CloudproviderSchedtagPredicate struct {
	*ServerBaseSchedtagPredicate
}

func NewCloudproviderSchedtagPredicate() core.FitPredicate {
	p := new(CloudproviderSchedtagPredicate)
	p.ServerBaseSchedtagPredicate = NewServerBaseSchedtagPredicate(p)
	return p
}

func (p *CloudproviderSchedtagPredicate) Name() string {
	return "cloudprovider_schedtag"
}

func (p *CloudproviderSchedtagPredicate) Clone() core.FitPredicate {
	return NewCloudproviderSchedtagPredicate()
}

func (p *CloudproviderSchedtagPredicate) PreExecute(ctx context.Context, u *core.Unit, cs []core.Candidater) (bool, error) {
	driver := u.GetHypervisorDriver()
	if driver == nil || !driver.DoScheduleCloudproviderTagFilter() {
		return false, nil
	}
	if u.SchedData().ResetCpuNumaPin {
		return false, nil
	}
	return p.ServerBaseSchedtagPredicate.PreExecute(ctx, u, cs)
}

type cloudproviderSchedtagW struct {
	schedData     *api.SchedInfo
	cloudprovider string
	schedtags     []*computeapi.SchedtagConfig
}

func (p *CloudproviderSchedtagPredicate) GetInputs(u *core.Unit) []ISchedtagCustomer {
	data := u.SchedData()
	tags := data.Schedtags
	return []ISchedtagCustomer{
		&cloudproviderSchedtagW{
			schedData:     data,
			cloudprovider: data.PreferManager,
			schedtags:     GetInputSchedtagByType(tags, computemodels.CloudproviderManager.KeywordPlural()),
		}}
}

func (w *cloudproviderSchedtagW) Keyword() string {
	return "server"
}

func (w *cloudproviderSchedtagW) ResourceKeyword() string {
	return "cloudprovider"
}

func (w *cloudproviderSchedtagW) IsSpecifyResource() bool {
	return w.cloudprovider != ""
}

func (w *cloudproviderSchedtagW) GetSchedtags() []*computeapi.SchedtagConfig {
	return w.schedtags
}

func (w *cloudproviderSchedtagW) GetDynamicConditionInput() *jsonutils.JSONDict {
	return w.schedData.ToConditionInput()
}

func (p *CloudproviderSchedtagPredicate) GetCandidateResource(c core.Candidater) ISchedtagCandidateResource {
	provider := c.Getter().Cloudprovider()
	return provider
}
