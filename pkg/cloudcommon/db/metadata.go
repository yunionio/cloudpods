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
	"yunion.io/x/pkg/util/printutils"
	"yunion.io/x/pkg/util/rbacscope"
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
)

const (
	SYSTEM_ADMIN_PREFIX  = "__sys_"
	SYS_TAG_PREFIX       = "__"
	CLOUD_TAG_PREFIX     = dbapi.CLOUD_TAG_PREFIX
	USER_TAG_PREFIX      = dbapi.USER_TAG_PREFIX
	SYS_CLOUD_TAG_PREFIX = dbapi.SYS_CLOUD_TAG_PREFIX
	CLASS_TAG_PREFIX     = dbapi.CLASS_TAG_PREFIX
	SKU_METADAT_KEY      = "md5"

	ORGANIZATION_TAG_PREFIX = dbapi.ORGANIZATION_TAG_PREFIX

	// TAG_DELETE_RANGE_USER  = "user"
	// TAG_DELETE_RANGE_CLOUD = CLOUD_TAG_PREFIX // "cloud"

	TAG_DELETE_RANGE_ALL = "all"

	OBJECT_TYPE_ID_SEP = "::"

	RE_BILLING_AT = "__re_billing_at"
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
	Metadata.TableSpec().AddIndex(false, "obj_type", "obj_id", "key", "deleted")
}

func (manager *SMetadataManager) InitializeData() error {
	/*no need to do this initilization any more
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
	}*/
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

func GetModelIdstr(model IModel) string {
	return getObjectIdstr(model.GetModelManager().Keyword(), model.GetId())
}

func getObjectIdstr(objType, objId string) string {
	return fmt.Sprintf("%s%s%s", objType, OBJECT_TYPE_ID_SEP, objId)
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

func (manager *SMetadataManager) fetchKeyValueQuery(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input apis.MetaGetPropertyTagValuePairsInput,
) (*sqlchemy.SQuery, error) {
	var err error
	sq := manager.Query().SubQuery()
	keyOnly := (input.KeyOnly != nil && *input.KeyOnly)
	var q *sqlchemy.SQuery
	var queryFields []sqlchemy.IQueryField
	if keyOnly {
		queryFields = []sqlchemy.IQueryField{
			sq.Field("key"),
			sqlchemy.COUNT("count", sq.Field("key")),
		}
	} else {
		queryFields = []sqlchemy.IQueryField{
			sq.Field("key"),
			sq.Field("value"),
			sqlchemy.COUNT("count", sq.Field("key")),
		}
	}
	q = sq.Query(queryFields...)

	q, err = manager.ListItemFilter(ctx, q, userCred, input.MetadataListInput)
	if err != nil {
		return nil, errors.Wrap(err, "ListItemFilter")
	}

	if keyOnly {
		q = q.GroupBy(q.Field("key"))
	} else {
		q = q.GroupBy(q.Field("key"), q.Field("value"))
	}
	if input.Order == string(sqlchemy.SQL_ORDER_DESC) {
		q = q.Desc(q.Field("key"))
		if !keyOnly {
			q = q.Desc(q.Field("value"))
		}
	} else {
		q = q.Asc(q.Field("key"))
		if !keyOnly {
			q = q.Asc(q.Field("value"))
		}
	}

	return q, nil
}

func (manager *SMetadataManager) GetPropertyTagValuePairs(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input apis.MetaGetPropertyTagValuePairsInput,
) (*printutils.ListResult, error) {
	q, err := manager.fetchKeyValueQuery(ctx, userCred, input)
	if err != nil {
		return nil, errors.Wrap(err, "fetchKeyValueQuery")
	}

	totalCnt, err := q.CountWithError()
	if err != nil {
		return nil, errors.Wrap(err, "CountWithError")
	}

	if totalCnt == 0 {
		emptyList := printutils.ListResult{Data: []jsonutils.JSONObject{}}
		return &emptyList, nil
	}

	maxLimit := consts.GetMaxPagingLimit()
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

	data, err := manager.metaDataQuery2List(ctx, q, userCred, input.MetadataListInput)
	if err != nil {
		return nil, errors.Wrap(err, "metadataQuery2List")
	}
	emptyList := printutils.ListResult{
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

	ret := make([]jsonutils.JSONObject, len(metadatas))
	keys := []string{}
	for i := range metadatas {
		if !utils.IsInStringArray(metadatas[i].Key, keys) {
			keys = append(keys, metadatas[i].Key)
		}
		ret[i] = jsonutils.Marshal(metadatas[i])
	}

	if input.Details == nil || !*input.Details {
		return ret, nil
	}
	mQ, err := manager.ListItemFilter(ctx, manager.Query(), userCred, input)
	if err != nil {
		return nil, errors.Wrap(err, "ListItemFilter")
	}
	metas := []SMetadata{}
	err = mQ.In("key", keys).All(&metas)
	if err != nil {
		return ret, errors.Wrapf(err, "q.All")
	}
	count := map[string]map[string]map[string]int64{}
	for i := range metas {
		meta := metas[i]
		_, ok := count[meta.Key]
		if !ok {
			count[meta.Key] = map[string]map[string]int64{}
		}
		_, ok = count[meta.Key][meta.Value]
		if !ok {
			count[meta.Key][meta.Value] = map[string]int64{}
		}
		k := fmt.Sprintf("%s_count", meta.ObjType)
		_, ok = count[meta.Key][meta.Value][k]
		if !ok {
			count[meta.Key][meta.Value][k] = 0
		}
		count[meta.Key][meta.Value][k] += 1
	}
	for i, meta := range metadatas {
		jsonutils.Update(ret[i], count[meta.Key][meta.Value])
	}

	return ret, nil
}

func (manager *SMetadataManager) metadataBaseFilter(q *sqlchemy.SQuery, input apis.MetadataBaseFilterInput) *sqlchemy.SQuery {
	if len(input.Key) > 0 {
		q = q.In("key", input.Key)
	}
	if len(input.Value) > 0 {
		q = q.In("value", input.Value)
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
	if len(withConditions) > 0 {
		q = q.Filter(sqlchemy.OR(withConditions...))
	}
	return q
}

// 元数据(标签)列表
func (manager *SMetadataManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input apis.MetadataListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SModelBaseManager.ListItemFilter(ctx, q, userCred, input.ModelBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SModelBaseManager.ListItemFilter")
	}

	q = manager.metadataBaseFilter(q, input.MetadataBaseFilterInput)

	if len(input.Search) > 0 {
		q = q.Filter(sqlchemy.OR(
			sqlchemy.Contains(q.Field("key"), input.Search),
			sqlchemy.Contains(q.Field("value"), input.Search),
		))
	}

	if !(input.Scope == string(rbacscope.ScopeSystem) && userCred.HasSystemAdminPrivilege()) {
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
				return nil, httperrors.NewNotFoundError("Not support resource %s tag filter", resource)
			}
			if !man.IsStandaloneManager() {
				continue
			}
			sq := man.Query("id")
			query := jsonutils.Marshal(input)
			ownerId, queryScope, err, _ := FetchCheckQueryOwnerScope(ctx, userCred, query, man, policy.PolicyActionList, true)
			if err != nil {
				log.Warningf("FetchCheckQueryOwnerScope.%s error: %v", man.Keyword(), err)
				continue
			}
			sq = man.FilterByOwner(sq, man, userCred, ownerId, queryScope)
			sq = man.FilterBySystemAttributes(sq, userCred, query, queryScope)
			sq = man.FilterByHiddenSystemAttributes(sq, userCred, query, queryScope)
			conditions = append(conditions, sqlchemy.In(q.Field("obj_id"), sq))
		}
		if len(conditions) > 0 {
			q = q.Filter(sqlchemy.OR(conditions...))
		}
	}

	/*for args, prefix := range map[string]string{"sys_meta": SYS_TAG_PREFIX, "cloud_meta": CLOUD_TAG_PREFIX, "user_meta": USER_TAG_PREFIX} {
		if jsonutils.QueryBoolean(query, args, false) {
			q = q.Filter(sqlchemy.Startswith(q.Field("key"), prefix))
		}
	}*/

	/*for args, prefix := range map[string]string{"with_sys_meta": SYS_TAG_PREFIX, "with_cloud_meta": CLOUD_TAG_PREFIX, "with_user_meta": USER_TAG_PREFIX} {
		if jsonutils.QueryBoolean(query, args, false) {
			withConditions = append(withConditions, sqlchemy.Startswith(q.Field("key"), prefix))
		}
	}*/

	return q, nil
}

func (manager *SMetadataManager) GetStringValue(ctx context.Context, model IModel, key string, userCred mcclient.TokenCredential) string {
	if !isAllowGetMetadata(ctx, model, userCred) {
		return ""
	}
	if strings.HasPrefix(key, SYSTEM_ADMIN_PREFIX) && (userCred == nil || !IsAllowGetSpec(ctx, rbacscope.ScopeSystem, userCred, model, "metadata")) {
		return ""
	}
	idStr := GetModelIdstr(model)
	m := SMetadata{}
	err := manager.Query().Equals("id", idStr).Equals("key", key).First(&m)
	if err == nil {
		return m.Value
	}
	return ""
}

func (manager *SMetadataManager) GetJsonValue(ctx context.Context, model IModel, key string, userCred mcclient.TokenCredential) jsonutils.JSONObject {
	if !isAllowGetMetadata(ctx, model, userCred) {
		return nil
	}
	if strings.HasPrefix(key, SYSTEM_ADMIN_PREFIX) && (userCred == nil || !IsAllowGetSpec(ctx, rbacscope.ScopeSystem, userCred, model, "metadata")) {
		return nil
	}
	idStr := GetModelIdstr(model)
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
	idStr := GetModelIdstr(model)
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

func infMap2StrMap(input map[string]interface{}) map[string]string {
	output := make(map[string]string)
	for k, v := range input {
		output[k] = stringutils.Interface2String(v)
	}
	return output
}

func (manager *SMetadataManager) SetValue(ctx context.Context, obj IModel, key string, value interface{}, userCred mcclient.TokenCredential) error {
	return manager.SetValuesWithLog(ctx, obj, map[string]interface{}{key: value}, userCred)
}

func (manager *SMetadataManager) SetValuesWithLog(ctx context.Context, obj IModel, store map[string]interface{}, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, obj)
	defer lockman.ReleaseObject(ctx, obj)

	changes, err := manager.rawSetValues(ctx, obj.Keyword(), obj.GetId(), infMap2StrMap(store), false, "")
	if err != nil {
		return err
	}
	if len(changes) > 0 {
		OpsLog.LogEvent(obj.GetIModel(), ACT_SET_METADATA, jsonutils.Marshal(changes), userCred)
		for _, change := range changes {
			if change.Key == RE_BILLING_AT {
				desc := obj.GetIModel().GetShortDesc(ctx)
				desc.Set("created_at", jsonutils.Marshal(change.NValue))
				OpsLog.LogEvent(obj.GetIModel(), ACT_RE_BILLING, desc, userCred)
				break
			}
		}
	}
	return nil
}

func (manager *SMetadataManager) rawSetValues(ctx context.Context, objType string, objId string, store map[string]string, replace bool, replaceRange string) ([]sMetadataChange, error) {
	idStr := getObjectIdstr(objType, objId)

	keys := make([]string, 0, len(store))
	changes := make([]sMetadataChange, 0)
	for key, value := range store {
		keys = append(keys, key)

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

		valStr := value
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

		newRecord.SetModelManager(manager, &newRecord)

		newRecord.ObjId = objId
		newRecord.ObjType = objType
		newRecord.Id = idStr
		newRecord.Key = key

		if len(record.Id) == 0 {
			err = manager.TableSpec().InsertOrUpdate(ctx, &newRecord)
		} else {
			rV, rD := record.Value, record.Deleted
			_, err = Update(&record, func() error {
				record.Value = newRecord.Value
				record.Key = key
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
	if replace {
		records := []SMetadata{}
		q := manager.Query().Equals("id", idStr).NotLike("key", `\_\_%`) //避免删除系统内置的metadata, _ 在mysql里面有特殊含义,需要转义
		// switch replaceRange {
		// case USER_TAG_PREFIX:
		// 	q = q.Startswith("key", USER_TAG_PREFIX)
		// case CLOUD_TAG_PREFIX:
		// 	q = q.Startswith("key", CLOUD_TAG_PREFIX)
		// case SYS_CLOUD_TAG_PREFIX:
		// 	q = q.Startswith("key", SYS_CLOUD_TAG_PREFIX)
		// }
		q = q.Startswith("key", replaceRange)
		q = q.Filter(sqlchemy.NOT(sqlchemy.In(q.Field("key"), keys)))
		if err := FetchModelObjects(manager, q, &records); err != nil {
			log.Errorf("failed to fetch metadata error: %v", err)
		}
		for _, rec := range records {
			_, err := Update(&rec, func() error {
				rec.Deleted = true
				return nil
			})
			if err != nil {
				log.Errorf("failed to delete metadata record %s %s %s", objType, objId, rec.Key)
			} else {
				changes = append(changes, sMetadataChange{Key: rec.Key, OValue: rec.Value})
			}
		}
	}
	return changes, nil
}

func (manager *SMetadataManager) SetAllWithoutDelelte(ctx context.Context, obj IModel, store map[string]interface{}, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, obj)
	defer lockman.ReleaseObject(ctx, obj)

	changes, err := manager.rawSetValues(ctx, obj.Keyword(), obj.GetId(), infMap2StrMap(store), false, "")
	if err != nil {
		return errors.Wrap(err, "setValues")
	}

	if len(changes) > 0 {
		OpsLog.LogEvent(obj.GetIModel(), ACT_SET_METADATA, jsonutils.Marshal(changes), userCred)
	}
	return nil
}

func (manager *SMetadataManager) SetAll(ctx context.Context, obj IModel, store map[string]interface{}, userCred mcclient.TokenCredential, delRange string) error {
	lockman.LockObject(ctx, obj)
	defer lockman.ReleaseObject(ctx, obj)

	changes, err := manager.rawSetValues(ctx, obj.Keyword(), obj.GetId(), infMap2StrMap(store), true, delRange)
	if err != nil {
		return errors.Wrap(err, "setValues")
	}

	if len(changes) > 0 {
		OpsLog.LogEvent(obj.GetIModel(), ACT_SET_METADATA, jsonutils.Marshal(changes), userCred)
	}
	return nil
}

func isAllowGetMetadata(ctx context.Context, obj IModel, userCred mcclient.TokenCredential) bool {
	if userCred != nil {
		for _, scope := range []rbacscope.TRbacScope{
			rbacscope.ScopeSystem,
			rbacscope.ScopeDomain,
			rbacscope.ScopeProject,
		} {
			if IsAllowGetSpec(ctx, scope, userCred, obj, "metadata") {
				return true
			}
		}
		return false
	}
	return true
}

type IMetadataGetter interface {
	GetId() string
	Keyword() string
}

func (manager *SMetadataManager) GetAll(ctx context.Context, obj IMetadataGetter, keys []string, keyPrefix string, userCred mcclient.TokenCredential) (map[string]string, error) {
	modelObj, isIModel := obj.(IModel)

	if isIModel && !isAllowGetMetadata(ctx, modelObj, userCred) {
		return map[string]string{}, nil
	}
	meta, err := manager.rawGetAll(obj.Keyword(), obj.GetId(), keys, keyPrefix)
	if err != nil {
		return nil, errors.Wrap(err, "rawGetAll")
	}
	ret := make(map[string]string)
	for k, v := range meta {
		if strings.HasPrefix(k, SYSTEM_ADMIN_PREFIX) && (userCred == nil || (isIModel && !IsAllowGetSpec(ctx, rbacscope.ScopeSystem, userCred, modelObj, "metadata"))) {
			continue
		}
		ret[k] = v
	}
	return ret, nil
}

func (manager *SMetadataManager) rawGetAll(objType, objId string, keys []string, keyPrefix string) (map[string]string, error) {
	idStr := getObjectIdstr(objType, objId)
	records := make([]SMetadata, 0)
	q := manager.Query().Equals("id", idStr)
	if len(keys) > 0 {
		q = q.In("key", keys)
	}
	if len(keyPrefix) > 0 {
		q = q.Startswith("key", keyPrefix)
	}
	err := FetchModelObjects(manager, q, &records)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	ret := make(map[string]string)
	for _, rec := range records {
		if len(rec.Value) > 0 || strings.HasPrefix(rec.Key, USER_TAG_PREFIX) || strings.HasPrefix(rec.Key, CLOUD_TAG_PREFIX) {
			ret[rec.Key] = rec.Value
		}
	}
	return ret, nil
}

/*func (manager *SMetadataManager) IsSystemAdminKey(key string) bool {
	return isMetadataKeySystemAdmin(key)
}*/

func isMetadataLoginKey(key string) bool {
	return strings.HasPrefix(key, "login_")
}

func isMetadataKeySystemAdmin(key string) bool {
	return strings.HasPrefix(key, SYSTEM_ADMIN_PREFIX)
}

func isMetadataKeyPrivateKey(key string) bool {
	for _, k := range []string{"admin", "project"} {
		for _, v := range []string{"ssh-private-key", "ssh-public-key"} {
			if key == fmt.Sprintf("%s-%s", k, v) {
				return true
			}
		}
	}
	return strings.HasPrefix(key, SYSTEM_ADMIN_PREFIX)
}

func isMetadataKeySysTag(key string) bool {
	return strings.HasPrefix(key, SYS_TAG_PREFIX)
}

func (manager *SMetadataManager) GetSysadminKey(key string) string {
	return fmt.Sprintf("%s%s", SYSTEM_ADMIN_PREFIX, key)
}

func IsMetadataKeyVisible(key string) bool {
	return !(isMetadataKeySysTag(key) || isMetadataKeySystemAdmin(key) || isMetadataKeyPrivateKey(key))
}

func GetVisibleMetadata(ctx context.Context, model IStandaloneModel, userCred mcclient.TokenCredential) (map[string]string, error) {
	metaData, err := model.GetAllMetadata(ctx, userCred)
	if err != nil {
		return nil, err
	}
	for _, key := range model.StandaloneModelManager().GetMetadataHiddenKeys() {
		delete(metaData, key)
	}
	for key := range metaData {
		if !IsMetadataKeyVisible(key) {
			delete(metaData, key)
		}
	}
	return metaData, nil
}

func metaList2Map(manager IMetadataBaseModelManager, userCred mcclient.TokenCredential, metaList []SMetadata) map[string]string {
	metaMap := make(map[string]string)

	hiddenKeys := manager.GetMetadataHiddenKeys()
	for _, meta := range metaList {
		if IsMetadataKeyVisible(meta.Key) && !utils.IsInStringArray(meta.Key, hiddenKeys) {
			metaMap[meta.Key] = meta.Value
		}
	}

	return metaMap
}

func CopyTags(ctx context.Context, objType string, keys1 []string, values []string, keys2 []string) error {
	return Metadata.copyTags(ctx, objType, keys1, values, keys2)
}

func (manager *SMetadataManager) copyTags(ctx context.Context, objType string, keys1 []string, values []string, keys2 []string) error {
	for i := 0; i < len(keys1) && i < len(values) && i < len(keys2); i++ {
		key1 := keys1[i]
		key2 := keys2[i]
		value := values[i]

		q := manager.Query("obj_id").Equals("obj_type", objType).Equals("key", key1).Equals("value", value)
		q2 := manager.Query("obj_id").Equals("obj_type", objType).Equals("key", key2).Equals("value", value)
		q = q.NotIn("obj_id", q2.SubQuery())

		results := []struct {
			ObjId string
		}{}
		err := q.All(&results)
		if err != nil {
			return errors.Wrapf(err, "copy key %s value %s to %s", key1, value, key2)
		}

		for _, result := range results {
			record := SMetadata{}
			record.SetModelManager(manager, &record)

			record.ObjId = result.ObjId
			record.ObjType = objType
			record.Id = getObjectIdstr(objType, result.ObjId)
			record.Key = key2
			record.Value = value

			err := manager.TableSpec().InsertOrUpdate(ctx, &record)
			if err != nil {
				return errors.Wrapf(err, "insert %s", jsonutils.Marshal(record))
			}
		}
	}
	return nil
}
