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

package vpcagent

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	VpcAgent VpcagentManager
)

func NewVpcagentManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(apis.SERVICE_TYPE_VPCAGENT, "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func init() {
	VpcAgent = VpcagentManager{
		ResourceManager: NewVpcagentManager("vpcagent", "vpcagent", []string{}, []string{}),
	}
	modules.Register(&VpcAgent)
}

type VpcagentManager struct {
	modulebase.ResourceManager
}

func (agent *VpcagentManager) DoSync(s *mcclient.ClientSession) error {
	params := jsonutils.NewDict()
	_, err := agent.PerformAction(s, "api", "sync", params)
	return errors.Wrap(err, "syncs")
}
