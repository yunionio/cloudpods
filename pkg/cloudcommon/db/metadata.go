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
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

const (
	SYSTEM_ADMIN_PREFIX = "__sys_"
	SYS_TAG_PREFIX      = "__"
	CLOUD_TAG_PREFIX    = "ext:"
	USER_TAG_PREFIX     = "user:"

	// TAG_DELETE_RANGE_USER  = "user"
	// TAG_DELETE_RANGE_CLOUD = CLOUD_TAG_PREFIX // "cloud"

	TAG_DELETE_RANGE_ALL = "all"
)

type SMetadataManager struct {
	SModelBaseManager
}

type SMetadata struct {
	SModelBase

	Id        string    `width:"128" charset:"ascii" primary:"true" list:"user" get:"user"` // = Column(VARCHAR(128, charset='ascii'), primary_key=True)
	Key       string    `width:"64" charset:"utf8" primary:"true" list:"user" get:"user"`   // = Column(VARCHAR(64, charset='ascii'),  primary_key=True)
	Value     string    `charset:"utf8" list:"user" get:"user"`                             // = Column(TEXT(charset='utf8'), nullable=True)
	UpdatedAt time.Time `nullable:"false" updated_at:"true"`                                // = Column(DateTime, default=get_utcnow, nullable=False, onupdate=get_utcnow)
	Deleted   bool      `nullable:"false" default:"false" index:"true"`
}

var Metadata *SMetadataManager
var ResourceMap map[string]*SVirtualResourceBaseManager

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

	ResourceMap = map[string]*SVirtualResourceBaseManager{
		"disk":     {SStatusStandaloneResourceBaseManager: NewStatusStandaloneResourceBaseManager(SVirtualResourceBase{}, "disks_tbl", "disk", "disks")},
		"server":   {SStatusStandaloneResourceBaseManager: NewStatusStandaloneResourceBaseManager(SVirtualResourceBase{}, "guests_tbl", "server", "servers")},
		"eip":      {SStatusStandaloneResourceBaseManager: NewStatusStandaloneResourceBaseManager(SVirtualResourceBase{}, "elasticips_tbl", "eip", "eips")},
		"snapshot": {SStatusStandaloneResourceBaseManager: NewStatusStandaloneResourceBaseManager(SVirtualResourceBase{}, "snapshots_tbl", "snpashot", "snpashots")},
	}
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
	return fmt.Sprintf("%s::%s", model.GetModelManager().Keyword(), model.GetId())
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

func (manager *SMetadataManager) AllowGetPropertyTagValuePairs(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (manager *SMetadataManager) GetPropertyTagValuePairs(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	q := manager.Query("key", "value").Distinct()
	sql, err := manager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	result := &struct {
		Total int
		Data  []struct {
			Key   string
			Value string
		} `json:"data,allowempty"`
	}{
		Total: 0,
		Data: []struct {
			Key   string
			Value string
		}{},
	}
	err = sql.All(&result.Data)
	if err != nil {
		return nil, err
	}
	result.Total = len(result.Data)
	return jsonutils.Marshal(result), nil
}

func (manager *SMetadataManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (manager *SMetadataManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	resources := jsonutils.GetQueryStringArray(query, "resources")
	if len(resources) == 0 {
		for resource := range ResourceMap {
			resources = append(resources, resource)
		}
	}
	conditions := []sqlchemy.ICondition{}
	admin := jsonutils.QueryBoolean(query, "admin", false)
	for _, resource := range resources {
		if man, ok := ResourceMap[resource]; ok {
			resourceView := man.Query().SubQuery()
			prefix := sqlchemy.NewStringField(fmt.Sprintf("%s::", man.Keyword()))
			field := sqlchemy.CONCAT(man.Keyword(), prefix, resourceView.Field("id"))
			sq := resourceView.Query(field)
			if !admin && !IsAllowList(rbacutils.ScopeSystem, userCred, man) {
				sq = man.FilterByOwner(sq, userCred, man.ResourceScope())
			}
			conditions = append(conditions, sqlchemy.In(q.Field("id"), sq))
		} else {
			return nil, httperrors.NewInputParameterError("Not support resource %s tag filter", resource)
		}
	}
	if len(conditions) > 0 {
		q = q.Filter(sqlchemy.OR(conditions...))
	}
	for args, prefix := range map[string]string{"sys_meta": SYS_TAG_PREFIX, "cloud_meta": CLOUD_TAG_PREFIX, "user_meta": USER_TAG_PREFIX} {
		if jsonutils.QueryBoolean(query, args, false) {
			q = q.Filter(sqlchemy.Startswith(q.Field("key"), prefix))
		}
	}

	withConditions := []sqlchemy.ICondition{}
	for args, prefix := range map[string]string{"with_sys_meta": SYS_TAG_PREFIX, "with_cloud_meta": CLOUD_TAG_PREFIX, "with_user_meta": USER_TAG_PREFIX} {
		if jsonutils.QueryBoolean(query, args, false) {
			withConditions = append(withConditions, sqlchemy.Startswith(q.Field("key"), prefix))
		}
	}

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
	NValue string
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
	changes, err := manager.SetValues(ctx, obj, store, userCred)
	if err != nil {
		return err
	}
	if len(changes) > 0 {
		OpsLog.LogEvent(obj, ACT_SET_METADATA, jsonutils.Marshal(changes), userCred)
	}
	return nil
}

func (manager *SMetadataManager) SetValues(ctx context.Context, obj IModel, store map[string]interface{}, userCred mcclient.TokenCredential) ([]sMetadataChange, error) {
	idStr := GetObjectIdstr(obj)

	lockman.LockObject(ctx, obj)
	defer lockman.ReleaseObject(ctx, obj)

	changes := make([]sMetadataChange, 0)
	for key, value := range store {

		valStr := stringutils.Interface2String(value)
		valStrLower := strings.ToLower(valStr)
		if valStrLower == "none" || valStrLower == "null" {
			valStr = ""
		}
		record := SMetadata{}
		err := manager.RawQuery().Equals("id", idStr).Equals("key", key).First(&record) //避免之前设置的tag被删除后再次设置时出现Duplicate entry error
		if err != nil {
			if err == sql.ErrNoRows {
				changes = append(changes, sMetadataChange{Key: key, NValue: valStr})
				record.Id = idStr
				record.Key = key
				record.Value = valStr
				err = manager.TableSpec().Insert(&record)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		} else {
			deleted := record.Deleted
			oValue := record.Value
			_, err := Update(&record, func() error {
				record.Deleted = false
				record.Value = valStr
				return nil
			})
			if err != nil {
				return nil, err
			}
			if deleted {
				changes = append(changes, sMetadataChange{Key: key, NValue: valStr})
			} else {
				if oValue != valStr {
					changes = append(changes, sMetadataChange{Key: key, OValue: oValue, NValue: valStr})
				}
			}
		}
	}
	return changes, nil
}

func (manager *SMetadataManager) SetAll(ctx context.Context, obj IModel, store map[string]interface{}, userCred mcclient.TokenCredential, delRange string) error {
	changes, err := manager.SetValues(ctx, obj, store, userCred)
	if err != nil {
		return err
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
		q = q.Like("key", USER_TAG_PREFIX+"%")
	case CLOUD_TAG_PREFIX:
		q = q.Like("key", CLOUD_TAG_PREFIX+"%")
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
		OpsLog.LogEvent(obj, ACT_SET_METADATA, jsonutils.Marshal(changes), userCred)
	}
	return nil
}

func (manager *SMetadataManager) GetAll(obj IModel, keys []string, userCred mcclient.TokenCredential) (map[string]string, error) {
	idStr := GetObjectIdstr(obj)
	records := make([]SMetadata, 0)
	q := manager.Query().Equals("id", idStr)
	if keys != nil && len(keys) > 0 {
		q = q.In("key", keys)
	}
	err := FetchModelObjects(manager, q, &records)
	if err != nil {
		return nil, err
	}
	ret := make(map[string]string)
	for _, rec := range records {
		if len(rec.Value) > 0 || strings.HasPrefix(rec.Key, USER_TAG_PREFIX) {
			ret[rec.Key] = rec.Value
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

func GetVisiableMetadata(model IMetadataModel, userCred mcclient.TokenCredential) (map[string]string, error) {
	metaData, err := model.GetAllMetadata(userCred)
	if err != nil {
		return nil, err
	}
	for _, key := range model.GetMetadataHideKeys() {
		delete(metaData, key)
	}
	for key := range metaData {
		if !IsMetadataKeyVisiable(key) {
			delete(metaData, key)
		}
	}
	return metaData, nil
}
