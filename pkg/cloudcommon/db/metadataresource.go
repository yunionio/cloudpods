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
	"crypto/md5"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/hashcache"
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

func ObjectIdQueryWithTagFiltersOptimized(ctx context.Context, q *sqlchemy.SQuery, idField string, modelName string, filters tagutils.STagFilters) *sqlchemy.SQuery {
	if len(filters.Filters) > 0 || len(filters.NoFilters) > 0 {
		idSubQ := q.Copy().SubQuery().Query()
		idSubQ.AppendField(sqlchemy.DISTINCT(idField, idSubQ.Field(idField)))
		if len(filters.Filters) > 0 {
			if GetMetadaManagerInContext(ctx) == Metadata {
				sq := tenantIdQueryWithTags(ctx, modelName, filters.Filters)
				q = q.In(idField, sq.SubQuery())
			} else { // clickhouse
				ids := tenantIdQueryWithTagsWithCache(ctx, modelName, filters.Filters)
				if len(ids) > 0 {
					q = q.In(idField, ids)
				}
			}
		}
		if len(filters.NoFilters) > 0 {
			if GetMetadaManagerInContext(ctx) == Metadata {
				sq := tenantIdQueryWithTags(ctx, modelName, filters.Filters)
				q = q.NotIn(idField, sq.SubQuery())
			} else { // clickhouse
				ids := tenantIdQueryWithTagsWithCache(ctx, modelName, filters.NoFilters)
				if len(ids) > 0 {
					q = q.NotIn(idField, ids)
				}
			}
		}
	}
	return q
}

func ObjectIdQueryWithTagFilters(ctx context.Context, q *sqlchemy.SQuery, idField string, modelName string, filters tagutils.STagFilters) *sqlchemy.SQuery {
	if len(filters.Filters) > 0 || len(filters.NoFilters) > 0 {
		idSubQ := q.Copy().SubQuery().Query()
		idSubQ.AppendField(sqlchemy.DISTINCT(idField, idSubQ.Field(idField)))
		subQ := idSubQ.SubQuery()
		if len(filters.Filters) > 0 {
			sq := objIdQueryWithTags(ctx, subQ, idField, modelName, filters.Filters)
			if sq != nil {
				sqq := sq.SubQuery()
				q = q.Join(sqq, sqlchemy.Equals(q.Field(idField), sqq.Field(idField)))
			}
		}
		if len(filters.NoFilters) > 0 {
			sq := objIdQueryWithTags(ctx, subQ, idField, modelName, filters.NoFilters)
			if sq != nil {
				sqq := sq.SubQuery()
				q = q.LeftJoin(sqq, sqlchemy.Equals(q.Field(idField), sqq.Field(idField)))
				q = q.Filter(sqlchemy.IsNull(sqq.Field(idField)))
			}
		}
	}
	return q
}

func ExtendQueryWithTag(ctx context.Context, q *sqlchemy.SQuery, idField string, modelName string, key string, fieldLabel string) *sqlchemy.SQuery {
	manager := GetMetadaManagerInContext(ctx)
	metadataQ := manager.Query().Equals("obj_type", modelName).Equals("key", key)
	metadataQ = metadataQ.AppendField(metadataQ.Field("value").Label(fieldLabel))
	metadataResQ := metadataQ.SubQuery()

	q = q.LeftJoin(metadataResQ, sqlchemy.Equals(q.Field(idField), metadataResQ.Field("obj_id")))
	q = q.AppendField(metadataResQ.Field(fieldLabel).Label(fieldLabel))

	return q
}

func tenantIdQueryWithTags(ctx context.Context, modelName string, tagsList []map[string][]string) *sqlchemy.SQuery {
	manager := GetMetadaManagerInContext(ctx)

	conditions := []sqlchemy.ICondition{}
	sq := manager.Query("obj_id")
	for _, tags := range tagsList {
		if len(tags) == 0 {
			continue
		}
		subconds := []sqlchemy.ICondition{}
		for key, val := range tags {
			if len(val) > 0 {
				sqq := sq.Copy().Equals("obj_type", modelName).Equals("key", key).In("value", val)
				subconds = append(subconds, sqlchemy.In(sq.Field("obj_id"), sqq.SubQuery()))
			} else {
				sqq := sq.Copy().Equals("obj_type", modelName).Equals("key", key)
				subconds = append(subconds, sqlchemy.In(sq.Field("obj_id"), sqq.SubQuery()))
			}
		}
		conditions = append(conditions, sqlchemy.AND(subconds...))
	}
	return sq.Filter(sqlchemy.OR(conditions...)).Distinct()
}

var (
	tagsCache = hashcache.NewCache(1024, time.Minute*15)
)

func tenantIdQueryWithTagsWithCache(ctx context.Context, modelName string, tagsList []map[string][]string) []string {
	manager := Metadata

	ret := []string{}
	sq := manager.Query("obj_id")
	for _, tags := range tagsList {
		if len(tags) == 0 {
			continue
		}
		hashKeys := []string{modelName, jsonutils.Marshal(tags).String()}
		hash := fmt.Sprintf("%x", md5.Sum([]byte(jsonutils.Marshal(hashKeys).String())))
		cache := tagsCache.Get(hash)
		if cache != nil {
			ids := cache.([]string)
			ret = append(ret, ids...)
			log.Debugf("cache hit %s %s %s", hash, hashKeys, ids)
			continue
		}
		conditions := []sqlchemy.ICondition{}
		for key, val := range tags {
			if len(val) > 0 {
				sqq := sq.Copy().Equals("obj_type", modelName).Equals("key", key).In("value", val)
				conditions = append(conditions, sqlchemy.In(sq.Field("obj_id"), sqq.SubQuery()))
			} else {
				sqq := sq.Copy().Equals("obj_type", modelName).Equals("key", key)
				conditions = append(conditions, sqlchemy.In(sq.Field("obj_id"), sqq.SubQuery()))
			}
		}
		ids, err := FetchIds(sq.Copy().Filter(sqlchemy.AND(conditions...)).Distinct())
		if err != nil {
			log.Errorf("FetchIds %s %v", sq.String(), err)
			continue
		}
		ret = append(ret, ids...)
		log.Debugf("cache miss %s %s %s", hash, hashKeys, ids)
		tagsCache.AtomicSet(hash, ids)
	}
	return ret
}

func objIdQueryWithTags(ctx context.Context, objIdSubQ *sqlchemy.SSubQuery, idField string, modelName string, tagsList []map[string][]string) *sqlchemy.SQuery {
	manager := GetMetadaManagerInContext(ctx)

	queries := make([]sqlchemy.IQuery, 0)
	for _, tags := range tagsList {
		if len(tags) == 0 {
			continue
		}
		objIdQ := objIdSubQ.Query()
		objIdQ = objIdQ.AppendField(objIdQ.Field(idField))
		for key, val := range tags {
			sq := manager.Query("obj_id").Equals("obj_type", modelName).Equals("key", key)
			if len(val) > 0 {
				ssq := sq.In("value", val).SubQuery()
				if utils.IsInArray(tagutils.NoValue, val) {
					objIdQ = objIdQ.LeftJoin(ssq, sqlchemy.Equals(objIdQ.Field(idField), ssq.Field("obj_id")))
				} else {
					objIdQ = objIdQ.Join(ssq, sqlchemy.Equals(objIdQ.Field(idField), ssq.Field("obj_id")))
				}
			} else {
				ssq := sq.SubQuery()
				objIdQ = objIdQ.Join(ssq, sqlchemy.Equals(objIdQ.Field(idField), ssq.Field("obj_id")))
			}
		}
		queries = append(queries, objIdQ.Distinct())
	}
	if len(queries) == 0 {
		return nil
	}
	var query *sqlchemy.SQuery
	if len(queries) == 1 {
		query = queries[0].(*sqlchemy.SQuery)
	} else {
		uq, _ := sqlchemy.UnionWithError(queries...)
		query = uq.Query()
	}
	return query
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
		metadatas := metadataMan.Query().Equals("obj_type", manager.Keyword()).SubQuery()
		sq := metadatas.Query(metadatas.Field("obj_id")).Startswith("key", USER_TAG_PREFIX).Distinct().SubQuery()
		if (input.WithoutUserMeta != nil && *input.WithoutUserMeta) || (input.WithUserMeta != nil && !*input.WithUserMeta) {
			q = q.Filter(sqlchemy.NotIn(q.Field("id"), sq))
		} else {
			q = q.Filter(sqlchemy.In(q.Field("id"), sq))
		}
	}

	if input.WithCloudMeta != nil {
		metadatas := metadataMan.Query().Equals("obj_type", manager.Keyword()).SubQuery()
		sq := metadatas.Query(metadatas.Field("obj_id")).Startswith("key", CLOUD_TAG_PREFIX).Distinct().SubQuery()
		if *input.WithCloudMeta {
			q = q.Filter(sqlchemy.In(q.Field("id"), sq))
		} else {
			q = q.Filter(sqlchemy.NotIn(q.Field("id"), sq))
		}
	}

	if input.WithAnyMeta != nil {
		metadatas := metadataMan.Query().Equals("obj_type", manager.Keyword()).SubQuery()
		sq := metadatas.Query(metadatas.Field("obj_id")).Distinct().SubQuery()
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

	if fields == nil || fields.Contains("__meta__") || fields.Contains("metadata") {
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
