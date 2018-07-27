package db

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/mcclient"
	"github.com/yunionio/pkg/util/reflectutils"
	"github.com/yunionio/sqlchemy"
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
	return SJointResourceBaseManager{SResourceBaseManager: NewResourceBaseManager(dt, tableName, keyword, keywordPlural), _master: master, _slave: slave}
}

func (manager *SJointResourceBaseManager) GetMasterManager() IStandaloneModelManager {
	return manager._master
}

func (manager *SJointResourceBaseManager) GetSlaveManager() IStandaloneModelManager {
	return manager._slave
}

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

func (manager *SJointResourceBaseManager) FetchByIds(id1 string, id2 string) (IJointModel, error) {
	obj, err := NewModelObject(manager)
	if err != nil {
		return nil, err
	}
	jointObj, ok := obj.(IJointModel)
	if !ok {
		return nil, fmt.Errorf("FetchByIds not a IJointModel")
	}
	q := manager.Query()
	masterField := queryField(q, manager.GetMasterManager())
	if masterField == nil {
		return nil, fmt.Errorf("cannot find master id")
	}
	slaveField := queryField(q, manager.GetSlaveManager())
	if slaveField == nil {
		return nil, fmt.Errorf("cannot find slave id")
	}
	cond1 := sqlchemy.AND(sqlchemy.Equals(masterField, id1), sqlchemy.Equals(slaveField, id2))
	cond2 := sqlchemy.AND(sqlchemy.Equals(slaveField, id1), sqlchemy.Equals(masterField, id2))
	q = q.Filter(sqlchemy.OR(cond1, cond2))
	count := q.Count()
	if count > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	} else if count == 0 {
		return nil, sql.ErrNoRows
	}
	err = q.First(jointObj)
	if err != nil {
		return nil, err
	}
	return jointObj, nil
}

func (manager *SJointResourceBaseManager) AllowListDescendent(ctx context.Context, userCred mcclient.TokenCredential, model IStandaloneModel, query jsonutils.JSONObject) bool {
	return false
}

func (manager *SJointResourceBaseManager) AllowAttach(ctx context.Context, userCred mcclient.TokenCredential, master IStandaloneModel, slave IStandaloneModel) bool {
	return false
}

func JointModelExtra(jointModel IJointModel, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	master := jointModel.Master()
	extra.Add(jsonutils.NewString(master.GetName()), master.GetModelManager().Keyword())
	alias := master.GetModelManager().Alias()
	if len(alias) > 0 {
		extra.Add(jsonutils.NewString(master.GetName()), alias)
	}
	slave := jointModel.Slave()
	extra.Add(jsonutils.NewString(slave.GetName()), slave.GetModelManager().Keyword())
	alias = slave.GetModelManager().Alias()
	if len(alias) > 0 {
		extra.Add(jsonutils.NewString(slave.GetName()), alias)
	}
	return extra
}

func (joint *SJointResourceBase) GetJointModelManager() IJointModelManager {
	return joint.SResourceBase.GetModelManager().(IJointModelManager)
}

func getFieldValue(joint IJointModel, name1 string, name2 string) string {
	jointValue := reflect.Indirect(reflect.ValueOf(joint))
	idStr, find := reflectutils.FindStructFieldInterface(jointValue, fmt.Sprintf("%s_id", name1))
	if find {
		return idStr.(string)
	}
	idStr, find = reflectutils.FindStructFieldInterface(jointValue, fmt.Sprintf("%s_id", name2))
	if find {
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
	log.Debugf("MasterID: %s %s", masterId, masterMan.KeywordPlural())
	if len(masterId) > 0 {
		master, _ := masterMan.FetchById(masterId)
		return master
	}
	return nil
}

func JointSlave(joint IJointModel) IStandaloneModel { // need override
	slaveMan := joint.GetJointModelManager().GetSlaveManager()
	slaveId := JointSlaveID(joint)
	log.Debugf("SlaveID: %s %s", slaveId, slaveMan.KeywordPlural())
	if len(slaveId) > 0 {
		slave, _ := slaveMan.FetchById(slaveId)
		return slave
	}
	return nil
}

func (joint *SJointResourceBase) Master() IStandaloneModel {
	return nil
}

func (joint *SJointResourceBase) Slave() IStandaloneModel {
	return nil
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
