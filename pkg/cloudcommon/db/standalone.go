package db

import (
	"context"
	"database/sql"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"
)

type SStandaloneResourceBase struct {
	SResourceBase

	Id         string `width:"128" charset:"ascii" primary:"true" list:"user"`
	Name       string `width:"128" charset:"utf8" nullable:"false" index:"true" list:"user" update:"user" create:"required"`
	ExternalId string `width:"128" charset:"ascii" index:"true" list:"admin" create:"admin_optional"`

	Description string `width:"256" charset:"utf8" get:"user" list:"user" update:"user" create:"optional"`

	IsEmulated bool `nullable:"false" default:"false" list:"admin" update:"true" create:"admin_optional"`
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
		return httperrors.NewInputParameterError(fmt.Sprintf("name longer than %d", manager.NameLength))
	}
	return nil
}

func (manager *SStandaloneResourceBaseManager) FetchById(idStr string) (IModel, error) {
	return fetchById(manager, idStr)
}

func (manager *SStandaloneResourceBaseManager) FetchByName(ownerProjId string, idStr string) (IModel, error) {
	return fetchByName(manager, ownerProjId, idStr)
}

func (manager *SStandaloneResourceBaseManager) FetchByIdOrName(ownerProjId string, idStr string) (IModel, error) {
	return fetchByIdOrName(manager, ownerProjId, idStr)
}

func (manager *SStandaloneResourceBaseManager) FetchByExternalId(idStr string) (IStandaloneModel, error) {
	q := manager.Query().Equals("external_id", idStr)
	count := q.Count()
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

func (manager *SStandaloneResourceBaseManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return q, err
	}

	showEmulated := jsonutils.QueryBoolean(query, "show_emulated", false)
	if !showEmulated {
		q = q.Filter(sqlchemy.IsFalse(q.Field("is_emulated")))
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

func (model *SStandaloneResourceBase) GetShortDesc() *jsonutils.JSONDict {
	desc := model.SResourceBase.GetShortDesc()
	desc.Add(jsonutils.NewString(model.GetName()), "name")
	desc.Add(jsonutils.NewString(model.GetId()), "id")
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
	if Metadata.IsSystemAdminKey(key) && !userCred.IsSystemAdmin() {
		return httperrors.NewNotSufficientPrivilegeError("cannot set system key")
	}
	return Metadata.SetValue(ctx, model, key, value, userCred)
}

func (model *SStandaloneResourceBase) SetAllMetadata(ctx context.Context, dictstore map[string]interface{}, userCred mcclient.TokenCredential) error {
	for k := range dictstore {
		if Metadata.IsSystemAdminKey(k) && !userCred.IsSystemAdmin() {
			return httperrors.NewNotSufficientPrivilegeError(fmt.Sprintf("not allow to set system key %s", k))
		}
	}
	return Metadata.SetAll(ctx, model, dictstore, userCred)
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
	return userCred.IsSystemAdmin()
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
	return userCred.IsSystemAdmin()
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
		dictStore[k] = v
	}
	model.SetAllMetadata(ctx, dictStore, userCred)
	return nil, nil
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

func (model *SStandaloneResourceBase) SetExternalId(idstr string) error {
	_, err := model.GetModelManager().TableSpec().Update(model, func() error {
		model.ExternalId = idstr
		return nil
	})
	return err
}
