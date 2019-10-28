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

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/huawei/client/auth"
	"yunion.io/x/onecloud/pkg/multicloud/huawei/client/responses"
)

type SDBInstanceManager struct {
	SResourceManager
}

func NewDBInstanceManager(regionId string, projectId string, signer auth.Signer, debug bool) *SDBInstanceManager {
	return &SDBInstanceManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(signer, debug),
		ServiceName:   ServiceNameRDS,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v3",
		Keyword:       "",
		KeywordPlural: "instances",

		ResourceKeyword: "instances",
	}}
}

func (self *SDBInstanceManager) Get(id string, querys map[string]string) (jsonutils.JSONObject, error) {
	if len(id) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	resp, err := self.GetInContextWithSpec(nil, "", "", map[string]string{"id": id}, "")
	if err != nil {
		return nil, err
	}
	instances, err := resp.GetArray("instances")
	if err != nil {
		return nil, err
	}
	if len(instances) == 0 {
		return nil, cloudprovider.ErrNotFound
	} else if len(instances) == 1 {
		return instances[0], nil
	}
	return nil, cloudprovider.ErrDuplicateId
}

func (self *SDBInstanceManager) ListParameters(queries map[string]string) (*responses.ListResult, error) {
	id, _ := queries["instance_id"]
	if len(id) == 0 {
		return nil, fmt.Errorf("SDBInstanceManager.ListParameters missing parameter instance_id")
	}

	delete(queries, "instance_id")
	return self.ListInContextWithSpec(nil, fmt.Sprintf("%s/configurations", id), queries, "configuration_parameters")
}

func (self *SDBInstanceManager) ListDatabases(queries map[string]string) (*responses.ListResult, error) {
	id, _ := queries["instance_id"]
	if len(id) == 0 {
		return nil, fmt.Errorf("SDBInstanceManager.ListDatabases missing parameter instance_id")
	}

	delete(queries, "instance_id")
	return self.ListInContextWithSpec(nil, fmt.Sprintf("%s/database/detail", id), queries, "databases")
}

func (self *SDBInstanceManager) ListAccounts(queries map[string]string) (*responses.ListResult, error) {
	id, _ := queries["instance_id"]
	if len(id) == 0 {
		return nil, fmt.Errorf("SDBInstanceManager.ListAccounts missing parameter instance_id")
	}

	delete(queries, "instance_id")
	return self.ListInContextWithSpec(nil, fmt.Sprintf("%s/db_user/detail", id), queries, "users")
}

func (self *SDBInstanceManager) ListPrivileges(queries map[string]string) (*responses.ListResult, error) {
	id, _ := queries["instance_id"]
	if len(id) == 0 {
		return nil, fmt.Errorf("SDBInstanceManager.ListPrivileges missing parameter instance_id")
	}

	delete(queries, "instance_id")
	return self.ListInContextWithSpec(nil, fmt.Sprintf("%s/db_user/database", id), queries, "databases")
}
