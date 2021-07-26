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

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/huaweistack/client/manager"
	"yunion.io/x/onecloud/pkg/multicloud/huaweistack/client/responses"
)

type SGroupManager struct {
	SResourceManager
}

func NewGroupManager(cfg manager.IManagerConfig) *SGroupManager {
	m := &SGroupManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(cfg),
		ServiceName:   ServiceNameIAM,
		Region:        cfg.GetRegionId(),
		ProjectId:     "",
		version:       "v3",
		Keyword:       "group",
		KeywordPlural: "groups",

		ResourceKeyword: "groups",
	}}
	m.SetDomainId(cfg.GetDomainId())
	return m
}

func (manager *SGroupManager) ListRoles(domainId string, groupId string) (*responses.ListResult, error) {
	if len(domainId) == 0 {
		return nil, fmt.Errorf("missing domainId")
	}
	if len(groupId) == 0 {
		return nil, fmt.Errorf("missing groupId")
	}
	manager.SetVersion(fmt.Sprintf("v3/domains/%s", domainId))
	return manager.ListInContextWithSpec(nil, fmt.Sprintf("%s/roles", groupId), nil, "roles")
}

func (manager *SGroupManager) DeleteProjectRole(projectId, groupId, roleId string) error {
	if len(projectId) == 0 {
		return fmt.Errorf("missing projectId")
	}
	if len(groupId) == 0 {
		return fmt.Errorf("missing groupId")
	}
	if len(roleId) == 0 {
		return fmt.Errorf("missing roleId")
	}
	manager.SetVersion(fmt.Sprintf("v3/projects/%s", projectId))
	_, err := manager.DeleteInContextWithSpec(nil, groupId, fmt.Sprintf("roles/%s", roleId), nil, nil, "")
	if err != nil && errors.Cause(err) == cloudprovider.ErrNotFound {
		return nil
	}
	return err
}

func (manager *SGroupManager) DeleteRole(domainId string, groupId, roleId string) error {
	if len(domainId) == 0 {
		return fmt.Errorf("missing domainId")
	}
	if len(groupId) == 0 {
		return fmt.Errorf("missing groupId")
	}
	if len(roleId) == 0 {
		return fmt.Errorf("missing roleId")
	}
	manager.SetVersion(fmt.Sprintf("v3/domains/%s", domainId))
	_, err := manager.DeleteInContextWithSpec(nil, groupId, fmt.Sprintf("roles/%s", roleId), nil, nil, "")
	if err != nil && errors.Cause(err) == cloudprovider.ErrNotFound {
		return nil
	}
	return err
}

func (manager *SGroupManager) AddProjectRole(projectId string, groupId, roleId string) error {
	if len(projectId) == 0 {
		return fmt.Errorf("missing projectId")
	}
	if len(groupId) == 0 {
		return fmt.Errorf("missing groupId")
	}
	if len(roleId) == 0 {
		return fmt.Errorf("missing roleId")
	}
	manager.SetVersion(fmt.Sprintf("v3/projects/%s", projectId))
	_, err := manager.UpdateInContextWithSpec(nil, groupId, fmt.Sprintf("roles/%s", roleId), nil, "")
	return err
}

func (manager *SGroupManager) AddRole(domainId string, groupId, roleId string) error {
	if len(domainId) == 0 {
		return fmt.Errorf("missing domainId")
	}
	if len(groupId) == 0 {
		return fmt.Errorf("missing groupId")
	}
	if len(roleId) == 0 {
		return fmt.Errorf("missing roleId")
	}
	manager.SetVersion(fmt.Sprintf("v3/domains/%s", domainId))
	_, err := manager.UpdateInContextWithSpec(nil, groupId, fmt.Sprintf("roles/%s", roleId), nil, "")
	return err
}
