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

package k8s

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Charts *ChartManager
)

type ChartManager struct {
	*ResourceManager
}

func init() {
	Charts = &ChartManager{NewResourceManager("chart", "charts",
		NewColumns("RepoWithName", "Version", "Description"),
		NewColumns())}
	modules.Register(Charts)
}

func (m ChartManager) Get_RepoWithName(obj jsonutils.JSONObject) interface{} {
	repo, _ := obj.GetString("repo")
	name, _ := obj.GetString("chart", "name")
	return fmt.Sprintf("%s/%s", repo, name)
}

func (m ChartManager) Get_Version(obj jsonutils.JSONObject) interface{} {
	version, _ := obj.GetString("chart", "version")
	return version
}

func (m ChartManager) Get_Description(obj jsonutils.JSONObject) interface{} {
	desc, _ := obj.GetString("chart", "description")
	return desc
}
