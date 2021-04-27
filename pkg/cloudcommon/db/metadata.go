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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	dbapi "yunion.io/x/onecloud/pkg/apis/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

const (
	SYSTEM_ADMIN_PREFIX  = "__sys_"
	SYS_TAG_PREFIX       = "__"
	CLOUD_TAG_PREFIX     = dbapi.CLOUD_TAG_PREFIX
	USER_TAG_PREFIX      = dbapi.USER_TAG_PREFIX
	SYS_CLOUD_TAG_PREFIX = dbapi.SYS_CLOUD_TAG_PREFIX

	// TAG_DELETE_RANGE_USER  = "user"
	// TAG_DELETE_RANGE_CLOUD = CLOUD_TAG_PREFIX // "cloud"

	TAG_DELETE_RANGE_ALL = "all"

	OBJECT_TYPE_ID_SEP = "::"
)

type SMetadataManager struct {
	SModelBaseManager
}

type SMetadata struct {
	SModelBase

	// 资源类型
	// example: network
	ObjType string `width:"40" charset:"ascii" index:"true" list:"user" get:"user"`

	// 资源ID
	// example: 87321a70-1ecb-422a-8b0c-c9aa632a46a7
	ObjId string `width:"88" charset:"ascii" index:"true" list:"user" get:"user"`

	// 资源组合ID
	// example: network::87321a70-1ecb-422a-8b0c-c9aa632a46a7
	Id string `width:"128" charset:"ascii" primary:"true" list:"user" get:"user"`

	// 标签KEY
	// exmaple: 部门
	Key string `width:"64" charset:"utf8" primary:"true" list:"user" get:"user"`

	// 标签值
	// example: 技术部
	Value string `charset:"utf8" list:"user" get:"user"`

	// 更新时间
	UpdatedAt time.Time `nullable:"false" updated_at:"true"`

	// 是否被删除
	Deleted bool `nullable:"false" default:"false" index:"true"`
}

var Metadata *SMetadataManager

func init() {
	Metadata = &SMetadataManager{
		SModelBaseManager: NewModelBaseManager(
			SMetadata{},
			"metadata_tbl",
			"metadata",
			"metadatas",
		),
	}
	Metadata.SetVirtualObject(Metadata)
}

func (manager *SMetadataManager) InitializeData() error {
	q := manager.RawQuery()
	q = q.Filter(sqlchemy.OR(
		sqlchemy.IsNullOrEmpty(q.Field("obj_type")),
		sqlchemy.IsNullOrEmpty(q.Field("obj_id")),
	))
	mds := make([]SMetadata, 0)
	err := FetchModelObjects(manager, q, &mds)
	if err != nil && err != sql.ErrNoRows {
		return errors.Wrap(err, "FetchModelObjects")
	}
	for i := range mds {
		_, err := Update(&mds[i], func() error {
			parts := strings.Split(mds[i].Id, OBJECT_TYPE_ID_SEP)
			if len(parts) == 2 {
				mds[i].ObjType = parts[0]
				mds[i].ObjId = parts[1]
				return nil
			} else {
				return errors.Wrapf(httperrors.ErrInvalidFormat, "invlid id format %s", mds[i].Id)
			}
		})
		if err != nil {
			return errors.Wrap(err, "update")
		}
	}
	return nil
}

func (m *SMetadata) GetId() string {
	return fmt.Sprintf("%s-%s", m.Id, m.Key)
}

func (m *SMetadata) GetName() string {
	return fmt.Sprintf("%s-%s", m.Id, m.Key)
}

func (m *SMetadata) GetModelManager() IModelManager {
	return Metadata
}

func GetObjectIdstr(model IModel) string {
	return fmt.Sprintf("%s%s%s", model.GetModelManager().Keyword(), OBJECT_TYPE_ID_SEP, model.GetId())
}

func (manager *SMetadataManager) Query(fields ...string) *sqlchemy.SQuery {
	return manager.SModelBaseManager.Query(fields...).IsFalse("deleted")
}

func (manager *SMetadataManager) RawQuery(fields ...string) *sqlchemy.SQuery {
	return manager.SModelBaseManager.Query(fields...)
}

func (m *SMetadata) MarkDelete() error {
	m.Deleted = true
	return nil
}

func (m *SMetadata) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return DeleteModel(ctx, userCred, m)
}

func (manager *SMetadataManager) AllowGetPropertyTagValuePairs(ctx context.Context, userCred mcclient.TokenCredential, input apis.MetadataListInput) bool {
	return true
}

func (manager *SMetadataManager) GetPropertyTagValuePairs(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input apis.MetadataListInput,
) (*modulebase.ListResult, error) {
	var err error
	sq := manager.Query().SubQuery()
	q := sq.Query(sq.Field("key"), sq.Field("value"), sqlchemy.COUNT("count", sq.Field("key")))

	q, err = manager.ListItemFilter(ctx, q, userCred, input)
	if err != nil {
		return nil, errors.Wrap(err, "ListItemFilter")
	}

	q = q.GroupBy(q.Field("key"), q.Field("value"))
	if input.Order == string(sqlchemy.SQL_ORDER_DESC) {
		q = q.Desc(q.Field("key")).Desc(q.Field("value"))
	} else {
		q = q.Asc(q.Field("key")).Asc(q.Field("value"))
	}

	totalCnt, err := q.CountWithError()
	if err != nil {
		return nil, errors.Wrap(err, "CountWithError")
	}

	if totalCnt == 0 {
		emptyList := modulebase.ListResult{Data: []jsonutils.JSONObject{}}
		return &emptyList, nil
	}

	var maxLimit int64 = consts.GetMaxPagingLimit()
	limit := consts.GetDefaultPagingLimit()
	if input.Limit != nil {
		limit = int64(*input.Limit)
	}
	offset := int64(0)
	if input.Offset != nil {
		offset = int64(*input.Offset)
	}
	if offset < 0 {
		offset = int64(totalCnt) + offset
	}
	if offset < 0 {
		offset = 0
	}
	if int64(totalCnt) > maxLimit && (limit <= 0 || limit > maxLimit) {
		limit = maxLimit
	}
	if limit > 0 {
		q = q.Limit(int(limit))
	}
	if offset > 0 {
		q = q.Offset(int(offset))
	}

	data, err := manager.metaDataQuery2List(ctx, q, userCred, input)
	if err != nil {
		return nil, errors.Wrap(err, "metadataQuery2List")
	}
	emptyList := modulebase.ListResult{
		Data:   data,
		Total:  totalCnt,
		Limit:  int(limit),
		Offset: int(offset),
	}
	return &emptyList, nil
}

func (manager *SMetadataManager) metaDataQuery2List(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input apis.MetadataListInput) ([]jsonutils.JSONObject, error) {
	metadatas := make([]struct {
		Key   string
		Value string
		Count int64
	}, 0)
	err := q.All(&metadatas)
	if err != nil {
		return nil, errors.Wrap(err, "Query.All")
	}
	ciMap := map[string]string{}

	ret := make([]jsonutils.JSONObject, len(metadatas))
	for i := range metadatas {
		if k, ok := ciMap[strings.ToLower(metadatas[i].Key)]; !ok {
			ciMap[strings.ToLower(metadatas[i].Key)] = metadatas[i].Key
		} else {
			metadatas[i].Key = k
		}
		if input.Details != nil && *input.Details {
			ret[i], err = manager.getKeyValueObjectCount(ctx, userCred, input, metadatas[i].Key, metadatas[i].Value, metadatas[i].Count)
			if err != nil {
				return nil, errors.Wrap(err, "getKeyValueObjectCount")
			}
		} else {
			ret[i] = jsonutils.Marshal(metadatas[i])
		}
	}

	return ret, nil
}

func (manager *SMetadataManager) getKeyValueObjectCount(ctx context.Context, userCred mcclient.TokenCredential, input apis.MetadataListInput, key string, value string, count int64) (jsonutils.JSONObject, error) {
	metadatas := manager.Query().SubQuery()
	q := metadatas.Query(metadatas.Field("obj_type"), sqlchemy.COUNT("obj_count"))
	q, err := manager.ListItemFilter(ctx, q, userCred, input)
	if err != nil {
		return nil, errors.Wrap(err, "ListItemFilter")
	}
	q = q.Equals("key", key)
	if len(value) > 0 {
		q = q.Equals("value", value)
	} else {
		q = q.IsNullOrEmpty("value")
	}
	q = q.GroupBy("key", "value", "obj_type")

	objectCount := make([]struct {
		ObjType  string
		ObjCount int64
	}, 0)
	err = q.All(&objectCount)
	if err != nil {
		return nil, errors.Wrap(err, "query.All")
	}

	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(key), "key")
	data.Add(jsonutils.NewString(value), "value")
	data.Add(jsonutils.NewInt(count), "count")
	for _, oc := range objectCount {
		data.Add(jsonutils.NewInt(oc.ObjCount), fmt.Sprintf("%s_count", oc.ObjType))
	}

	return data, nil
}

func (manager *SMetadataManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

// 元数据(标签)列表
func (manager *SMetadataManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input apis.MetadataListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SModelBaseManager.ListItemFilter(ctx, q, userCred, input.ModelBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SModelBaseManager.ListItemFilter")
	}

	if len(input.Key) > 0 {
		q = q.In("key", input.Key)
	}
	if len(input.Value) > 0 {
		q = q.In("value", input.Value)
	}
	if len(input.Search) > 0 {
		q = q.Filter(sqlchemy.OR(
			sqlchemy.Contains(q.Field("key"), input.Search),
			sqlchemy.Contains(q.Field("value"), input.Search),
		))
	}

	resources := input.Resources
	if len(resources) == 0 {
		for resource := range globalTables {
			resources = append(resources, resource)
		}
	}
	conditions := []sqlchemy.ICondition{}
	for _, resource := range resources {
		man, ok := globalTables[resource]
		if !ok {
			return nil, httperrors.NewInputParameterError("Not support resource %s tag filter", resource)
		}
		if !man.IsStandaloneManager() {
			continue
		}
		sq := man.Query("id")
		query := jsonutils.Marshal(input)
		ownerId, queryScope, err := FetchCheckQueryOwnerScope(ctx, userCred, query, man, policy.PolicyActionList, true)
		if err != nil {
			log.Warningf("FetchCheckQueryOwnerScope.%s error: %v", man.Keyword(), err)
			continue
		}
		sq = man.FilterByOwner(sq, ownerId, queryScope)
		sq = man.FilterBySystemAttributes(sq, userCred, query, queryScope)
		sq = man.FilterByHiddenSystemAttributes(sq, userCred, query, queryScope)
		conditions = append(conditions, sqlchemy.In(q.Field("obj_id"), sq))
	}
	if len(conditions) > 0 {
		q = q.Filter(sqlchemy.OR(conditions...))
	}

	if input.SysMeta != nil && *input.SysMeta {
		q = q.Filter(sqlchemy.Startswith(q.Field("key"), SYS_TAG_PREFIX))
	}
	if input.CloudMeta != nil && *input.CloudMeta {
		q = q.Filter(sqlchemy.Startswith(q.Field("key"), CLOUD_TAG_PREFIX))
	}
	if input.UserMeta != nil && *input.UserMeta {
		q = q.Filter(sqlchemy.Startswith(q.Field("key"), USER_TAG_PREFIX))
	}

	/*for args, prefix := range map[string]string{"sys_meta": SYS_TAG_PREFIX, "cloud_meta": CLOUD_TAG_PREFIX, "user_meta": USER_TAG_PREFIX} {
		if jsonutils.QueryBoolean(query, args, false) {
			q = q.Filter(sqlchemy.Startswith(q.Field("key"), prefix))
		}
	}*/

	withConditions := []sqlchemy.ICondition{}
	if input.WithSysMeta != nil && *input.WithSysMeta {
		withConditions = append(withConditions, sqlchemy.Startswith(q.Field("key"), SYS_TAG_PREFIX))
	}
	if input.WithCloudMeta != nil && *input.WithCloudMeta {
		withConditions = append(withConditions, sqlchemy.Startswith(q.Field("key"), CLOUD_TAG_PREFIX))
	}
	if input.WithUserMeta != nil && *input.WithUserMeta {
		withConditions = append(withConditions, sqlchemy.Startswith(q.Field("key"), USER_TAG_PREFIX))
	}

	/*for args, prefix := range map[string]string{"with_sys_meta": SYS_TAG_PREFIX, "with_cloud_meta": CLOUD_TAG_PREFIX, "with_user_meta": USER_TAG_PREFIX} {
		if jsonutils.QueryBoolean(query, args, false) {
			withConditions = append(withConditions, sqlchemy.Startswith(q.Field("key"), prefix))
		}
	}*/

	if len(withConditions) > 0 {
		q = q.Filter(sqlchemy.OR(withConditions...))
	}

	return q, nil
}

func (manager *SMetadataManager) GetStringValue(model IModel, key string, userCred mcclient.TokenCredential) string {
	if strings.HasPrefix(key, SYSTEM_ADMIN_PREFIX) && (userCred == nil || !IsAllowGetSpec(rbacutils.ScopeSystem, userCred, model, "metadata")) {
		return ""
	}
	idStr := GetObjectIdstr(model)
	m := SMetadata{}
	err := manager.Query().Equals("id", idStr).Equals("key", key).First(&m)
	if err == nil {
		return m.Value
	}
	return ""
}

func (manager *SMetadataManager) GetJsonValue(model IModel, key string, userCred mcclient.TokenCredential) jsonutils.JSONObject {
	if strings.HasPrefix(key, SYSTEM_ADMIN_PREFIX) && (userCred == nil || !IsAllowGetSpec(rbacutils.ScopeSystem, userCred, model, "metadata")) {
		return nil
	}
	idStr := GetObjectIdstr(model)
	m := SMetadata{}
	err := manager.Query().Equals("id", idStr).Equals("key", key).First(&m)
	if err == nil {
		json, _ := jsonutils.ParseString(m.Value)
		return json
	}
	return nil
}

type sMetadataChange struct {
	Key    string
	OValue string
	NValue string `json:",allowempty"`
}

func (manager *SMetadataManager) RemoveAll(ctx context.Context, model IModel, userCred mcclient.TokenCredential) error {
	idStr := GetObjectIdstr(model)
	if len(idStr) == 0 {
		return fmt.Errorf("invalid model")
	}

	lockman.LockObject(ctx, model)
	defer lockman.ReleaseObject(ctx, model)

	changes := []sMetadataChange{}
	records := make([]SMetadata, 0)
	q := manager.Query().Equals("id", idStr)
	err := FetchModelObjects(manager, q, &records)
	if err != nil {
		return fmt.Errorf("find metadata for %s fail: %s", idStr, err)
	}
	for _, rec := range records {
		if err = rec.Delete(ctx, userCred); err != nil {
			log.Errorf("remove metadata %v error: %v", rec, err)
			continue
		}
		changes = append(changes, sMetadataChange{Key: rec.Key, OValue: rec.Value})
	}
	if len(changes) > 0 {
		OpsLog.LogEvent(model, ACT_SET_METADATA, jsonutils.Marshal(changes), userCred)
	}
	return nil
}

func (manager *SMetadataManager) SetValue(ctx context.Context, obj IModel, key string, value interface{}, userCred mcclient.TokenCredential) error {
	return manager.SetValuesWithLog(ctx, obj, map[string]interface{}{key: value}, userCred)
}

func (manager *SMetadataManager) SetValuesWithLog(ctx context.Context, obj IModel, store map[string]interface{}, userCred mcclient.TokenCredential) error {
	changes, err := manager.setValues(ctx, obj, store, userCred)
	if err != nil {
		return err
	}
	if len(changes) > 0 {
		OpsLog.LogEvent(obj.GetIModel(), ACT_SET_METADATA, jsonutils.Marshal(changes), userCred)
	}
	return nil
}

func (manager *SMetadataManager) setValues(ctx context.Context, obj IModel, store map[string]interface{}, userCred mcclient.TokenCredential) ([]sMetadataChange, error) {
	idStr := GetObjectIdstr(obj)

	// no need to lock
	// lockman.LockObject(ctx, obj)
	// defer lockman.ReleaseObject(ctx, obj)

	changes := make([]sMetadataChange, 0)
	for key, value := range store {

		record := SMetadata{}
		record.SetModelManager(manager, &record)

		err := manager.RawQuery().Equals("id", idStr).Equals("key", key).First(&record) //避免之前设置的tag被删除后再次设置时出现Duplicate entry error
		if err != nil {
			if errors.Cause(err) != sql.ErrNoRows {
				return changes, errors.Wrap(err, "RawQuery")
			} else {
				record.Deleted = true
			}
		}

		newRecord := SMetadata{}
		newRecord.SetModelManager(manager, &newRecord)

		newRecord.ObjId = obj.GetId()
		newRecord.ObjType = obj.GetModelManager().Keyword()
		newRecord.Id = idStr
		newRecord.Key = key

		valStr := stringutils.Interface2String(value)
		valStrLower := strings.ToLower(valStr)
		if valStrLower == "none" || valStrLower == "null" {
			newRecord.Value = record.Value
			newRecord.Deleted = true
		} else {
			newRecord.Value = valStr
			newRecord.Deleted = false
		}

		if record.Deleted == newRecord.Deleted && record.Value == newRecord.Value {
			// no changes
			continue
		}

		if len(record.Id) == 0 {
			err = manager.TableSpec().InsertOrUpdate(ctx, &newRecord)
		} else {
			rV, rD := record.Value, record.Deleted
			_, err = Update(&record, func() error {
				record.Value = newRecord.Value
				record.Deleted = newRecord.Deleted
				return nil
			})
			record.Value, record.Deleted = rV, rD
		}
		if err != nil {
			return nil, errors.Wrapf(err, "InsertOrUpdate %s=%s", key, valStr)
		}

		if record.Deleted != newRecord.Deleted {
			if record.Deleted {
				// create
				changes = append(changes, sMetadataChange{Key: key, NValue: valStr})
			} else {
				// delete
				changes = append(changes, sMetadataChange{Key: key, OValue: record.Value})
			}
		} else {
			// change
			changes = append(changes, sMetadataChange{Key: key, OValue: record.Value, NValue: valStr})
		}
	}
	return changes, nil
}

func (manager *SMetadataManager) SetAll(ctx context.Context, obj IModel, store map[string]interface{}, userCred mcclient.TokenCredential, delRange string) error {
	changes, err := manager.setValues(ctx, obj, store, userCred)
	if err != nil {
		return errors.Wrap(err, "setValues")
	}

	idStr := GetObjectIdstr(obj)

	lockman.LockObject(ctx, obj)
	defer lockman.ReleaseObject(ctx, obj)

	keys := []string{}
	for key := range store {
		keys = append(keys, key)
	}

	records := []SMetadata{}
	q := manager.Query().Equals("id", idStr).NotLike("key", `\_\_%`) //避免删除系统内置的metadata, _ 在mysql里面有特殊含义,需要转义
	switch delRange {
	case USER_TAG_PREFIX:
		q = q.Startswith("key", USER_TAG_PREFIX)
	case CLOUD_TAG_PREFIX:
		q = q.Startswith("key", CLOUD_TAG_PREFIX)
	case SYS_CLOUD_TAG_PREFIX:
		q = q.Startswith("key", SYS_CLOUD_TAG_PREFIX)
	}
	q = q.Filter(sqlchemy.NOT(sqlchemy.In(q.Field("key"), keys)))
	if err := FetchModelObjects(manager, q, &records); err != nil {
		log.Errorf("failed to fetch metadata error: %v", err)
	}
	for _, rec := range records {
		if err := rec.Delete(ctx, userCred); err != nil {
			log.Errorf("failed to delete metadata error: %v", err)
			continue
		}
		changes = append(changes, sMetadataChange{Key: rec.Key, OValue: rec.Value})
	}
	if len(changes) > 0 {
		OpsLog.LogEvent(obj.GetIModel(), ACT_SET_METADATA, jsonutils.Marshal(changes), userCred)
	}
	return nil
}

func (manager *SMetadataManager) GetAll(obj IModel, keys []string, keyPrefix string, userCred mcclient.TokenCredential) (map[string]string, error) {
	idStr := GetObjectIdstr(obj)
	records := make([]SMetadata, 0)
	q := manager.Query().Equals("id", idStr)
	if keys != nil && len(keys) > 0 {
		q = q.In("key", keys)
	}
	if len(keyPrefix) > 0 {
		q = q.Startswith("key", keyPrefix)
	}
	err := FetchModelObjects(manager, q, &records)
	if err != nil {
		return nil, err
	}
	ret := make(map[string]string)
	for _, rec := range records {
		if len(rec.Value) > 0 || strings.HasPrefix(rec.Key, USER_TAG_PREFIX) || strings.HasPrefix(rec.Key, CLOUD_TAG_PREFIX) {
			ret[strings.ToLower(rec.Key)] = rec.Value
		}
	}
	return ret, nil
}

func (manager *SMetadataManager) IsSystemAdminKey(key string) bool {
	return IsMetadataKeySystemAdmin(key)
}

func IsMetadataKeySystemAdmin(key string) bool {
	return strings.HasPrefix(key, SYSTEM_ADMIN_PREFIX)
}

func IsMetadataKeySysTag(key string) bool {
	return strings.HasPrefix(key, SYS_TAG_PREFIX)
}

func (manager *SMetadataManager) GetSysadminKey(key string) string {
	return fmt.Sprintf("%s%s", SYSTEM_ADMIN_PREFIX, key)
}

func IsMetadataKeyVisiable(key string) bool {
	return !(IsMetadataKeySysTag(key) || IsMetadataKeySystemAdmin(key))
}

func GetVisiableMetadata(model IStandaloneModel, userCred mcclient.TokenCredential) (map[string]string, error) {
	metaData, err := model.GetAllMetadata(userCred)
	if err != nil {
		return nil, err
	}
	for _, key := range model.StandaloneModelManager().GetMetadataHiddenKeys() {
		delete(metaData, key)
	}
	for key := range metaData {
		if !IsMetadataKeyVisiable(key) {
			delete(metaData, key)
		}
	}
	return metaData, nil
}

func metaList2Map(manager IMetadataBaseModelManager, userCred mcclient.TokenCredential, metaList []SMetadata) map[string]string {
	metaMap := make(map[string]string)

	hiddenKeys := manager.GetMetadataHiddenKeys()
	for _, meta := range metaList {
		if IsMetadataKeyVisiable(meta.Key) && !utils.IsInStringArray(meta.Key, hiddenKeys) {
			metaMap[meta.Key] = meta.Value
		}
	}

	return metaMap
}
