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

package openstack

import (
	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SProject struct {
	Description string
	Enabled     bool
	ID          string
	Name        string
}

func (p *SProject) GetId() string {
	return p.ID
}

func (p *SProject) GetGlobalId() string {
	return p.GetId()
}

func (p *SProject) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (p *SProject) GetName() string {
	return p.Name
}

func (p *SProject) GetStatus() string {
	return api.EXTERNAL_PROJECT_STATUS_AVAILABLE
}

func (p *SProject) IsEmulated() bool {
	return false
}

func (p *SProject) Refresh() error {
	return nil
}
