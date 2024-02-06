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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/onecloud/pkg/util/tagutils"
)

type SMetadataResourceBaseModelManager struct{}

func ObjectIdQueryWithPolicyResult(ctx context.Context, q *sqlchemy.SQuery, manager IModelManager, result rbacutils.SPolicyResult) *sqlchemy.SQuery {
	scope := manager.ResourceScope()
	if scope == rbacscope.ScopeDomain || scope == rbacscope.ScopeProject {
		if !result.DomainTags.IsEmpty() {
			tagFilters := tagutils.STagFilters{}
			tagFilters.AddFilters(result.DomainTags)
			q = ObjectIdQueryWithTagFilters(ctx, q, "domain_id", "domain", tagFilters)
		}
	}
	if scope == rbacscope.ScopeProject {
		if !result.ProjectTags.IsEmpty() {
			tagFilters := tagutils.STagFilters{}
			tagFilters.AddFilters(result.ProjectTags)
			q = ObjectIdQueryWithTagFilters(ctx, q, "tenant_id", "project", tagFilters)
		}
	}
	if !result.ObjectTags.IsEmpty() {
		tagFilters := tagutils.STagFilters{}
		tagFilters.AddFilters(result.ObjectTags)
		q = ObjectIdQueryWithTagFilters(ctx, q, "id", manager.Keyword(), tagFilters)
	}
	return q
}

func ObjectIdQueryWithTagFilters(ctx context.Context, q *sqlchemy.SQuery, idField string, modelName string, filters tagutils.STagFilters) *sqlchemy.SQuery {
	if len(filters.Filters) > 0 {
		sq := objIdQueryWithTags(ctx, modelName, filters.Filters)
		if sq != nil {
			q = q.In(idField, sq.SubQuery())
		}
	}
	if len(filters.NoFilters) > 0 {
		sq := objIdQueryWithTags(ctx, modelName, filters.NoFilters)
		if sq != nil {
			q = q.Filter(sqlchemy.NotIn(q.Field(idField), sq.SubQuery()))
		}
	}
	return q
}

func objIdQueryWithTags(ctx context.Context, modelName string, tagsList []map[string][]string) *sqlchemy.SQuery {
	manager := GetMetadaManagerInContext(ctx)
	q := manager.Query("obj_id").Equals("obj_type", modelName).Distinct()

	conditions := []sqlchemy.ICondition{}
	for _, tags := range tagsList {
		if len(tags) == 0 {
			continue
		}
		for k, vv := range tags {
			condition := sqlchemy.Equals(q.Field("key"), k)
			if len(vv) > 0 && utils.IsInArray(tagutils.NoValue, vv) {
				condition = sqlchemy.AND(
					sqlchemy.Equals(q.Field("key"), k),
					sqlchemy.In(q.Field("value"), vv),
				)
			}
			conditions = append(conditions, condition)
		}
	}

	if len(conditions) == 0 {
		return nil
	}

	return q.Filter(sqlchemy.OR(conditions...))
}

func (meta *SMetadataResourceBaseModelManager) ListItemFilter(
	ctx context.Context,
	manager IModelManager,
	q *sqlchemy.SQuery,
	input apis.MetadataResourceListInput,
) *sqlchemy.SQuery {
	metadataMan := GetMetadaManagerInContext(ctx)

	inputTagFilters := tagutils.STagFilters{}
	if len(input.Tags) > 0 {
		inputTagFilters.AddFilter(input.Tags)
	}
	if !input.ObjTags.IsEmpty() {
		inputTagFilters.AddFilters(input.ObjTags)
	}
	if len(input.NoTags) > 0 {
		inputTagFilters.AddNoFilter(input.NoTags)
	}
	if !input.NoObjTags.IsEmpty() {
		inputTagFilters.AddNoFilters(input.NoObjTags)
	}
	q = ObjectIdQueryWithTagFilters(ctx, q, "id", manager.Keyword(), inputTagFilters)

	//if !input.PolicyObjectTags.IsEmpty() {
	//	projTagFilters := tagutils.STagFilters{}
	//	projTagFilters.AddFilters(input.PolicyObjectTags)
	//	q = ObjectIdQueryWithTagFilters(q, "id", manager.Keyword(), projTagFilters)
	//}

	if input.WithoutUserMeta != nil || input.WithUserMeta != nil {
		sq := metadataMan.Query("obj_id").Equals("obj_type", manager.Keyword()).Startswith("key", USER_TAG_PREFIX).Distinct().SubQuery()
		if (input.WithoutUserMeta != nil && *input.WithoutUserMeta) || (input.WithUserMeta != nil && !*input.WithUserMeta) {
			q = q.Filter(sqlchemy.NotIn(q.Field("id"), sq))
		} else {
			q = q.Filter(sqlchemy.In(q.Field("id"), sq))
		}
	}

	if input.WithCloudMeta != nil {
		sq := metadataMan.Query("obj_id").Equals("obj_type", manager.Keyword()).Startswith("key", CLOUD_TAG_PREFIX).Distinct().SubQuery()
		if *input.WithCloudMeta {
			q = q.Filter(sqlchemy.In(q.Field("id"), sq))
		} else {
			q = q.Filter(sqlchemy.NotIn(q.Field("id"), sq))
		}
	}

	if input.WithAnyMeta != nil {
		sq := Metadata.Query("obj_id").Equals("obj_type", manager.Keyword()).SubQuery()
		if *input.WithAnyMeta {
			q = q.Filter(sqlchemy.In(q.Field("id"), sq))
		} else {
			q = q.Filter(sqlchemy.NotIn(q.Field("id"), sq))
		}
	}

	return q
}

func (meta *SMetadataResourceBaseModelManager) QueryDistinctExtraField(manager IModelManager, q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	if strings.HasPrefix(field, "tag:") {
		tagKey := field[4:]
		metaQ := Metadata.Query("obj_id", "value").Equals("obj_type", manager.Keyword()).Equals("key", tagKey).SubQuery()
		q = q.AppendField(metaQ.Field("value", field)).Distinct()
		q = q.LeftJoin(metaQ, sqlchemy.Equals(q.Field("id"), metaQ.Field("obj_id")))
		q = q.Asc(metaQ.Field("value"))
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (meta *SMetadataResourceBaseModelManager) OrderByExtraFields(
	manager IModelManager,
	q *sqlchemy.SQuery,
	input apis.MetadataResourceListInput,
) *sqlchemy.SQuery {
	if len(input.OrderByTag) > 0 {
		order := sqlchemy.SQL_ORDER_ASC
		tagKey := input.OrderByTag
		if stringutils2.HasSuffixIgnoreCase(input.OrderByTag, string(sqlchemy.SQL_ORDER_ASC)) {
			tagKey = tagKey[0 : len(tagKey)-len(sqlchemy.SQL_ORDER_ASC)-1]
		} else if stringutils2.HasSuffixIgnoreCase(input.OrderByTag, string(sqlchemy.SQL_ORDER_DESC)) {
			tagKey = tagKey[0 : len(tagKey)-len(sqlchemy.SQL_ORDER_DESC)-1]
			order = sqlchemy.SQL_ORDER_DESC
		}
		metaQ := Metadata.Query("obj_id", "value").Equals("obj_type", manager.Keyword()).Equals("key", tagKey).SubQuery()
		q = q.LeftJoin(metaQ, sqlchemy.Equals(q.Field("id"), metaQ.Field("obj_id")))
		if order == sqlchemy.SQL_ORDER_ASC {
			q = q.Asc(metaQ.Field("value"))
		} else {
			q = q.Desc(metaQ.Field("value"))
		}
	}
	return q
}

func (meta *SMetadataResourceBaseModelManager) FetchCustomizeColumns(
	manager IModelManager,
	userCred mcclient.TokenCredential,
	objs []interface{},
	fields stringutils2.SSortedStrings,
) []apis.MetadataResourceInfo {
	ret := make([]apis.MetadataResourceInfo, len(objs))
	resIds := make([]string, len(objs))
	for i := range objs {
		resIds[i] = GetModelIdstr(objs[i].(IModel))
	}

	if fields == nil || fields.Contains("__meta__") {
		q := Metadata.Query("id", "key", "value")
		metaKeyValues := make(map[string][]SMetadata)
		err := FetchQueryObjectsByIds(q, "id", resIds, &metaKeyValues)
		if err != nil {
			log.Errorf("FetchQueryObjectsByIds metadata fail %s", err)
			return ret
		}

		for i := range objs {
			if metaList, ok := metaKeyValues[resIds[i]]; ok {
				ret[i].Metadata = metaList2Map(manager.(IMetadataBaseModelManager), userCred, metaList)
			}
		}
	}

	return ret
}

const (
	TAG_EXPORT_KEY_PREFIX = "tag:"
)

func (meta *SMetadataResourceBaseModelManager) GetExportExtraKeys(keys stringutils2.SSortedStrings, rowMap map[string]string) *jsonutils.JSONDict {
	res := jsonutils.NewDict()
	for _, key := range keys {
		if strings.HasPrefix(key, TAG_EXPORT_KEY_PREFIX) {
			res.Add(jsonutils.NewString(rowMap[key]), key)
		}
	}
	return res
}

func (meta *SMetadataResourceBaseModelManager) ListItemExportKeys(manager IModelManager, q *sqlchemy.SQuery, keys stringutils2.SSortedStrings) *sqlchemy.SQuery {
	keyMaps := map[string]bool{}
	for _, key := range keys {
		if strings.HasPrefix(key, TAG_EXPORT_KEY_PREFIX) {
			tagKey := key[len(TAG_EXPORT_KEY_PREFIX):]
			if _, ok := keyMaps[strings.ToLower(tagKey)]; !ok {
				metaQ := Metadata.Query("obj_id", "value").Equals("obj_type", manager.Keyword()).Equals("key", tagKey).SubQuery()
				q = q.LeftJoin(metaQ, sqlchemy.Equals(q.Field("id"), metaQ.Field("obj_id")))
				q = q.AppendField(metaQ.Field("value", key))
				keyMaps[strings.ToLower(tagKey)] = true
			}
		}
	}
	return q
}
