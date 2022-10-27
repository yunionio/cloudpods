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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

// https://support.huaweicloud.com/api-em/zh-cn_topic_0121230880.html
type SEnterpriseProject struct {
	multicloud.SProjectBase
	HuaweiTags

	Id          string
	Name        string
	Description string
	Status      int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (self *SHuaweiClient) GetEnterpriseProjects() ([]SEnterpriseProject, error) {
	projects := []SEnterpriseProject{}
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return nil, errors.Wrap(err, "newGeneralAPIClient")
	}
	err = doListAllWithOffset(client.EnterpriseProjects.List, map[string]string{}, &projects)
	if err != nil {
		return nil, errors.Wrap(err, "doListAllWithOffset")
	}
	return projects, nil
}

func (ep *SEnterpriseProject) GetId() string {
	return ep.Id
}

func (ep *SEnterpriseProject) GetGlobalId() string {
	return ep.Id
}

func (ep *SEnterpriseProject) GetStatus() string {
	if ep.Status == 1 {
		return api.EXTERNAL_PROJECT_STATUS_AVAILABLE
	}
	return api.EXTERNAL_PROJECT_STATUS_UNAVAILABLE
}

func (ep *SEnterpriseProject) GetName() string {
	return ep.Name
}

func (self *SHuaweiClient) CreateExterpriseProject(name, desc string) (*SEnterpriseProject, error) {
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return nil, errors.Wrap(err, "newGeneralAPIClient")
	}
	params := map[string]string{
		"name": name,
	}
	if len(desc) > 0 {
		params["description"] = desc
	}
	resp, err := client.EnterpriseProjects.Create(jsonutils.Marshal(params))
	if err != nil {
		if strings.Contains(err.Error(), "EPS.0004") {
			return nil, cloudprovider.ErrNotSupported
		}
		return nil, errors.Wrap(err, "EnterpriseProjects.Create")
	}
	project := &SEnterpriseProject{}
	err = resp.Unmarshal(&project)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return project, nil
}

func (self *SHuaweiClient) CreateIProject(name string) (cloudprovider.ICloudProject, error) {
	return self.CreateExterpriseProject(name, "")
}

func (self *SHuaweiClient) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	projects, err := self.GetEnterpriseProjects()
	if err != nil {
		return nil, errors.Wrap(err, "GetProjects")
	}
	ret := []cloudprovider.ICloudProject{}
	for i := range projects {
		ret = append(ret, &projects[i])
	}
	return ret, nil
}
