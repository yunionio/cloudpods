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

type SDomainManager struct {
	SResourceManager
}

func NewDomainManager(cfg manager.IManagerConfig) *SDomainManager {
	m := &SDomainManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(cfg),
		ServiceName:   ServiceNameIAM,
		Region:        "",
		ProjectId:     "",
		version:       "v3/auth",
		Keyword:       "domain",
		KeywordPlural: "domains",

		ResourceKeyword: "domains",
	}}
	m.SetDomainId(cfg.GetDomainId())
	return m
}

func (manager *SDomainManager) DeleteRole(domainId string, groupId, roleId string) error {
	if len(domainId) == 0 {
		return fmt.Errorf("missing domainId")
	}
	if len(groupId) == 0 {
		return fmt.Errorf("missing groupId")
	}
	if len(roleId) == 0 {
		return fmt.Errorf("missing roleId")
	}
	manager.SetVersion("v3")
	defer manager.SetVersion("v3/auth")
	res := fmt.Sprintf("groups/%s/roles/%s", groupId, roleId)
	_, err := manager.DeleteInContextWithSpec(nil, domainId, res, nil, nil, "")
	if err != nil && errors.Cause(err) == cloudprovider.ErrNotFound {
		return nil
	}
	return err
}

func (manager *SDomainManager) ListRoles(domainId string, groupId string) (*responses.ListResult, error) {
	if len(domainId) == 0 {
		return nil, fmt.Errorf("missing domainId")
	}
	if len(groupId) == 0 {
		return nil, fmt.Errorf("missing groupId")
	}
	manager.SetVersion("v3")
	defer manager.SetVersion("v3/auth")
	res := fmt.Sprintf("%s/groups/%s/roles", domainId, groupId)
	return manager.ListInContextWithSpec(nil, res, nil, "roles")
}

func (manager *SDomainManager) AddRole(domainId string, groupId, roleId string) error {
	if len(domainId) == 0 {
		return fmt.Errorf("missing domainId")
	}
	if len(groupId) == 0 {
		return fmt.Errorf("missing groupId")
	}
	if len(roleId) == 0 {
		return fmt.Errorf("missing roleId")
	}
	manager.SetVersion("v3")
	defer manager.SetVersion("v3/auth")
	res := fmt.Sprintf("groups/%s/roles/%s", groupId, roleId)
	_, err := manager.UpdateInContextWithSpec(nil, domainId, res, nil, "")
	return err
}
