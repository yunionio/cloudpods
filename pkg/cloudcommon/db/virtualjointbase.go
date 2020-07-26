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
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SVirtualJointResourceBase struct {
	SJointResourceBase
}

type SVirtualJointResourceBaseManager struct {
	SJointResourceBaseManager
}

func NewVirtualJointResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string, main IVirtualModelManager, subordinate IVirtualModelManager) SVirtualJointResourceBaseManager {
	return SVirtualJointResourceBaseManager{SJointResourceBaseManager: NewJointResourceBaseManager(dt, tableName, keyword, keywordPlural, main, subordinate)}
}

func (manager *SVirtualJointResourceBaseManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	if jsonutils.QueryBoolean(query, "admin", false) && !IsAllowList(rbacutils.ScopeSystem, userCred, manager) {
		return false
	}
	return true
	// return manager.SJointResourceBaseManager.AllowListItems(ctx, userCred, query)
}

func (manager *SVirtualJointResourceBaseManager) AllowListDescendent(ctx context.Context, userCred mcclient.TokenCredential, main IStandaloneModel, query jsonutils.JSONObject) bool {
	mainVirtual := main.(IVirtualModel)
	if mainVirtual.IsOwner(userCred) || IsAllowList(rbacutils.ScopeSystem, userCred, manager) {
		return true
	}
	return false
}

func (manager *SVirtualJointResourceBaseManager) AllowAttach(ctx context.Context, userCred mcclient.TokenCredential, main IStandaloneModel, subordinate IStandaloneModel) bool {
	mainVirtual := main.(IVirtualModel)
	subordinateVirtual := subordinate.(IVirtualModel)
	if mainVirtual.GetOwnerId() == subordinateVirtual.GetOwnerId() {
		return true
	} else {
		subordinateValue := reflect.Indirect(reflect.ValueOf(subordinateVirtual))
		val, find := reflectutils.FindStructFieldInterface(subordinateValue, "IsPublic")
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
	mainVirtual := self.Main().(IVirtualModel)
	return mainVirtual.IsOwner(userCred) || IsAllowGet(rbacutils.ScopeSystem, userCred, self)
}

func (self *SVirtualJointResourceBase) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	mainVirtual := self.Main().(IVirtualModel)
	return mainVirtual.IsOwner(userCred) || IsAllowUpdate(rbacutils.ScopeSystem, userCred, self)
}

func (manager *SVirtualJointResourceBaseManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		mainQ := manager.GetMainManager().Query("id")
		mainQ = manager.GetMainManager().FilterByOwner(mainQ, owner, scope)
		subordinateQ := manager.GetSubordinateManager().Query("id")
		subordinateQ = manager.GetSubordinateManager().FilterByOwner(subordinateQ, owner, scope)
		iManager := manager.GetIJointModelManager()
		q = q.In(iManager.GetMainFieldName(), mainQ.SubQuery())
		q = q.In(iManager.GetSubordinateFieldName(), subordinateQ.SubQuery())
	}
	return q
}

func (manager *SVirtualJointResourceBaseManager) FilterBySystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	q = manager.SJointResourceBaseManager.FilterBySystemAttributes(q, userCred, query, scope)
	mainQ := manager.GetMainManager().Query("id")
	mainQ = manager.GetMainManager().FilterBySystemAttributes(mainQ, userCred, query, scope)
	subordinateQ := manager.GetSubordinateManager().Query("id")
	subordinateQ = manager.GetSubordinateManager().FilterBySystemAttributes(subordinateQ, userCred, query, scope)
	iManager := manager.GetIJointModelManager()
	q = q.In(iManager.GetMainFieldName(), mainQ.SubQuery())
	q = q.In(iManager.GetSubordinateFieldName(), subordinateQ.SubQuery())
	return q
}

func (manager *SVirtualJointResourceBaseManager) FilterByHiddenSystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	q = manager.SJointResourceBaseManager.FilterByHiddenSystemAttributes(q, userCred, query, scope)
	mainQ := manager.GetMainManager().Query("id")
	mainQ = manager.GetMainManager().FilterByHiddenSystemAttributes(mainQ, userCred, query, scope)
	subordinateQ := manager.GetSubordinateManager().Query("id")
	subordinateQ = manager.GetSubordinateManager().FilterByHiddenSystemAttributes(subordinateQ, userCred, query, scope)
	iManager := manager.GetIJointModelManager()
	q = q.In(iManager.GetMainFieldName(), mainQ.SubQuery())
	q = q.In(iManager.GetSubordinateFieldName(), subordinateQ.SubQuery())
	return q
}

func (model *SVirtualJointResourceBase) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (apis.VirtualJointResourceBaseDetails, error) {
	return apis.VirtualJointResourceBaseDetails{}, nil
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
