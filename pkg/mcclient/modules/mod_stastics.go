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

package modules

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type StatisticsManager struct {
	modulebase.ResourceManager
}

var (
	Statistics StatisticsManager
)

func (this *StatisticsManager) GetByEnv(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	node_labels, err := params.GetString("node_labels")
	ret := jsonutils.NewDict()
	if err != nil {
		return ret, err
	}

	path := fmt.Sprintf("/%s/env?node_labels=%s", this.KeywordPlural, node_labels)
	return modulebase.Get(this.ResourceManager, s, path, this.Keyword)
}

func (this *StatisticsManager) GetByResType(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	node_labels, err := params.GetString("node_labels")
	ret := jsonutils.NewDict()
	if err != nil {
		return ret, err
	}

	path := fmt.Sprintf("/%s/res_type?node_labels=%s", this.KeywordPlural, node_labels)
	return modulebase.Get(this.ResourceManager, s, path, this.Keyword)
}

func (this *StatisticsManager) GetHardware(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	node_labels, err := params.GetString("node_labels")
	ret := jsonutils.NewDict()
	if err != nil {
		return ret, err
	}

	path := fmt.Sprintf("/%s/hardware?node_labels=%s", this.KeywordPlural, node_labels)
	return modulebase.Get(this.ResourceManager, s, path, this.Keyword)
}

func init() {
	Statistics = StatisticsManager{NewServiceTreeManager("statistic", "statistics",
		[]string{"ID"},
		[]string{})}

	register(&Statistics)
}
