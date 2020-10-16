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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SMetadataResourceBaseModelManager struct{}

func (meta *SMetadataResourceBaseModelManager) ListItemFilter(
	manager IModelManager,
	q *sqlchemy.SQuery,
	input apis.MetadataResourceListInput,
) *sqlchemy.SQuery {
	tags := map[string][]string{}
	for _, tag := range input.Tags {
		if _, ok := tags[tag.Key]; !ok {
			tags[tag.Key] = []string{}
		}
		if len(tag.Value) > 0 && !utils.IsInStringArray(tag.Value, tags[tag.Key]) {
			tags[tag.Key] = append(tags[tag.Key], tag.Value)
		}
	}

	if len(tags) > 0 {
		metadataResQ := Metadata.Query().Equals("obj_type", manager.Keyword()).SubQuery()
		metadataView := metadataResQ.Query()
		idx := 0
		for key, values := range tags {
			if idx == 0 {
				metadataView = metadataView.Equals("key", key)
				if len(values) > 0 {
					metadataView = metadataView.In("value", values)
				}
			} else {
				subMetataView := metadataResQ.Query().Equals("key", key)
				if len(values) > 0 {
					subMetataView = subMetataView.In("value", values)
				}
				sq := subMetataView.SubQuery()
				metadataView.Join(sq, sqlchemy.Equals(metadataView.Field("id"), sq.Field("id")))
			}
			idx++
		}
		metadatas := metadataView.SubQuery()
		sq := metadatas.Query(metadatas.Field("obj_id")).Distinct().SubQuery()
		q = q.Filter(sqlchemy.In(q.Field("id"), sq))
	}

	if input.WithoutUserMeta != nil || input.WithUserMeta != nil {
		metadatas := Metadata.Query().Equals("obj_type", manager.Keyword()).SubQuery()
		sq := metadatas.Query(metadatas.Field("obj_id")).Startswith("key", USER_TAG_PREFIX).Distinct().SubQuery()
		if (input.WithoutUserMeta != nil && *input.WithoutUserMeta) || (input.WithUserMeta != nil && !*input.WithUserMeta) {
			q = q.Filter(sqlchemy.NotIn(q.Field("id"), sq))
		} else {
			q = q.Filter(sqlchemy.In(q.Field("id"), sq))
		}
	}

	if input.WithCloudMeta != nil {
		metadatas := Metadata.Query().Equals("obj_type", manager.Keyword()).SubQuery()
		sq := metadatas.Query(metadatas.Field("obj_id")).Startswith("key", CLOUD_TAG_PREFIX).Distinct().SubQuery()
		if *input.WithCloudMeta {
			q = q.Filter(sqlchemy.In(q.Field("id"), sq))
		} else {
			q = q.Filter(sqlchemy.NotIn(q.Field("id"), sq))
		}
	}

	if input.WithAnyMeta != nil {
		metadatas := Metadata.Query().Equals("obj_type", manager.Keyword()).SubQuery()
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
		resIds[i] = GetObjectIdstr(objs[i].(IModel))
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
	for _, key := range keys {
		if strings.HasPrefix(key, TAG_EXPORT_KEY_PREFIX) {
			tagKey := key[len(TAG_EXPORT_KEY_PREFIX):]
			metaQ := Metadata.Query("obj_id", "value").Equals("obj_type", manager.Keyword()).Equals("key", tagKey).SubQuery()
			q = q.LeftJoin(metaQ, sqlchemy.Equals(q.Field("id"), metaQ.Field("obj_id")))
			q = q.AppendField(metaQ.Field("value", key))
		}
	}
	return q
}
