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
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SResourceBase struct {
	RemoteFileTags
	multicloud.SBillingBase
	Id        string
	Name      string
	Emulated  bool
	Status    string
	CreatedAt time.Time
	ProjectId string
}

func (self *SResourceBase) GetId() string {
	return self.Id
}

func (self *SResourceBase) GetName() string {
	return self.Name
}

func (self *SResourceBase) Refresh() error {
	return nil
}

func (self *SResourceBase) GetStatus() string {
	if len(self.Status) == 0 {
		return "unknown"
	}
	return self.Status
}

func (self *SResourceBase) GetProjectId() string {
	return self.ProjectId
}

func (self *SResourceBase) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(self.GetName()).CN(self.GetName()).EN(self.GetName())
	return table
}

func (self *SResourceBase) GetGlobalId() string {
	if len(self.Id) == 0 {
		panic("empty id")
	}
	return self.Id
}

func (self *SResourceBase) IsEmulated() bool {
	return self.Emulated
}

func (self *SResourceBase) GetCreatedAt() time.Time {
	return self.CreatedAt
}

func (self *SResourceBase) GetSysTags() map[string]string {
	return self.RemoteFileTags.GetSysTags()
}

func (self *SResourceBase) GetTags() (map[string]string, error) {
	return self.RemoteFileTags.GetTags()
}

func (self *SResourceBase) SetTags(tags map[string]string, replace bool) error {
	return self.RemoteFileTags.SetTags(tags, replace)
}
