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

package huawei

import (
	"fmt"
	"strings"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/huawei/client"
)

// https://support.huaweicloud.com/api-iam/zh-cn_topic_0057845625.html
type SProject struct {
	client *SHuaweiClient

	IsDomain    bool   `json:"is_domain"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	ID          string `json:"id"`
	ParentID    string `json:"parent_id"`
	DomainID    string `json:"domain_id"`
	Name        string `json:"name"`
}

func (self *SProject) GetRegionID() string {
	return strings.Split(self.Name, "_")[0]
}

func (self *SProject) GetHealthStatus() string {
	if self.Enabled {
		return api.CLOUD_PROVIDER_HEALTH_NORMAL
	}

	return api.CLOUD_PROVIDER_HEALTH_SUSPENDED
}

func (self *SHuaweiClient) fetchProjects() ([]SProject, error) {
	huawei, _ := client.NewClientWithAccessKey("", "", self.accessKey, self.secret, self.debug)
	projects := make([]SProject, 0)
	err := doListAll(huawei.Projects.List, nil, &projects)
	return projects, err
}

func (self *SHuaweiClient) GetProjectById(projectId string) (SProject, error) {
	projects, err := self.fetchProjects()
	if err != nil {
		return SProject{}, err
	}

	for _, project := range projects {
		if project.ID == projectId {
			return project, nil
		}
	}
	return SProject{}, fmt.Errorf("project %s not found", projectId)
}

func (self *SHuaweiClient) GetProjects() ([]SProject, error) {
	return self.fetchProjects()
}

func (self *SHuaweiClient) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return nil, cloudprovider.ErrNotImplemented
}
