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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
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

func (manager *SVirtualJointResourceBaseManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		masterQ := manager.GetMasterManager().Query("id")
		masterQ = manager.GetMasterManager().FilterByOwner(ctx, masterQ, manager.GetMasterManager(), userCred, owner, scope)
		slaveQ := manager.GetSlaveManager().Query("id")
		slaveQ = manager.GetSlaveManager().FilterByOwner(ctx, slaveQ, manager.GetSlaveManager(), userCred, owner, scope)
		iManager := manager.GetIJointModelManager()
		q = q.In(iManager.GetMasterFieldName(), masterQ.SubQuery())
		q = q.In(iManager.GetSlaveFieldName(), slaveQ.SubQuery())
	}
	return q
}

func (manager *SVirtualJointResourceBaseManager) FilterBySystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
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

func (manager *SVirtualJointResourceBaseManager) FilterByHiddenSystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	q = manager.SJointResourceBaseManager.FilterByHiddenSystemAttributes(q, userCred, query, scope)
	masterQ := manager.GetMasterManager().Query("id")
	masterQ = manager.GetMasterManager().FilterByHiddenSystemAttributes(masterQ, userCred, query, scope)
	slaveQ := manager.GetSlaveManager().Query("id")
	slaveQ = manager.GetSlaveManager().FilterByHiddenSystemAttributes(slaveQ, userCred, query, scope)
	iManager := manager.GetIJointModelManager()
	q = q.In(iManager.GetMasterFieldName(), masterQ.SubQuery())
	q = q.In(iManager.GetSlaveFieldName(), slaveQ.SubQuery())
	return q
}

func (manager *SVirtualJointResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.VirtualJointResourceBaseDetails {
	ret := make([]apis.VirtualJointResourceBaseDetails, len(objs))
	upperRet := manager.SJointResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range objs {
		ret[i] = apis.VirtualJointResourceBaseDetails{
			JointResourceBaseDetails: upperRet[i],
		}
	}
	return ret
}

func (manager *SVirtualJointResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.VirtualJointResourceBaseListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SJointResourceBaseManager.ListItemFilter(ctx, q, userCred, query.JointResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SJointResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (manager *SVirtualJointResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.VirtualJointResourceBaseListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SJointResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.JointResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SJointResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (self *SVirtualJointResourceBase) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.VirtualJointResourceBaseUpdateInput,
) (apis.VirtualJointResourceBaseUpdateInput, error) {
	var err error
	input.JointResourceBaseUpdateInput, err = self.SJointResourceBase.ValidateUpdateData(ctx, userCred, query, input.JointResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SJointResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func (manager *SVirtualJointResourceBaseManager) FilterById(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	return q.Equals("row_id", idStr)
}
