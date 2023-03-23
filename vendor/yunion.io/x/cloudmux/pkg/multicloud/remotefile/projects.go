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

package remotefile

import (
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SProject struct {
	multicloud.SProjectBase
	RemoteFileTags

	Id   string
	Name string
}

func (project *SProject) GetGlobalId() string {
	return project.Id
}

func (project *SProject) GetId() string {
	return project.Id
}

func (project *SProject) GetName() string {
	return project.Name
}

func (project *SProject) Refresh() error {
	return nil
}

func (project *SProject) GetDescription() string {
	return ""
}

func (project *SProject) GetStatus() string {
	return api.EXTERNAL_PROJECT_STATUS_AVAILABLE
}
