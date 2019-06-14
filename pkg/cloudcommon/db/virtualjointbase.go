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
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
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
	if jsonutils.QueryBoolean(query, "admin", false) && !IsAllowList(rbacutils.ScopeSystem, userCred, manager) {
		return false
	}
	return true
	// return manager.SJointResourceBaseManager.AllowListItems(ctx, userCred, query)
}

func (manager *SVirtualJointResourceBaseManager) AllowListDescendent(ctx context.Context, userCred mcclient.TokenCredential, master IStandaloneModel, query jsonutils.JSONObject) bool {
	masterVirtual := master.(IVirtualModel)
	if masterVirtual.IsOwner(userCred) || IsAllowList(rbacutils.ScopeSystem, userCred, manager) {
		return true
	}
	return false
}

func (manager *SVirtualJointResourceBaseManager) AllowAttach(ctx context.Context, userCred mcclient.TokenCredential, master IStandaloneModel, slave IStandaloneModel) bool {
	masterVirtual := master.(IVirtualModel)
	slaveVirtual := slave.(IVirtualModel)
	if masterVirtual.GetOwnerId() == slaveVirtual.GetOwnerId() {
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
	return masterVirtual.IsOwner(userCred) || IsAllowGet(rbacutils.ScopeSystem, userCred, self)
}

func (self *SVirtualJointResourceBase) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	masterVirtual := self.Master().(IVirtualModel)
	return masterVirtual.IsOwner(userCred) || IsAllowUpdate(rbacutils.ScopeSystem, userCred, self)
}

func (manager *SVirtualJointResourceBaseManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		masterQ := manager.GetMasterManager().Query("id")
		masterQ = manager.GetMasterManager().FilterByOwner(masterQ, owner, scope)
		slaveQ := manager.GetSlaveManager().Query("id")
		slaveQ = manager.GetSlaveManager().FilterByOwner(slaveQ, owner, scope)
		iManager := manager.GetIJointModelManager()
		q = q.In(iManager.GetMasterFieldName(), masterQ.SubQuery())
		q = q.In(iManager.GetSlaveFieldName(), slaveQ.SubQuery())
	}
	return q
}

func (manager *SVirtualJointResourceBaseManager) FilterBySystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	q = manager.SJointResourceBaseManager.FilterBySystemAttributes(q, userCred, query, scope)
	masterQ := manager.GetMasterManager().Query("id")
	masterQ = manager.GetMasterManager().FilterBySystemAttributes(masterQ, userCred, query, scope)
	slaveQ := manager.GetSlaveManager().Query("id")
	slaveQ = manager.GetSlaveManager().FilterBySystemAttributes(slaveQ, userCred, query, scope)
	iManager := manager.GetIJointModelManager()
	q = q.In(iManager.GetMasterFieldName(), masterQ.SubQuery())
	q = q.In(iManager.GetSlaveFieldName(), slaveQ.SubQuery())
	return q
}
