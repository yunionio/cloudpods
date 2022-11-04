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
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

type SScopeResourceManager struct {
	db.SModelBaseManager
}

var ScopeResourceManager *SScopeResourceManager

func init() {
	ScopeResourceManager = &SScopeResourceManager{
		SModelBaseManager: db.NewModelBaseManager(
			SScopeResource{},
			"scope_resource_tbl",
			"scope_resource",
			"scope_resources",
		),
	}
	ScopeResourceManager.SetVirtualObject(ScopeResourceManager)
}

type SScopeResource struct {
	db.SModelBase

	DomainId  string `width:"64" charset:"ascii" primary:"true"`
	ProjectId string `width:"64" charset:"ascii" primary:"true"`
	OwnerId   string `width:"64" charset:"ascii" primary:"true"`
	RegionId  string `width:"32" charset:"ascii" primary:"true"`
	ServiceId string `width:"32" charset:"ascii" primary:"true"`
	Resource  string `width:"32" charset:"ascii" primary:"true"`
	Count     int
	UpdatedAt time.Time `nullable:"true" updated_at:"true"`
}

type sScopeResourceCount struct {
	Resource   string
	ResCount   int
	ProjectId  string
	LastUpdate time.Time
}

func (manager *SScopeResourceManager) getScopeResource(domainId, projId, ownerId string) (map[string]int, time.Time, error) {
	resources := manager.Query().SubQuery()
	q := resources.Query(
		resources.Field("resource"),
		sqlchemy.SUM("res_count", resources.Field("count")),
		sqlchemy.MAX("last_update", resources.Field("updated_at")),
	)
	if len(domainId) > 0 {
		q = q.Filter(sqlchemy.Equals(resources.Field("domain_id"), domainId))
	}
	if len(projId) > 0 {
		q = q.Filter(sqlchemy.Equals(resources.Field("project_id"), projId))
	}
	if len(ownerId) > 0 {
		q = q.Filter(sqlchemy.Equals(resources.Field("owner_id"), ownerId))
	}
	q = q.GroupBy(resources.Field("resource"))
	resCnts := make([]sScopeResourceCount, 0)
	err := q.All(&resCnts)
	if err != nil && err != sql.ErrNoRows {
		return nil, time.Time{}, errors.Wrap(err, "query.All")
	}
	ret := make(map[string]int)
	lastUpdate := time.Time{}
	for i := range resCnts {
		if resCnts[i].ResCount == 0 {
			continue
		}
		ret[resCnts[i].Resource] = resCnts[i].ResCount
		if lastUpdate.IsZero() || lastUpdate.Before(resCnts[i].LastUpdate) {
			lastUpdate = resCnts[i].LastUpdate
		}
	}
	return ret, lastUpdate, nil
}

func (manager *SScopeResourceManager) FetchProjectsScopeResources(projIds []string) (map[string]map[string]int, map[string]time.Time, error) {
	resources := manager.Query().SubQuery()
	q := resources.Query(
		resources.Field("resource"),
		resources.Field("project_id"),
		sqlchemy.SUM("res_count", resources.Field("count")),
		sqlchemy.MAX("last_update", resources.Field("updated_at")),
	).Filter(sqlchemy.In(resources.Field("project_id"), projIds))
	q = q.GroupBy(resources.Field("resource"))
	resCnts := make([]sScopeResourceCount, 0)
	err := q.All(&resCnts)
	if err != nil && err != sql.ErrNoRows {
		return nil, nil, errors.Wrap(err, "query.All")
	}
	ret := make(map[string]map[string]int)
	lastUpdate := map[string]time.Time{}
	for i := range resCnts {
		_, ok := ret[resCnts[i].ProjectId]
		if !ok {
			ret[resCnts[i].ProjectId] = map[string]int{}
			lastUpdate[resCnts[i].ProjectId] = resCnts[i].LastUpdate
		}
		if resCnts[i].ResCount == 0 {
			continue
		}
		last := lastUpdate[resCnts[i].ProjectId]
		ret[resCnts[i].ProjectId][resCnts[i].Resource] = resCnts[i].ResCount
		if last.IsZero() || last.Before(resCnts[i].LastUpdate) {
			lastUpdate[resCnts[i].ProjectId] = resCnts[i].LastUpdate
		}
	}
	return ret, lastUpdate, nil
}
