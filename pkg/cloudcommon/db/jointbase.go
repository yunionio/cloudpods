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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SJointResourceBase struct {
	SResourceBase

	RowId int64 `primary:"true" auto_increment:"true" list:"user"`
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

func (manager *SJointResourceBaseManager) CreateByInsertOrUpdate() bool {
	return false
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

func JointModelExtra(jointModel IJointModel) (string, string) {
	masterName, slaveName := "", ""
	master := JointMaster(jointModel)
	if master != nil {
		masterName = master.GetName()
	}
	slave := JointSlave(jointModel)
	if slave != nil {
		slaveName = slave.GetName()
	}
	return masterName, slaveName
}

func (joint *SJointResourceBase) GetJointModelManager() IJointModelManager {
	return joint.SResourceBase.GetModelManager().(IJointModelManager)
}

func getFieldValue(joint IJointModel, keyword string, alias string, fieldIdKey string) string {
	jointValue := reflect.Indirect(reflect.ValueOf(joint))
	for _, valKey := range []string{
		fmt.Sprintf("%s_id", keyword),
		fmt.Sprintf("%s_id", alias),
		fieldIdKey,
	} {
		idStr, ok := reflectutils.FindStructFieldInterface(jointValue, valKey)
		if ok {
			return idStr.(string)
		}
	}
	return ""
}

func JointMasterID(joint IJointModel) string {
	jointMan := joint.GetJointModelManager()
	masterMan := jointMan.GetMasterManager()
	return getFieldValue(joint, masterMan.Keyword(), masterMan.Alias(), jointMan.GetMasterFieldName())
}

func JointSlaveID(joint IJointModel) string {
	jointMan := joint.GetJointModelManager()
	slaveMan := jointMan.GetSlaveManager()
	return getFieldValue(joint, slaveMan.Keyword(), slaveMan.Alias(), jointMan.GetSlaveFieldName())
}

func JointMaster(joint IJointModel) IStandaloneModel {
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

func JointSlave(joint IJointModel) IStandaloneModel {
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

func (manager *SJointResourceBaseManager) ResourceScope() rbacscope.TRbacScope {
	return manager.GetMasterManager().ResourceScope()
}

func (manager *SJointResourceBaseManager) NamespaceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeSystem
}

func (manager *SJointResourceBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input apis.JoinResourceBaseCreateInput) (apis.JoinResourceBaseCreateInput, error) {
	var err error
	input.ResourceBaseCreateInput, err = manager.SResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.ResourceBaseCreateInput)
	if err != nil {
		return input, err
	}
	return input, nil
}

func (manager *SJointResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.JointResourceBaseDetails {
	ret := make([]apis.JointResourceBaseDetails, len(objs))
	upperRet := manager.SResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range objs {
		ret[i] = apis.JointResourceBaseDetails{
			ResourceBaseDetails: upperRet[i],
		}
	}
	return ret
}

func (manager *SJointResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.JointResourceBaseListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (manager *SJointResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.JointResourceBaseListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (model *SJointResourceBase) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.JointResourceBaseUpdateInput,
) (apis.JointResourceBaseUpdateInput, error) {
	var err error
	input.ResourceBaseUpdateInput, err = model.SResourceBase.ValidateUpdateData(ctx, userCred, query, input.ResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SResourceBase.ValidateUpdateData")
	}
	return input, nil
}
