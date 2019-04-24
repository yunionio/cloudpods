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
	"fmt"
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SVirtualJointResourceBase struct {
	SJointResourceBase
}

type SVirtualJointResourceBaseManager struct {
	SJointResourceBaseManager
}

func NewVirtualJointResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string, master IVirtualModelManager, slave IVirtualModelManager) SVirtualJointResourceBaseManager {
	return SVirtualJointResourceBaseManager{SJointResourceBaseManager: NewJointResourceBaseManager(dt, tableName, keyword, keywordPlural, master, slave)}
}

func (manager *SVirtualJointResourceBaseManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	if jsonutils.QueryBoolean(query, "admin", false) && !IsAdminAllowList(userCred, manager) {
		return false
	}
	return true
	// return manager.SJointResourceBaseManager.AllowListItems(ctx, userCred, query)
}

func (manager *SVirtualJointResourceBaseManager) AllowListDescendent(ctx context.Context, userCred mcclient.TokenCredential, master IStandaloneModel, query jsonutils.JSONObject) bool {
	masterVirtual := master.(IVirtualModel)
	if masterVirtual.IsOwner(userCred) || IsAdminAllowList(userCred, manager) {
		return true
	}
	return false
}

func (manager *SVirtualJointResourceBaseManager) AllowAttach(ctx context.Context, userCred mcclient.TokenCredential, master IStandaloneModel, slave IStandaloneModel) bool {
	masterVirtual := master.(IVirtualModel)
	slaveVirtual := slave.(IVirtualModel)
	if masterVirtual.GetOwnerProjectId() == slaveVirtual.GetOwnerProjectId() {
		return true
	} else {
		slaveValue := reflect.Indirect(reflect.ValueOf(slaveVirtual))
		val, find := reflectutils.FindStructFieldInterface(slaveValue, "IsPublic")
		if find {
			valBool, ok := val.(bool)
			if ok && valBool {
				return true
			}
		}
	}
	return false
}

func (self *SVirtualJointResourceBase) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	masterVirtual := self.Master().(IVirtualModel)
	return masterVirtual.IsOwner(userCred) || IsAdminAllowGet(userCred, self)
}

func (self *SVirtualJointResourceBase) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	masterVirtual := self.Master().(IVirtualModel)
	return masterVirtual.IsOwner(userCred) || IsAdminAllowUpdate(userCred, self)
}

func (manager *SVirtualJointResourceBaseManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SJointResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	masterField := manager.MasterField(q)
	slaveField := manager.SlaveField(q)
	if masterField == nil || slaveField == nil {
		msg := "cannot find master or slave fields!!!"
		log.Errorf(msg)
		return nil, fmt.Errorf(msg)
	}
	masterTable := manager.GetMasterManager().Query().SubQuery()
	slaveTable := manager.GetSlaveManager().Query().SubQuery()
	masterQueryId, _ := query.GetString(fmt.Sprintf("%s_id", manager.GetMasterManager().Keyword()))
	if len(masterQueryId) == 0 && len(manager.GetMasterManager().Alias()) > 0 {
		masterQueryId, _ = query.GetString(fmt.Sprintf("%s_id", manager.GetMasterManager().Alias()))
	}
	slaveQueryId, _ := query.GetString(fmt.Sprintf("%s_id", manager.GetSlaveManager().Keyword()))
	if len(slaveQueryId) == 0 && len(manager.GetSlaveManager().Alias()) > 0 {
		slaveQueryId, _ = query.GetString(fmt.Sprintf("%s_id", manager.GetSlaveManager().Alias()))
	}
	q = q.Join(masterTable, sqlchemy.AND(sqlchemy.Equals(masterField, masterTable.Field("id")),
		sqlchemy.IsFalse(masterTable.Field("deleted"))))
	q = q.Join(slaveTable, sqlchemy.AND(sqlchemy.Equals(slaveField, slaveTable.Field("id")),
		sqlchemy.IsFalse(slaveTable.Field("deleted"))))
	if jsonutils.QueryBoolean(query, "admin", false) && IsAdminAllowList(userCred, manager) {
		isSystem := jsonutils.QueryBoolean(query, "system", false)
		if !isSystem {
			if len(slaveQueryId) == 0 {
				q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(slaveTable.Field("is_system")),
					sqlchemy.IsFalse(slaveTable.Field("is_system"))))
			}
			if len(masterQueryId) == 0 {
				q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(masterTable.Field("is_system")),
					sqlchemy.IsFalse(masterTable.Field("is_system"))))
			}
		}
		tenant, _ := query.GetString("tenant")
		if len(tenant) > 0 {
			tc, _ := TenantCacheManager.FetchTenantByIdOrName(ctx, tenant)
			if tc == nil {
				return nil, httperrors.NewTenantNotFoundError("tenant %s not found", tenant)
			}
			q = q.Filter(sqlchemy.OR(sqlchemy.Equals(masterTable.Field("tenant_id"), tc.GetId()),
				sqlchemy.Equals(slaveTable.Field("tenant_id"), tc.GetId())))
		}
	} else {
		q = q.Filter(sqlchemy.OR(sqlchemy.Equals(masterTable.Field("tenant_id"), userCred.GetProjectId()),
			sqlchemy.Equals(slaveTable.Field("tenant_id"), userCred.GetProjectId())))
		if len(slaveQueryId) == 0 {
			q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(slaveTable.Field("is_system")),
				sqlchemy.IsFalse(slaveTable.Field("is_system"))))
		}
		if len(masterQueryId) == 0 {
			q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(masterTable.Field("is_system")),
				sqlchemy.IsFalse(masterTable.Field("is_system"))))
		}
	}
	return q, nil
}
