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

package compute

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type AgentListOptions struct {
	options.BaseListOptions
}

func (o AgentListOptions) Params() (jsonutils.JSONObject, error) {
	return o.BaseListOptions.Params()
}

func (o AgentListOptions) Description() string {
	return "List all agent"
}

type AgentOpsOperations struct {
	ID string `help:"ID or name of agent"`
}

func (o AgentOpsOperations) GetId() string {
	return o.ID
}

func (o AgentOpsOperations) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type AgentShowOpt struct {
	AgentOpsOperations
}

func (o AgentShowOpt) Description() string {
	return "Show details of an agent"
}

type AgentEnableOpt struct {
	AgentOpsOperations
}

func (o AgentEnableOpt) Description() string {
	return "Enable agent"
}

type AgentDisableOpt struct {
	AgentOpsOperations
}

func (o AgentDisableOpt) Description() string {
	return "Disable agent"
}

type AgentDeleteOpt struct {
	AgentOpsOperations
}

func (o AgentDeleteOpt) Description() string {
	return "Delete agent"
}

type AgentEnableImageCacheOpt struct {
	AgentOpsOperations
}

func (o AgentEnableImageCacheOpt) Description() string {
	return "Enable cache image of a agent"
}

type AgentDisableImageCacheOpt struct {
	AgentOpsOperations
}

func (o AgentDisableImageCacheOpt) Description() string {
	return "Disable cache image of a agent"
}

func init() {
	cmd := shell.NewResourceCmd(&modules.Baremetalagents).WithKeyword("agent")
	cmd.List(new(AgentListOptions))
	cmd.Show(new(AgentShowOpt))
	cmd.Perform("enable", new(AgentEnableOpt))
	cmd.Perform("disable", new(AgentDisableOpt))
	cmd.Perform("enable-image-cache", new(AgentEnableImageCacheOpt))
	cmd.Perform("disable-image-cache", new(AgentDisableImageCacheOpt))
	cmd.Delete(new(AgentDeleteOpt))
}
