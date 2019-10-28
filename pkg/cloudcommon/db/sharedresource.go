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

package db

import (
	"context"

	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// sharing resoure between project
type SSharedResource struct {
	SResourceBase

	Id int64 `primary:"true" auto_increment:"true" list:"user"`

	ResourceType    string `width:"32" charset:"ascii" nullable:"false" list:"user"`
	ResourceId      string `width:"128" charset:"ascii" nullable:"false" index:"true" list:"user"`
	OwnerProjectId  string `width:"128" charset:"ascii" nullable:"false" index:"true" list:"user"`
	TargetProjectId string `width:"128" charset:"ascii" nullable:"false" index:"true" list:"user"`
}

type SSharedResourceManager struct {
	SResourceBaseManager
}

var SharedResourceManager *SSharedResourceManager

func init() {
	SharedResourceManager = &SSharedResourceManager{
		SResourceBaseManager: NewResourceBaseManager(
			SSharedResource{},
			"shared_resources_tbl",
			"shared_resource",
			"shared_resources",
		),
	}
}

func (manager *SSharedResourceManager) CleanModelSharedProjects(ctx context.Context, userCred mcclient.TokenCredential, model *SVirtualResourceBase) error {
	srs := make([]SSharedResource, 0)
	q := manager.Query()
	err := q.Filter(sqlchemy.AND(
		sqlchemy.Equals(q.Field("owner_project_id"), model.ProjectId),
		sqlchemy.Equals(q.Field("resource_id"), model.GetId()),
		sqlchemy.Equals(q.Field("resource_type"), model.GetModelManager().Keyword()),
	)).All(&srs)
	if err != nil {
		return httperrors.NewInternalServerError("Fetch project error %s", err)
	}
	for i := 0; i < len(srs); i++ {
		srs[i].SetModelManager(manager, &srs[i])
		if err := srs[i].Delete(ctx, userCred); err != nil {
			return httperrors.NewInternalServerError("Unshare project failed %s", err)
		}
	}
	return nil
}
