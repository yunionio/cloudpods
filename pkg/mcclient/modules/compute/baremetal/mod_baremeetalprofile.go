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

package baremetal

import (
	"sort"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	baremetalapi "yunion.io/x/onecloud/pkg/apis/compute/baremetal"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	baremetaloptions "yunion.io/x/onecloud/pkg/mcclient/options/compute/baremetal"
)

type BaremetalProfileManager struct {
	modulebase.ResourceManager
}

func (m *BaremetalProfileManager) GetMatchProfiles(s *mcclient.ClientSession, oemName string, model string) ([]baremetalapi.BaremetalProfileSpec, error) {
	params := baremetaloptions.BaremetalProfileListOptions{}
	limit := 0
	params.Limit = &limit
	params.OemName = []string{"", oemName}
	if len(model) > 0 {
		params.Model = []string{"", model}
	}

	results, err := m.List(s, jsonutils.Marshal(params))
	if err != nil {
		return nil, errors.Wrap(err, "List")
	}
	ret := make([]baremetalapi.BaremetalProfileSpec, 0)
	for i := range results.Data {
		details := baremetalapi.BaremetalProfileDetails{}
		err := results.Data[i].Unmarshal(&details)
		if err != nil {
			return nil, errors.Wrap(err, "Unmarshal")
		}
		ret = append(ret, details.ToSpec())
	}
	sort.Sort(baremetalapi.BaremetalProfileSpecs(ret))
	return ret, nil
}

var (
	BaremetalProfiles BaremetalProfileManager
)

func init() {
	BaremetalProfiles = BaremetalProfileManager{modules.NewComputeManager("baremetal_profile", "baremetal_profiles",
		[]string{
			"oem_name",
			"model",
			"lan_channel",
			"root_id",
			"root_name",
			"string_pass",
		},
		[]string{},
	)}

	modules.RegisterCompute(&BaremetalProfiles)
}
