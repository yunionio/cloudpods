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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SStandaloneResourceBase struct {
	SResourceBase

	Id         string `width:"128" charset:"ascii" primary:"true" list:"user"`
	Name       string `width:"128" charset:"utf8" nullable:"false" index:"true" list:"user" update:"user" create:"required"`
	ExternalId string `width:"256" charset:"utf8" index:"true" list:"user" create:"admin_optional"`

	Description string `width:"256" charset:"utf8" get:"user" list:"user" update:"user" create:"optional"`

	IsEmulated bool `nullable:"false" default:"false" list:"admin" create:"admin_optional"`
}

func (model *SStandaloneResourceBase) BeforeInsert() {
	if len(model.Id) == 0 {
		model.Id = stringutils.UUID4()
	}
}

type SStandaloneResourceBaseManager struct {
	SResourceBaseManager
	NameRequireAscii bool
	NameLength       int
}

func NewStandaloneResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SStandaloneResourceBaseManager {
	return SStandaloneResourceBaseManager{SResourceBaseManager: NewResourceBaseManager(dt, tableName, keyword, keywordPlural)}
}

func (manager *SStandaloneResourceBaseManager) FilterById(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	return q.Equals("id", idStr)
}

func (manager *SStandaloneResourceBaseManager) FilterByNotId(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	return q.NotEquals("id", idStr)
}

func (manager *SStandaloneResourceBaseManager) FilterByName(q *sqlchemy.SQuery, name string) *sqlchemy.SQuery {
	return q.Equals("name", name)
}

/*func (manager *SStandaloneResourceBaseManager) IsNewNameUnique(name string, projectId string) bool {
	return manager.Query().Equals("name", name).Count() == 0
}*/

func (manager *SStandaloneResourceBaseManager) ValidateName(name string) error {
	if manager.NameRequireAscii && !regutils.MatchName(name) {
		return httperrors.NewInputParameterError("name starts with letter, and contains letter, number and ._@- only")
	}
	if manager.NameLength > 0 && len(name) > manager.NameLength {
		return httperrors.NewInputParameterError("name longer than %d", manager.NameLength)
	}
	return nil
}

func (manager *SStandaloneResourceBaseManager) FetchById(idStr string) (IModel, error) {
	return FetchById(manager, idStr)
}

func (manager *SStandaloneResourceBaseManager) FetchByName(userCred mcclient.IIdentityProvider, idStr string) (IModel, error) {
	return FetchByName(manager, userCred, idStr)
}

func (manager *SStandaloneResourceBaseManager) FetchByIdOrName(userCred mcclient.IIdentityProvider, idStr string) (IModel, error) {
	return FetchByIdOrName(manager, userCred, idStr)
}

func (manager *SStandaloneResourceBaseManager) FetchByExternalId(idStr string) (IStandaloneModel, error) {
	q := manager.Query().Equals("external_id", idStr)
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count == 1 {
		obj, err := NewModelObject(manager)
		if err != nil {
			return nil, err
		}
		err = q.First(obj)
		if err != nil {
			return nil, err
		} else {
			return obj.(IStandaloneModel), nil
		}
	} else if count > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	} else {
		return nil, sql.ErrNoRows
	}
}

type STagValue struct {
	value string
	exist bool
}

func (manager *SStandaloneResourceBaseManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return q, err
	}

	showEmulated := jsonutils.QueryBoolean(query, "show_emulated", false)
	if !showEmulated {
		q = q.Filter(sqlchemy.IsFalse(q.Field("is_emulated")))
	}

	tags := map[string]STagValue{}
	if query.Contains("tags") {
		idx := 0
		for {
			key, _ := query.GetString("tags", fmt.Sprintf("%d", idx), "key")
			if len(key) == 0 {
				break
			}
			value := STagValue{exist: false}
			if query.Contains("tags", fmt.Sprintf("%d", idx), "value") {
				value.value, _ = query.GetString("tags", fmt.Sprintf("%d", idx), "value")
				value.exist = true
			}
			tags[key] = value
			idx++
		}
	}

	if len(tags) > 0 {
		metadataView := Metadata.Query("id")
		idx := 0
		for k, v := range tags {
			if idx == 0 {
				metadataView = metadataView.Equals("key", k)
				if v.exist {
					metadataView = metadataView.Equals("value", v.value)
				}
			} else {
				subMetataView := Metadata.Query().Equals("key", k)
				if v.exist {
					subMetataView = subMetataView.Equals("value", v.value)
				}
				sq := subMetataView.SubQuery()
				metadataView.Join(sq, sqlchemy.Equals(metadataView.Field("id"), sq.Field("id")))
			}
			idx++
		}
		metadataView = metadataView.Filter(sqlchemy.Like(metadataView.Field("id"), manager.Keyword()+"::%")).Distinct()
		resourceIds := []string{}
		rows, err := metadataView.Rows()
		if err != nil {
			log.Errorf("query metadata ids error: %v", err)
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var metadataID string
			err = rows.Scan(&metadataID)
			if err != nil {
				log.Errorf("get metadata id scan error: %v", err)
				return nil, err
			}
			resourceIds = append(resourceIds, strings.TrimLeft(metadataID, manager.Keyword()+"::"))
		}
		q = q.Filter(sqlchemy.In(q.Field("id"), resourceIds))
	}
	return q, nil
}

func (model *SStandaloneResourceBase) StandaloneModelManager() IStandaloneModelManager {
	return model.GetModelManager().(IStandaloneModelManager)
}

func (model *SStandaloneResourceBase) GetId() string {
	return model.Id
}

func (model *SStandaloneResourceBase) GetName() string {
	return model.Name
}

func (model *SStandaloneResourceBase) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := model.SResourceBase.GetShortDesc(ctx)
	desc.Add(jsonutils.NewString(model.GetName()), "name")
	desc.Add(jsonutils.NewString(model.GetId()), "id")
	if len(model.ExternalId) > 0 {
		desc.Add(jsonutils.NewString(model.ExternalId), "external_id")
	}
	return desc
}

/*
 * userCred: optional
 */
func (model *SStandaloneResourceBase) GetMetadata(key string, userCred mcclient.TokenCredential) string {
	return Metadata.GetStringValue(model, key, userCred)
}

func (model *SStandaloneResourceBase) GetMetadataJson(key string, userCred mcclient.TokenCredential) jsonutils.JSONObject {
	return Metadata.GetJsonValue(model, key, userCred)
}

func (model *SStandaloneResourceBase) SetMetadata(ctx context.Context, key string, value interface{}, userCred mcclient.TokenCredential) error {
	if Metadata.IsSystemAdminKey(key) && !IsAdminAllowPerform(userCred, model, "metadata") {
		return httperrors.NewNotSufficientPrivilegeError("cannot set system key")
	}
	return Metadata.SetValue(ctx, model, key, value, userCred)
}

func (model *SStandaloneResourceBase) SetAllMetadata(ctx context.Context, dictstore map[string]interface{}, userCred mcclient.TokenCredential) error {
	for k := range dictstore {
		if Metadata.IsSystemAdminKey(k) && !IsAdminAllowPerform(userCred, model, "metadata") {
			return httperrors.NewNotSufficientPrivilegeError("not allow to set system key %s", k)
		}
	}
	return Metadata.SetValuesWithLog(ctx, model, dictstore, userCred)
}

func (model *SStandaloneResourceBase) SetUserMetadataValues(ctx context.Context, dictstore map[string]interface{}, userCred mcclient.TokenCredential) error {
	return Metadata.SetValuesWithLog(ctx, model, dictstore, userCred)
}

func (model *SStandaloneResourceBase) SetUserMetadataAll(ctx context.Context, dictstore map[string]interface{}, userCred mcclient.TokenCredential) error {
	return Metadata.SetAll(ctx, model, dictstore, userCred, "user")
}

func (model *SStandaloneResourceBase) RemoveMetadata(ctx context.Context, key string, userCred mcclient.TokenCredential) error {
	return Metadata.SetValue(ctx, model, key, "", userCred)
}

func (model *SStandaloneResourceBase) RemoveAllMetadata(ctx context.Context, userCred mcclient.TokenCredential) error {
	return Metadata.RemoveAll(ctx, model, userCred)
}

func (model *SStandaloneResourceBase) GetAllMetadata(userCred mcclient.TokenCredential) (map[string]string, error) {
	return Metadata.GetAll(model, nil, userCred)
}

func (model *SStandaloneResourceBase) AllowGetDetailsMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return IsAdminAllowGetSpec(userCred, model, "metadata")
}

func (model *SStandaloneResourceBase) GetDetailsMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	fields := jsonutils.GetQueryStringArray(query, "field")
	val, err := Metadata.GetAll(model, fields, userCred)
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(val), nil
}

func (model *SStandaloneResourceBase) AllowPerformMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return IsAdminAllowPerform(userCred, model, "metadata")
}

func (model *SStandaloneResourceBase) PerformMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	dict, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, httperrors.NewInputParameterError("input data not key value dict")
	}
	dictMap, err := dict.GetMap()
	if err != nil {
		return nil, err
	}
	dictStore := make(map[string]interface{})
	for k, v := range dictMap {
		dictStore[k], _ = v.GetString()
	}
	err = model.SetAllMetadata(ctx, dictStore, userCred)
	return nil, err
}

func (model *SStandaloneResourceBase) AllowPerformUserMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return IsAdminAllowPerform(userCred, model, "user-metadata")
}

func (model *SStandaloneResourceBase) PerformUserMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	dict, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, httperrors.NewInputParameterError("input data not key value dict")
	}
	dictMap, err := dict.GetMap()
	if err != nil {
		return nil, err
	}
	dictStore := make(map[string]interface{})
	for k, v := range dictMap {
		dictStore["user:"+k], _ = v.GetString()
	}
	err = model.SetUserMetadataValues(ctx, dictStore, userCred)
	return nil, err
}

func (model *SStandaloneResourceBase) AllowPerformSetUserMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return IsAdminAllowPerform(userCred, model, "set-user-metadata")
}

func (model *SStandaloneResourceBase) PerformSetUserMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	dict, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, httperrors.NewInputParameterError("input data not key value dict")
	}
	dictMap, err := dict.GetMap()
	if err != nil {
		return nil, err
	}
	dictStore := make(map[string]interface{})
	for k, v := range dictMap {
		dictStore["user:"+k], _ = v.GetString()
	}
	err = model.SetUserMetadataAll(ctx, dictStore, userCred)
	return nil, err
}

func (model *SStandaloneResourceBase) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := model.SResourceBase.GetCustomizeColumns(ctx, userCred, query)
	withMeta, _ := query.GetString("with_meta")
	if utils.ToBool(withMeta) {
		jsonMeta, err := Metadata.GetAll(model, nil, userCred)
		if err == nil {
			extra.Add(jsonutils.Marshal(jsonMeta), "metadata")
		} else {
			log.Errorf("metadata GetAll fail: %s", err)
		}
	}
	return extra
}

func (model *SStandaloneResourceBase) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	model.SResourceBase.PostUpdate(ctx, userCred, query, data)

	jsonMeta, _ := data.Get("__meta__")
	if jsonMeta != nil {
		model.PerformMetadata(ctx, userCred, nil, jsonMeta)
	}
}

func (model *SStandaloneResourceBase) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	model.SResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)

	jsonMeta, _ := data.Get("__meta__")
	if jsonMeta != nil {
		model.PerformMetadata(ctx, userCred, nil, jsonMeta)
	}
}

func (model *SStandaloneResourceBase) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	if model.Deleted {
		model.RemoveAllMetadata(ctx, userCred)
	}
	model.SResourceBase.PostDelete(ctx, userCred)
}

func (model *SStandaloneResourceBase) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return DeleteModel(ctx, userCred, model)
}

func (model SStandaloneResourceBase) GetExternalId() string {
	return model.ExternalId
}

func (model *SStandaloneResourceBase) SetExternalId(userCred mcclient.TokenCredential, idstr string) error {
	if model.ExternalId != idstr {
		diff, err := Update(model, func() error {
			model.ExternalId = idstr
			return nil
		})
		if err == nil {
			OpsLog.LogEvent(model, ACT_UPDATE, diff, userCred)
		}
		return err
	}
	return nil
}
