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

package models

import (
	"database/sql"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

type SProjectResourceManager struct {
	db.SModelBaseManager
}

var ProjectResourceManager *SProjectResourceManager

func init() {
	ProjectResourceManager = &SProjectResourceManager{
		SModelBaseManager: db.NewModelBaseManager(
			SProjectResource{},
			"project_resource_tbl",
			"project_resource",
			"project_resources",
		),
	}
	ProjectResourceManager.SetVirtualObject(ProjectResourceManager)
}

type SProjectResource struct {
	db.SModelBase

	ProjectId string `width:"64" charset:"ascii" primary:"true"`
	RegionId  string `width:"32" charset:"ascii" primary:"true"`
	ServiceId string `width:"32" charset:"ascii" primary:"true"`
	Resource  string `width:"32" charset:"ascii" primary:"true"`
	Count     int
}

type sProjectResourceCount struct {
	Resource string
	ResCount int
}

func (manager *SProjectResourceManager) getProjectResource(projId string) (map[string]int, error) {
	resources := manager.Query().SubQuery()
	q := resources.Query(
		resources.Field("resource"),
		sqlchemy.SUM("res_count", resources.Field("count")),
	)
	q = q.Filter(sqlchemy.Equals(resources.Field("project_id"), projId))
	q = q.GroupBy(resources.Field("resource"))
	resCnts := make([]sProjectResourceCount, 0)
	err := q.All(&resCnts)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "query.All")
	}
	ret := make(map[string]int)
	for i := range resCnts {
		if resCnts[i].ResCount == 0 {
			continue
		}
		ret[resCnts[i].Resource] = resCnts[i].ResCount
	}
	return ret, nil
}
