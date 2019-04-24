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

package fsdriver

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
)

func unmarshalNicConfigs(jsonNics []jsonutils.JSONObject) []types.SServerNic {
	nics := make([]types.SServerNic, 0)
	for i := range jsonNics {
		nic := types.SServerNic{}
		if err := jsonNics[i].Unmarshal(&nic); err == nil {
			nics = append(nics, nic)
		}
	}
	return nics
}

func findTeamingNic(nics []types.SServerNic, mac string) *types.SServerNic {
	for i := range nics {
		if nics[i].TeamWith == mac {
			return &nics[i]
		}
	}
	return nil
}

func convertNicConfigs(jsonNics []jsonutils.JSONObject) ([]*types.SServerNic, []*types.SServerNic) {
	nics := unmarshalNicConfigs(jsonNics)

	allNics := make([]*types.SServerNic, 0)
	bondNics := make([]*types.SServerNic, 0)

	for i := range nics {
		// skip nics without mac
		if len(nics[i].Mac) == 0 {
			continue
		}
		// skip team slave nics
		if len(nics[i].TeamWith) > 0 {
			continue
		}
		teamNic := findTeamingNic(nics, nics[i].Mac)
		if teamNic == nil {
			nnic := nics[i]
			nnic.Name = fmt.Sprintf("eth%d", nnic.Index)
			allNics = append(allNics, &nnic)
			continue
		}
		master := nics[i]
		nnic := nics[i]
		tnic := *teamNic
		nnic.Name = fmt.Sprintf("eth%d", nnic.Index)
		nnic.TeamingMaster = &master
		nnic.Ip = ""
		nnic.Gateway = ""
		tnic.Name = fmt.Sprintf("eth%d", tnic.Index)
		tnic.TeamingMaster = &master
		tnic.Ip = ""
		tnic.Gateway = ""
		master.Name = fmt.Sprintf("bond%d", len(bondNics))
		master.TeamingSlaves = []*types.SServerNic{&nnic, &tnic}
		master.Mac = ""
		allNics = append(allNics, &nnic, &tnic, &master)
		bondNics = append(bondNics, &master)
	}
	return allNics, bondNics
}
