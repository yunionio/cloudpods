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
	"database/sql"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type DBJointModelDispatcher struct {
	DBModelDispatcher
}

func NewJointModelHandler(manager IJointModelManager) *DBJointModelDispatcher {
	// registerModelManager(manager)
	return &DBJointModelDispatcher{DBModelDispatcher: DBModelDispatcher{modelManager: manager}}
}

func (dispatcher *DBJointModelDispatcher) JointModelManager() IJointModelManager {
	return dispatcher.modelManager.(IJointModelManager)
}

func (dispatcher *DBJointModelDispatcher) MasterKeywordPlural() string {
	jointManager := dispatcher.JointModelManager()
	if jointManager == nil {
		log.Fatalf("nil jointModelManager")
	}
	return jointManager.GetMasterManager().KeywordPlural()
}

func (dispatcher *DBJointModelDispatcher) SlaveKeywordPlural() string {
	jointManager := dispatcher.JointModelManager()
	if jointManager == nil {
		log.Fatalf("nil jointModelManager")
	}
	return jointManager.GetSlaveManager().KeywordPlural()
}

func (dispatcher *DBJointModelDispatcher) ListMasterDescendent(ctx context.Context, idStr string, query jsonutils.JSONObject) (*modules.ListResult, error) {
	//log.Debugf("ListMasterDescendent %s %s", dispatcher.JointModelManager().GetMasterManager().Keyword(), idStr)
	userCred := fetchUserCredential(ctx)

	var queryDict *jsonutils.JSONDict
	if query != nil {
		queryDict, _ = query.(*jsonutils.JSONDict)
		if queryDict == nil {
			return nil, fmt.Errorf("fail to convert query to dict")
		}
	}

	model, err := fetchItem(dispatcher.JointModelManager().GetMasterManager(), ctx, userCred, idStr, query)
	if err == sql.ErrNoRows {
		return nil, httperrors.NewResourceNotFoundError2(dispatcher.JointModelManager().GetMasterManager().Keyword(), idStr)
	} else if err != nil {
		return nil, err
	}
	queryDict.Add(jsonutils.NewString(model.GetId()), fmt.Sprintf("%s_id", dispatcher.JointModelManager().GetMasterManager().Keyword()))
	if len(dispatcher.JointModelManager().GetMasterManager().Alias()) > 0 {
		queryDict.Add(jsonutils.NewString(model.GetId()), fmt.Sprintf("%s_id", dispatcher.JointModelManager().GetMasterManager().Alias()))
	}

	return dispatcher._listJoint(ctx, userCred, model.(IStandaloneModel), queryDict)
}

func (dispatcher *DBJointModelDispatcher) ListSlaveDescendent(ctx context.Context, idStr string, query jsonutils.JSONObject) (*modules.ListResult, error) {
	//log.Debugf("ListSlaveDescendent %s %s", dispatcher.JointModelManager().GetMasterManager().Keyword(), idStr)
	userCred := fetchUserCredential(ctx)

	var queryDict *jsonutils.JSONDict
	if query != nil {
		queryDict, _ = query.(*jsonutils.JSONDict)
		if queryDict == nil {
			return nil, fmt.Errorf("fail to convert query to dict")
		}
	}

	model, err := fetchItem(dispatcher.JointModelManager().GetSlaveManager(), ctx, userCred, idStr, query)
	if err == sql.ErrNoRows {
		return nil, httperrors.NewResourceNotFoundError2(dispatcher.JointModelManager().GetSlaveManager().Keyword(), idStr)
	} else if err != nil {
		return nil, err
	}
	queryDict.Add(jsonutils.NewString(model.GetId()), fmt.Sprintf("%s_id", dispatcher.JointModelManager().GetSlaveManager().Keyword()))
	if len(dispatcher.JointModelManager().GetSlaveManager().Alias()) > 0 {
		queryDict.Add(jsonutils.NewString(model.GetId()), fmt.Sprintf("%s_id", dispatcher.JointModelManager().GetSlaveManager().Alias()))
	}

	return dispatcher._listJoint(ctx, userCred, model.(IStandaloneModel), queryDict)
}

func (dispatcher *DBJointModelDispatcher) _listJoint(ctx context.Context, userCred mcclient.TokenCredential, ctxModel IModel, queryDict jsonutils.JSONObject) (*modules.ListResult, error) {
	var isAllow bool
	if consts.IsRbacEnabled() {
		isAdmin := jsonutils.QueryBoolean(queryDict, "admin", false)
		isAllow = isJointListRbacAllowed(dispatcher.JointModelManager(), userCred, isAdmin)
	} else {
		isAllow = dispatcher.JointModelManager().AllowListDescendent(ctx, userCred, ctxModel, queryDict)
	}
	if !isAllow {
		return nil, httperrors.NewForbiddenError("Not allow to list")
	}

	items, err := ListItems(dispatcher.JointModelManager(), ctx, userCred, queryDict, "")
	if err != nil {
		log.Errorf("Fail to list items: %s", err)
		return nil, httperrors.NewGeneralError(err)
	}
	return items, nil
}

func fetchJointItem(dispatcher *DBJointModelDispatcher, ctx context.Context, userCred mcclient.TokenCredential, id1 string, id2 string, query jsonutils.JSONObject) (IStandaloneModel, IStandaloneModel, IJointModel, error) {
	master, err := fetchItem(dispatcher.JointModelManager().GetMasterManager(), ctx, userCred, id1, query)
	if err != nil {
		return nil, nil, nil, httperrors.NewGeneralError(err)
	}
	slave, err := fetchItem(dispatcher.JointModelManager().GetSlaveManager(), ctx, userCred, id2, query)
	if err != nil {
		return nil, nil, nil, httperrors.NewGeneralError(err)
	}
	item, err := FetchJointByIds(dispatcher.JointModelManager(), master.GetId(), slave.GetId(), query)
	if err != nil {
		return nil, nil, nil, err
	}
	return master.(IStandaloneModel), slave.(IStandaloneModel), item, nil
}

func (dispatcher *DBJointModelDispatcher) Get(ctx context.Context, id1 string, id2 string, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	userCred := fetchUserCredential(ctx)
	_, _, item, err := fetchJointItem(dispatcher, ctx, userCred, id1, id2, query)
	if err == sql.ErrNoRows {
		return nil, httperrors.NewResourceNotFoundError2(dispatcher.modelManager.Keyword(), id1+"-"+id2)
	} else if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	var isAllow bool
	if consts.IsRbacEnabled() {
		isAllow = isJointObjectRbacAllowed(dispatcher.JointModelManager(), item, userCred, policy.PolicyActionGet)
	} else {
		isAllow = item.AllowGetJointDetails(ctx, userCred, query, item)
	}
	if !isAllow {
		return nil, httperrors.NewForbiddenError("Not allow to get details")
	}
	return getItemDetails(dispatcher.JointModelManager(), item, ctx, userCred, query)
}

func attachItems(dispatcher *DBJointModelDispatcher, master IStandaloneModel, slave IStandaloneModel, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	var isAllow bool
	if consts.IsRbacEnabled() {
		isAllow = isObjectRbacAllowed(master.GetModelManager(), master, userCred, policy.PolicyActionPerform, "attach") &&
			isObjectRbacAllowed(slave.GetModelManager(), slave, userCred, policy.PolicyActionPerform, "attach")
	} else {
		isAllow = dispatcher.JointModelManager().AllowAttach(ctx, userCred, master, slave)
	}
	if !isAllow {
		return nil, httperrors.NewForbiddenError("Not allow to attach")
	}
	ownerProjId, err := fetchOwnerProjectId(ctx, dispatcher.JointModelManager(), userCred, data)
	dataDict, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, fmt.Errorf("body not a json dict")
	}
	dataDict.Add(jsonutils.NewString(master.GetId()), fmt.Sprintf("%s_id", dispatcher.JointModelManager().GetMasterManager().Keyword()))
	if len(dispatcher.JointModelManager().GetMasterManager().Alias()) > 0 {
		dataDict.Add(jsonutils.NewString(master.GetId()), fmt.Sprintf("%s_id", dispatcher.JointModelManager().GetMasterManager().Alias()))
	}
	dataDict.Add(jsonutils.NewString(slave.GetId()), fmt.Sprintf("%s_id", dispatcher.JointModelManager().GetSlaveManager().Keyword()))
	if len(dispatcher.JointModelManager().GetSlaveManager().Alias()) > 0 {
		dataDict.Add(jsonutils.NewString(slave.GetId()), fmt.Sprintf("%s_id", dispatcher.JointModelManager().GetSlaveManager().Alias()))
	}
	item, err := doCreateItem(dispatcher.JointModelManager(), ctx, userCred, ownerProjId, query, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	item.PostCreate(ctx, userCred, ownerProjId, query, data)
	OpsLog.LogAttachEvent(ctx, master, slave, userCred, jsonutils.Marshal(item))
	dispatcher.modelManager.OnCreateComplete(ctx, []IModel{item}, userCred, query, data)
	return getItemDetails(dispatcher.JointModelManager(), item, ctx, userCred, query)
}

func (dispatcher *DBJointModelDispatcher) Attach(ctx context.Context, id1 string, id2 string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	userCred := fetchUserCredential(ctx)
	master, err := fetchItem(dispatcher.JointModelManager().GetMasterManager(), ctx, userCred, id1, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError2(dispatcher.JointModelManager().GetMasterManager().Keyword(), id1)
		} else {
			return nil, httperrors.NewGeneralError(err)
		}
	}
	slave, err := fetchItem(dispatcher.JointModelManager().GetSlaveManager(), ctx, userCred, id2, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError2(dispatcher.JointModelManager().GetSlaveManager().Keyword(), id2)
		} else {
			return nil, httperrors.NewGeneralError(err)
		}
	}

	_, _, joinItem, err := fetchJointItem(dispatcher, ctx, userCred, master.GetId(), slave.GetId(), query)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if joinItem != nil {
		return nil, httperrors.NewNotAcceptableError("Object %s %s has attached %s %s", master.KeywordPlural(), master.GetId(), slave.KeywordPlural(), slave.GetId())
	}

	lockman.LockJointObject(ctx, master, slave)
	defer lockman.ReleaseJointObject(ctx, master, slave)
	return attachItems(dispatcher, master.(IStandaloneModel), slave.(IStandaloneModel), ctx, userCred, query, data)
}

func (dispatcher *DBJointModelDispatcher) Update(ctx context.Context, id1 string, id2 string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	userCred := fetchUserCredential(ctx)
	master, slave, item, err := fetchJointItem(dispatcher, ctx, userCred, id1, id2, query)
	if err == sql.ErrNoRows {
		if jsonutils.QueryBoolean(query, "auto_create", false) {
			queryDict := query.(*jsonutils.JSONDict)
			queryDict.Remove("auto_create")
			return dispatcher.Attach(ctx, id1, id2, query, data)
		}
		return nil, httperrors.NewResourceNotFoundError2(dispatcher.modelManager.Keyword(), id1+"-"+id2)
	} else if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	var isAllow bool
	if consts.IsRbacEnabled() {
		isAllow = isJointObjectRbacAllowed(dispatcher.JointModelManager(), item, userCred, policy.PolicyActionUpdate)
	} else {
		isAllow = item.AllowUpdateJointItem(ctx, userCred, item)
	}
	if !isAllow {
		return nil, httperrors.NewForbiddenError("Not allow to update item")
	}

	lockman.LockJointObject(ctx, master, slave)
	defer lockman.ReleaseJointObject(ctx, master, slave)
	return updateItem(dispatcher.JointModelManager(), item, ctx, userCred, query, data)
}

func (dispatcher *DBJointModelDispatcher) Detach(ctx context.Context, id1 string, id2 string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	userCred := fetchUserCredential(ctx)
	master, slave, item, err := fetchJointItem(dispatcher, ctx, userCred, id1, id2, query)
	if err == sql.ErrNoRows {
		return nil, httperrors.NewResourceNotFoundError2(dispatcher.modelManager.Keyword(), id1+"-"+id2)
	} else if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	lockman.LockJointObject(ctx, master, slave)
	defer lockman.ReleaseJointObject(ctx, master, slave)
	obj, err := deleteItem(dispatcher.JointModelManager(), item, ctx, userCred, query, data)
	if err == nil {
		OpsLog.LogDetachEvent(ctx, item.Master(), item.Slave(), userCred, jsonutils.Marshal(item))
	}
	return obj, err
}

func DetachJoint(ctx context.Context, userCred mcclient.TokenCredential, item IJointModel) error {
	err := item.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}
	err = item.Delete(ctx, userCred)
	if err == nil {
		OpsLog.LogDetachEvent(ctx, item.Master(), item.Slave(), userCred, item.GetShortDesc(ctx))
	}
	return err
}
