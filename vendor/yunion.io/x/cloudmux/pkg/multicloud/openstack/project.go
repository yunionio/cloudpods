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
	"fmt"
	"net/url"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/cloudmux/pkg/multicloud/openstack/oscli"
)

type SProject struct {
	multicloud.SProjectBase
	OpenStackTags
	client      *SOpenStackClient
	Description string
	Enabled     bool
	Id          string
	Name        string
}

func (p *SProject) GetId() string {
	return p.Id
}

func (p *SProject) GetGlobalId() string {
	return p.GetId()
}

func (p *SProject) GetName() string {
	return p.Name
}

func (p *SProject) GetStatus() string {
	if !p.Enabled {
		return api.EXTERNAL_PROJECT_STATUS_UNKNOWN
	}
	_, err := p.getToken()
	if err != nil {
		log.Errorf("get project %s token error: %v %T", p.Name, err, err)
		return api.EXTERNAL_PROJECT_STATUS_UNKNOWN
	}
	return api.EXTERNAL_PROJECT_STATUS_AVAILABLE
}

func (p *SProject) getToken() (oscli.TokenCredential, error) {
	return p.client.getProjectToken(p.Id, p.Name)
}

func (p *SProject) IsEmulated() bool {
	return false
}

func (p *SProject) Refresh() error {
	return nil
}

type SProjectLinks struct {
	Next     string
	Previous string
	Self     string
}

func (link SProjectLinks) GetNextMark() string {
	if len(link.Next) == 0 || link.Next == "null" {
		return ""
	}
	next, err := url.Parse(link.Next)
	if err != nil {
		log.Errorf("parse next link %s error: %v", link.Next, err)
		return ""
	}
	return next.Query().Get("marker")
}

func (cli *SOpenStackClient) GetProjects() ([]SProject, error) {
	resource := "/v3/projects"
	projects := []SProject{}
	query := url.Values{}
	for {
		resp, err := cli.iamRequest("", httputils.GET, resource, query, nil)
		if err != nil {
			return nil, errors.Wrap(err, "iamRequest")
		}
		part := struct {
			Projects []SProject
			Links    SProjectLinks
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrap(err, "iamRequest")
		}
		projects = append(projects, part.Projects...)
		marker := part.Links.GetNextMark()
		if len(marker) == 0 {
			break
		}
		query.Set("marker", marker)
	}
	return projects, nil
}

func (cli *SOpenStackClient) DeleteProject(projectId string) error {
	resource := fmt.Sprintf("/v3/projects/%s", projectId)
	_, err := cli.iamRequest(cli.getDefaultRegionName(), httputils.DELETE, resource, nil, nil)
	return err
}
