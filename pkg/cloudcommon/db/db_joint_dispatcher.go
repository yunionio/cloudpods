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
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
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

func (dispatcher *DBJointModelDispatcher) ListMasterDescendent(ctx context.Context, idStr string, query jsonutils.JSONObject) (*modulebase.ListResult, error) {
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
	queryDict.Add(jsonutils.NewString(model.GetId()), dispatcher.JointModelManager().GetMasterFieldName())
	if len(dispatcher.JointModelManager().GetMasterManager().Alias()) > 0 {
		queryDict.Add(jsonutils.NewString(model.GetId()), fmt.Sprintf("%s_id", dispatcher.JointModelManager().GetMasterManager().Alias()))
	}

	return dispatcher._listJoint(ctx, userCred, model.(IStandaloneModel), queryDict)
}

func (dispatcher *DBJointModelDispatcher) ListSlaveDescendent(ctx context.Context, idStr string, query jsonutils.JSONObject) (*modulebase.ListResult, error) {
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
	queryDict.Add(jsonutils.NewString(model.GetId()), dispatcher.JointModelManager().GetSlaveFieldName())
	if len(dispatcher.JointModelManager().GetSlaveManager().Alias()) > 0 {
		queryDict.Add(jsonutils.NewString(model.GetId()), fmt.Sprintf("%s_id", dispatcher.JointModelManager().GetSlaveManager().Alias()))
	}

	return dispatcher._listJoint(ctx, userCred, model.(IStandaloneModel), queryDict)
}

func (dispatcher *DBJointModelDispatcher) _listJoint(ctx context.Context, userCred mcclient.TokenCredential, ctxModel IStandaloneModel, queryDict jsonutils.JSONObject) (*modulebase.ListResult, error) {
	items, err := ListItems(dispatcher.JointModelManager(), ctx, userCred, queryDict, nil)
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
	if consts.IsRbacEnabled() {
		err := isJointObjectRbacAllowed(item, userCred, policy.PolicyActionGet)
		if err != nil {
			return nil, err
		}
	} else if !item.AllowGetJointDetails(ctx, userCred, query, item) {
		return nil, httperrors.NewForbiddenError("Not allow to get details")
	}
	return getItemDetails(dispatcher.JointModelManager(), item, ctx, userCred, query)
}

func attachItems(dispatcher *DBJointModelDispatcher, master IStandaloneModel, slave IStandaloneModel, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if consts.IsRbacEnabled() {
		err := isObjectRbacAllowed(master, userCred, policy.PolicyActionPerform, "attach")
		if err != nil {
			return nil, err
		}
		err = isObjectRbacAllowed(slave, userCred, policy.PolicyActionPerform, "attach")
		if err != nil {
			return nil, err
		}
	} else if !dispatcher.JointModelManager().AllowAttach(ctx, userCred, master, slave) {
		return nil, httperrors.NewForbiddenError("Not allow to attach")
	}
	// ownerProjId, err := fetchOwnerId(ctx, dispatcher.JointModelManager(), userCred, data)
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
	item, err := doCreateItem(dispatcher.JointModelManager(), ctx, userCred, nil, query, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	item.PostCreate(ctx, userCred, nil, query, data)
	OpsLog.LogAttachEvent(ctx, master, slave, userCred, jsonutils.Marshal(item))
	dispatcher.modelManager.OnCreateComplete(ctx, []IModel{item}, userCred, nil, query, data)
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

	if consts.IsRbacEnabled() {
		err := isJointObjectRbacAllowed(item, userCred, policy.PolicyActionUpdate)
		if err != nil {
			return nil, err
		}
	} else if !item.AllowUpdateJointItem(ctx, userCred, item) {
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

	if consts.IsRbacEnabled() {
		err := isObjectRbacAllowed(master, userCred, policy.PolicyActionPerform, "detach")
		if err != nil {
			return nil, err
		}
		err = isObjectRbacAllowed(slave, userCred, policy.PolicyActionPerform, "detach")
		if err != nil {
			return nil, err
		}
	} else if !item.AllowDetach(ctx, userCred, query, data) {
		return nil, httperrors.NewForbiddenError("Not allow to attach")
	}

	lockman.LockJointObject(ctx, master, slave)
	defer lockman.ReleaseJointObject(ctx, master, slave)

	obj, err := deleteItem(dispatcher.JointModelManager(), item, ctx, userCred, query, data)
	if err == nil {
		OpsLog.LogDetachEvent(ctx, JointMaster(item), JointSlave(item), userCred, jsonutils.Marshal(item))
	}
	return obj, err
}

func DetachJoint(ctx context.Context, userCred mcclient.TokenCredential, item IJointModel) error {
	err := ValidateDeleteCondition(item, ctx, nil)
	if err != nil {
		return err
	}
	err = item.Delete(ctx, userCred)
	if err == nil {
		OpsLog.LogDetachEvent(ctx, JointMaster(item), JointSlave(item), userCred, item.GetShortDesc(ctx))
	}
	return err
}
