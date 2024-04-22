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
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-ignore
type SElasticcachesecgroupManager struct {
	SElasticcacheJointsManager
	SSecurityGroupResourceBaseManager
}

var ElasticcachesecgroupManager *SElasticcachesecgroupManager

func init() {
	db.InitManager(func() {
		ElasticcachesecgroupManager = &SElasticcachesecgroupManager{
			SElasticcacheJointsManager: NewElasticcacheJointsManager(
				SElasticcachesecgroup{},
				"elasticcachesecgroups_tbl",
				"elasticcachesecgroup",
				"elasticcachesecgroups",
				SecurityGroupManager,
			),
		}
		ElasticcachesecgroupManager.SetVirtualObject(ElasticcachesecgroupManager)
	})
}

// +onecloud:model-api-gen
type SElasticcachesecgroup struct {
	SElasticcacheJointsBase

	SSecurityGroupResourceBase `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
}

func (manager *SElasticcachesecgroupManager) GetSlaveFieldName() string {
	return "secgroup_id"
}

func (self *SElasticcachesecgroup) getSecgroup() *SSecurityGroup {
	secgrp, err := SecurityGroupManager.FetchById(self.SecgroupId)
	if err != nil {
		log.Errorf("failed to find secgroup %s", self.SecgroupId)
		return nil
	}
	secgroup := secgrp.(*SSecurityGroup)
	secgroup.SetModelManager(SecurityGroupManager, secgroup)
	return secgroup
}

func (self *SElasticcachesecgroup) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (manager *SElasticcachesecgroupManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ElasticcachesecgroupListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SElasticcacheJointsManager.ListItemFilter(ctx, q, userCred, query.ElasticcacheJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SElasticcacheJointsManager.ListItemFilter")
	}
	q, err = manager.SSecurityGroupResourceBaseManager.ListItemFilter(ctx, q, userCred, query.SecgroupFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSecurityGroupResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (manager *SElasticcachesecgroupManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ElasticcachesecgroupListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SElasticcacheJointsManager.OrderByExtraFields(ctx, q, userCred, query.ElasticcacheJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SElasticcacheJointsManager.OrderByExtraFields")
	}
	q, err = manager.SSecurityGroupResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.SecgroupFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSecurityGroupResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SElasticcachesecgroupManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SElasticcacheJointsManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SElasticcacheJointsManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SSecurityGroupResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SSecurityGroupResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SSecurityGroupResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}

func (manager *SElasticcachesecgroupManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ElasticcachesecgroupDetails {
	rows := make([]api.ElasticcachesecgroupDetails, len(objs))

	ecRows := manager.SElasticcacheJointsManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	secgroupIds := make([]string, len(rows))
	for i := range rows {
		rows[i].ElasticcacheJointResourceDetails = ecRows[i]
		secgroupIds[i] = objs[i].(*SElasticcachesecgroup).SecgroupId
	}

	secgroupIdMaps, err := db.FetchIdNameMap2(SecurityGroupManager, secgroupIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 fail %s", err)
		return rows
	}

	for i := range rows {
		if name, ok := secgroupIdMaps[secgroupIds[i]]; ok {
			rows[i].Secgroup = name
		}
	}

	return rows
}

func fetchElasticcacheSecgroups(cacheIds []string) map[string][]apis.StandaloneShortDesc {
	secgroups := SecurityGroupManager.Query().SubQuery()
	cachesecgroups := ElasticcachesecgroupManager.Query().SubQuery()
	q := cachesecgroups.Query(
		cachesecgroups.Field("elasticcache_id", "elasticcache_id"),
		cachesecgroups.Field("secgroup_id", "secgroup_id"),
		secgroups.Field("name", "secgroup_name"))
	q = q.Filter(sqlchemy.In(cachesecgroups.Field("elasticcache_id"), cacheIds))
	q = q.Join(secgroups, sqlchemy.Equals(q.Field("secgroup_id"), secgroups.Field("id")))

	type secgroupInfo struct {
		SecgroupId     string
		SecgroupName   string
		ElasticcacheId string
	}

	gsgs := make([]secgroupInfo, 0)
	err := q.All(&gsgs)
	if err != nil {
		log.Debugf("fetchElasticcacheSecgroups %s", err)
		return nil
	}

	ret := make(map[string][]apis.StandaloneShortDesc)
	for i := range gsgs {
		gsg, ok := ret[gsgs[i].ElasticcacheId]
		if !ok {
			gsg = make([]apis.StandaloneShortDesc, 0)
		}
		gsg = append(gsg, apis.StandaloneShortDesc{
			Id:   gsgs[i].SecgroupId,
			Name: gsgs[i].SecgroupName,
		})
		ret[gsgs[i].ElasticcacheId] = gsg
	}

	return ret
}
