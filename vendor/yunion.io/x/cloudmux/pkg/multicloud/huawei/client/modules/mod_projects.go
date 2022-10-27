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

package modules

import (
	"fmt"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei/client/manager"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei/client/responses"
)

type SProjectManager struct {
	SResourceManager
}

func NewProjectManager(cfg manager.IManagerConfig) *SProjectManager {
	m := &SProjectManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(cfg),
		ServiceName:   ServiceNameIAM,
		Region:        "",
		ProjectId:     "",
		version:       "v3",
		Keyword:       "project",
		KeywordPlural: "projects",

		ResourceKeyword: "projects",
	}}
	m.SetDomainId(cfg.GetDomainId())
	return m
}

func (manager *SProjectManager) ListRoles(projectId, groupId string) (*responses.ListResult, error) {
	if len(projectId) == 0 {
		return nil, fmt.Errorf("missing projectId")
	}
	if len(groupId) == 0 {
		return nil, fmt.Errorf("missing groupId")
	}
	res := fmt.Sprintf("%s/groups/%s/roles", projectId, groupId)
	return manager.ListInContextWithSpec(nil, res, nil, "roles")
}

func (manager *SProjectManager) DeleteProjectRole(projectId, groupId, roleId string) error {
	if len(projectId) == 0 {
		return fmt.Errorf("missing projectId")
	}
	if len(groupId) == 0 {
		return fmt.Errorf("missing groupId")
	}
	if len(roleId) == 0 {
		return fmt.Errorf("missing roleId")
	}
	res := fmt.Sprintf("groups/%s/roles/%s", groupId, roleId)
	_, err := manager.DeleteInContextWithSpec(nil, projectId, res, nil, nil, "")
	if err != nil && errors.Cause(err) == cloudprovider.ErrNotFound {
		return nil
	}
	return err
}

func (manager *SProjectManager) AddProjectRole(projectId string, groupId, roleId string) error {
	if len(projectId) == 0 {
		return fmt.Errorf("missing projectId")
	}
	if len(groupId) == 0 {
		return fmt.Errorf("missing groupId")
	}
	if len(roleId) == 0 {
		return fmt.Errorf("missing roleId")
	}
	res := fmt.Sprintf("groups/%s/roles/%s", groupId, roleId)
	_, err := manager.UpdateInContextWithSpec(nil, projectId, res, nil, "")
	return err
}
