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

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SJointResourceBase struct {
	SResourceBase

	RowId int64 `primary:"true" auto_increment:"true"`
}

type SJointResourceBaseManager struct {
	SResourceBaseManager

	_master IStandaloneModelManager
	_slave  IStandaloneModelManager
}

func NewJointResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string, master IStandaloneModelManager, slave IStandaloneModelManager) SJointResourceBaseManager {
	log.Debugf("Initialize %s", keywordPlural)
	if master == nil {
		msg := fmt.Sprintf("%s master is nil, retry initialization later...", keywordPlural)
		log.Errorf(msg)
		panic(msg)
	}
	if slave == nil {
		msg := fmt.Sprintf("%s slave is nil, retry initialization later...", keywordPlural)
		log.Errorf(msg)
		panic(msg)
	}
	return SJointResourceBaseManager{
		SResourceBaseManager: NewResourceBaseManager(dt, tableName, keyword, keywordPlural),
		_master:              master,
		_slave:               slave,
	}
}

func (manager *SJointResourceBaseManager) GetIJointModelManager() IJointModelManager {
	return manager.GetVirtualObject().(IJointModelManager)
}

func (manager *SJointResourceBaseManager) GetMasterManager() IStandaloneModelManager {
	return manager._master
}

func (manager *SJointResourceBaseManager) GetSlaveManager() IStandaloneModelManager {
	return manager._slave
}

/*
func queryField(q *sqlchemy.SQuery, manager IModelManager) sqlchemy.IQueryField {
	field := q.Field(fmt.Sprintf("%s_id", manager.Keyword()))
	if field == nil && len(manager.Alias()) > 0 {
		field = q.Field(fmt.Sprintf("%s_id", manager.Alias()))
	}
	return field
}

func (manager *SJointResourceBaseManager) MasterField(q *sqlchemy.SQuery) sqlchemy.IQueryField {
	return queryField(q, manager.GetMasterManager())
}

func (manager *SJointResourceBaseManager) SlaveField(q *sqlchemy.SQuery) sqlchemy.IQueryField {
	return queryField(q, manager.GetSlaveManager())
}
*/

func (manager *SJointResourceBaseManager) FilterByParams(q *sqlchemy.SQuery, params jsonutils.JSONObject) *sqlchemy.SQuery {
	return q
}

func (manager *SJointResourceBaseManager) AllowListDescendent(ctx context.Context, userCred mcclient.TokenCredential, model IStandaloneModel, query jsonutils.JSONObject) bool {
	return IsAllowList(rbacutils.ScopeSystem, userCred, manager)
}

func (manager *SJointResourceBaseManager) AllowAttach(ctx context.Context, userCred mcclient.TokenCredential, master IStandaloneModel, slave IStandaloneModel) bool {
	return IsAllowCreate(rbacutils.ScopeSystem, userCred, manager)
}

func JointModelExtra(jointModel IJointModel, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	master := jointModel.Master()
	if master != nil {
		extra.Add(jsonutils.NewString(master.GetName()), master.GetModelManager().Keyword())
		alias := master.GetModelManager().Alias()
		if len(alias) > 0 {
			extra.Add(jsonutils.NewString(master.GetName()), alias)
		}
	}
	slave := jointModel.Slave()
	if slave != nil {
		extra.Add(jsonutils.NewString(slave.GetName()), slave.GetModelManager().Keyword())
		alias := slave.GetModelManager().Alias()
		if len(alias) > 0 {
			extra.Add(jsonutils.NewString(slave.GetName()), alias)
		}
	}
	return extra
}

func (joint *SJointResourceBase) GetJointModelManager() IJointModelManager {
	return joint.SResourceBase.GetModelManager().(IJointModelManager)
}

func getFieldValue(joint IJointModel, keyword string, alias string) string {
	jointValue := reflect.Indirect(reflect.ValueOf(joint))
	idStr, ok := reflectutils.FindStructFieldInterface(jointValue, fmt.Sprintf("%s_id", keyword))
	if ok {
		return idStr.(string)
	}
	idStr, ok = reflectutils.FindStructFieldInterface(jointValue, fmt.Sprintf("%s_id", alias))
	if ok {
		return idStr.(string)
	}
	return ""
}

func JointMasterID(joint IJointModel) string { // need override
	masterMan := joint.GetJointModelManager().GetMasterManager()
	return getFieldValue(joint, masterMan.Keyword(), masterMan.Alias())
}

func JointSlaveID(joint IJointModel) string { // need override
	slaveMan := joint.GetJointModelManager().GetSlaveManager()
	return getFieldValue(joint, slaveMan.Keyword(), slaveMan.Alias())
}

func JointMaster(joint IJointModel) IStandaloneModel { // need override
	masterMan := joint.GetJointModelManager().GetMasterManager()
	masterId := JointMasterID(joint)
	//log.Debugf("MasterID: %s %s", masterId, masterMan.KeywordPlural())
	if len(masterId) > 0 {
		master, _ := masterMan.FetchById(masterId)
		if master != nil {
			return master.(IStandaloneModel)
		}
	}
	return nil
}

func JointSlave(joint IJointModel) IStandaloneModel { // need override
	slaveMan := joint.GetJointModelManager().GetSlaveManager()
	slaveId := JointSlaveID(joint)
	//log.Debugf("SlaveID: %s %s", slaveId, slaveMan.KeywordPlural())
	if len(slaveId) > 0 {
		slave, _ := slaveMan.FetchById(slaveId)
		if slave != nil {
			return slave.(IStandaloneModel)
		}
	}
	return nil
}

func (joint *SJointResourceBase) GetIJointModel() IJointModel {
	return joint.GetVirtualObject().(IJointModel)
}

func (joint *SJointResourceBase) Master() IStandaloneModel {
	return nil
}

func (joint *SJointResourceBase) Slave() IStandaloneModel {
	return nil
}

func (self *SJointResourceBase) AllowGetJointDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, item IJointModel) bool {
	master := item.Master()
	switch master.(type) {
	case IVirtualModel:
		return master.(IVirtualModel).IsOwner(userCred) || IsAllowGet(rbacutils.ScopeSystem, userCred, master)
	default: // case item implemented customized AllowGetDetails, eg hostjoints
		return item.AllowGetDetails(ctx, userCred, query)
	}
}

func (self *SJointResourceBase) AllowUpdateJointItem(ctx context.Context, userCred mcclient.TokenCredential, item IJointModel) bool {
	master := item.Master()
	switch master.(type) {
	case IVirtualModel:
		return master.(IVirtualModel).IsOwner(userCred) || IsAllowUpdate(rbacutils.ScopeSystem, userCred, master)
	default: // case item implemented customized AllowGetDetails, eg hostjoints
		return item.AllowUpdateItem(ctx, userCred)
	}
}

func (self *SJointResourceBase) AllowDetach(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

/*
func (joint *SJointResourceBase) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := joint.SResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return JointModelExtra(joint, extra)
}

func (joint *SJointResourceBase) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := joint.SResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return JointModelExtra(joint, extra)
}
*/

func (manager *SJointResourceBaseManager) ResourceScope() rbacutils.TRbacScope {
	return manager.GetMasterManager().ResourceScope()
}

func (manager *SJointResourceBaseManager) NamespaceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeSystem
}

func (manager *SJointResourceBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input apis.JoinResourceBaseCreateInput) (apis.JoinResourceBaseCreateInput, error) {
	var err error
	input.ResourceBaseCreateInput, err = manager.SResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.ResourceBaseCreateInput)
	if err != nil {
		return input, err
	}
	return input, nil
}
