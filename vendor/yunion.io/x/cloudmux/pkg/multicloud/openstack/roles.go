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

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SRole struct {
	Id   string
	Name string
}

func (cli *SOpenStackClient) GetRoles(name string) ([]SRole, error) {
	resource := "/v3/roles"
	query := url.Values{}
	if len(name) > 0 {
		query.Set("name", name)
	}
	resp, err := cli.iamRequest(cli.getDefaultRegionName(), httputils.GET, resource, query, nil)
	if err != nil {
		return nil, errors.Wrap(err, "iamRequest")
	}
	roles := []SRole{}
	err = resp.Unmarshal(&roles, "roles")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return roles, nil
}

func (cli *SOpenStackClient) AssignRoleToUserOnProject(userId, projectId, roleName string) error {
	if len(roleName) == 0 {
		return errors.Error("empty role name")
	}
	roles, err := cli.GetRoles(roleName)
	if err != nil {
		return errors.Wrapf(err, "GetRoles(%s)", roleName)
	}
	if len(roles) == 0 {
		return errors.Wrapf(cloudprovider.ErrNotFound, "role %s", roleName)
	}
	if len(roles) > 1 {
		return errors.Wrapf(cloudprovider.ErrDuplicateId, "roles %d for %s", len(roles), roleName)
	}
	resource := fmt.Sprintf("/v3/projects/%s/users/%s/roles/%s", projectId, userId, roles[0].Id)
	_, err = cli.iamRequest(cli.getDefaultRegionName(), httputils.PUT, resource, nil, map[string]string{})
	return err
}
