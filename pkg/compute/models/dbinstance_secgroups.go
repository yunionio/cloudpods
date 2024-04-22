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

	"gopkg.in/fatih/set.v0"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-ignore
type SDBInstanceSecgroupManager struct {
	SDBInstanceJointsManager
	SSecurityGroupResourceBaseManager
}

var DBInstanceSecgroupManager *SDBInstanceSecgroupManager

func init() {
	db.InitManager(func() {
		DBInstanceSecgroupManager = &SDBInstanceSecgroupManager{
			SDBInstanceJointsManager: NewDBInstanceJointsManager(
				SDBInstanceSecgroup{},
				"dbinstance_secgroups_tbl",
				"dbinstance_secgroup",
				"dbinstance_secgroups",
				SecurityGroupManager,
			),
		}
		DBInstanceSecgroupManager.SetVirtualObject(DBInstanceSecgroupManager)
	})
}

// +onecloud:swagger-gen-ignore
type SDBInstanceSecgroup struct {
	SDBInstanceJointsBase

	SSecurityGroupResourceBase `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
}

func (manager *SDBInstanceSecgroupManager) GetSlaveFieldName() string {
	return "secgroup_id"
}

func (self *SDBInstanceSecgroup) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (manager *SDBInstanceSecgroupManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DBInstanceSecgroupListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SDBInstanceJointsManager.ListItemFilter(ctx, q, userCred, query.DBInstanceJoinListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDBInstanceJointsManager.ListItemFilter")
	}
	q, err = manager.SSecurityGroupResourceBaseManager.ListItemFilter(ctx, q, userCred, query.SecgroupFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSecurityGroupResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (manager *SDBInstanceSecgroupManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DBInstanceSecgroupListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SDBInstanceJointsManager.OrderByExtraFields(ctx, q, userCred, query.DBInstanceJoinListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDBInstanceJointsManager.OrderByExtraFields")
	}
	q, err = manager.SSecurityGroupResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.SecgroupFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSecurityGroupResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SDBInstanceSecgroupManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualJointResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualJointResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SSecurityGroupResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SSecurityGroupResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SSecurityGroupResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}

func (manager *SDBInstanceSecgroupManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.DBInstanceSecgroupDetails {
	rows := make([]api.DBInstanceSecgroupDetails, len(objs))

	vjRows := manager.SVirtualJointResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	secgrpIds := make([]string, len(rows))
	rdsIds := make([]string, len(rows))
	for i := range rows {
		rows[i].VirtualJointResourceBaseDetails = vjRows[i]
		rdsSec := objs[i].(*SDBInstanceSecgroup)
		secgrpIds[i], rdsIds[i] = rdsSec.SecgroupId, rdsSec.DBInstanceId
	}

	secMaps, err := db.FetchIdNameMap2(SecurityGroupManager, secgrpIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 for SecurityGroupManager fail %s", err)
		return rows
	}
	rdsMaps, err := db.FetchIdNameMap2(DBInstanceManager, rdsIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 for DBInstanceManager fail %s", err)
		return rows
	}

	for i := range rows {
		rows[i].Secgroup, _ = secMaps[secgrpIds[i]]
		rows[i].DBInstance, _ = rdsMaps[rdsIds[i]]
	}

	return rows
}

func (manager *SDBInstanceSecgroupManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SVirtualJointResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SDBInstanceSecgroupManager) SyncDBInstanceSecgroups(ctx context.Context, userCred mcclient.TokenCredential, rds *SDBInstance, extIds []string) compare.SyncResult {
	result := compare.SyncResult{}

	secgroups, err := rds.GetSecgroups()
	if err != nil {
		result.Error(err)
		return result
	}

	extSecgroups, err := rds.getSecgroupsByExternalIds(extIds)
	if err != nil {
		result.Error(err)
		return result
	}

	localSet := set.New(set.ThreadSafe)
	for i := range secgroups {
		localSet.Add(secgroups[i].Id)
	}
	remoteSet := set.New(set.ThreadSafe)
	for i := range extSecgroups {
		remoteSet.Add(extSecgroups[i].Id)
	}
	for _, del := range set.Difference(localSet, remoteSet).List() {
		err = rds.RevokeSecgroup(ctx, userCred, del.(string))
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}
	for _, add := range set.Difference(remoteSet, localSet).List() {
		err = rds.AssignSecgroup(ctx, userCred, add.(string))
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}
	return result
}
