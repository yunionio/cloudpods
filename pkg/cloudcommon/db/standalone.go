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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type UUIDGenerator func() string

var (
	DefaultUUIDGenerator = stringutils.UUID4
)

type SStandaloneResourceBase struct {
	SResourceBase

	// 资源UUID
	Id string `width:"128" charset:"ascii" primary:"true" list:"user" json:"id"`
	// 资源名称
	Name string `width:"128" charset:"utf8" nullable:"false" index:"true" list:"user" update:"user" create:"required" json:"name"`

	// 资源描述信息
	Description string `width:"256" charset:"utf8" get:"user" list:"user" update:"user" create:"optional" json:"description"`

	// 是否是模拟资源, 部分从公有云上同步的资源并不真实存在, 例如宿主机
	// list 接口默认不会返回这类资源，除非显示指定 is_emulate=true 过滤参数
	IsEmulated bool `nullable:"false" default:"false" list:"admin" create:"admin_optional" json:"is_emulated"`
}

func (model *SStandaloneResourceBase) BeforeInsert() {
	if len(model.Id) == 0 {
		model.Id = DefaultUUIDGenerator()
	}
}

type SStandaloneResourceBaseManager struct {
	SResourceBaseManager
	SMetadataResourceBaseModelManager

	NameRequireAscii bool
	NameLength       int
}

func NewStandaloneResourceBaseManager(
	dt interface{},
	tableName string,
	keyword string,
	keywordPlural string,
) SStandaloneResourceBaseManager {
	return SStandaloneResourceBaseManager{
		SResourceBaseManager: NewResourceBaseManager(dt, tableName, keyword, keywordPlural),
	}
}

func (manager *SStandaloneResourceBaseManager) IsStandaloneManager() bool {
	return true
}

func (self *SStandaloneResourceBaseManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return IsAdminAllowList(userCred, self)
}

func (self *SStandaloneResourceBaseManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return IsAdminAllowCreate(userCred, self)
}

func (self *SStandaloneResourceBase) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return IsAdminAllowGet(userCred, self)
}

func (self *SStandaloneResourceBase) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return IsAdminAllowUpdate(userCred, self)
}

func (self *SStandaloneResourceBase) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return IsAdminAllowDelete(userCred, self)
}

func (manager *SStandaloneResourceBaseManager) GetIStandaloneModelManager() IStandaloneModelManager {
	return manager.GetVirtualObject().(IStandaloneModelManager)
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

func (manager *SStandaloneResourceBaseManager) FilterByHiddenSystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	q = manager.SResourceBaseManager.FilterByHiddenSystemAttributes(q, userCred, query, scope)
	showEmulated := jsonutils.QueryBoolean(query, "show_emulated", false)
	if showEmulated {
		var isAllow bool
		if consts.IsRbacEnabled() {
			allowScope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionList, "show_emulated")
			if !scope.HigherThan(allowScope) {
				isAllow = true
			}
		} else {
			if userCred.HasSystemAdminPrivilege() {
				isAllow = true
			}
		}
		if !isAllow {
			showEmulated = false
		}
	}
	if !showEmulated {
		q = q.IsFalse("is_emulated")
	}
	return q
}

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
	return FetchById(manager.GetIStandaloneModelManager(), idStr)
}

func (manager *SStandaloneResourceBaseManager) FetchByName(userCred mcclient.IIdentityProvider, idStr string) (IModel, error) {
	return FetchByName(manager.GetIStandaloneModelManager(), userCred, idStr)
}

func (manager *SStandaloneResourceBaseManager) FetchByIdOrName(userCred mcclient.IIdentityProvider, idStr string) (IModel, error) {
	return FetchByIdOrName(manager.GetIStandaloneModelManager(), userCred, idStr)
}

func (manager *SStandaloneResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input apis.StandaloneResourceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ResourceBaseListInput)
	if err != nil {
		return q, errors.Wrap(err, "SResourceBaseManager.ListItemFilte")
	}

	// show_emulated is handled by FilterByHiddenSystemAttributes

	if len(input.Names) > 0 {
		q = q.In("name", input.Names)
	}

	if len(input.Ids) > 0 {
		q = q.In("id", input.Ids)
	}

	q = manager.SMetadataResourceBaseModelManager.ListItemFilter(manager.GetIModelManager(), q, input.MetadataResourceListInput)

	return q, nil
}

func (manager *SStandaloneResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SMetadataResourceBaseModelManager.QueryDistinctExtraField(manager.GetIModelManager(), q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SStandaloneResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input apis.StandaloneResourceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.ResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SResourceBaseManager.OrderByExtraFields")
	}

	q = manager.SMetadataResourceBaseModelManager.OrderByExtraFields(manager.GetIModelManager(), q, input.MetadataResourceListInput)

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

func (model *SStandaloneResourceBase) GetIStandaloneModel() IStandaloneModel {
	return model.GetVirtualObject().(IStandaloneModel)
}

func (model *SStandaloneResourceBase) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := model.SResourceBase.GetShortDesc(ctx)
	desc.Add(jsonutils.NewString(model.GetName()), "name")
	desc.Add(jsonutils.NewString(model.GetId()), "id")
	/*if len(model.ExternalId) > 0 {
		desc.Add(jsonutils.NewString(model.ExternalId), "external_id")
	}*/
	return desc
}

func (model *SStandaloneResourceBase) GetShortDescV2(ctx context.Context) *apis.StandaloneResourceShortDescDetail {
	desc := &apis.StandaloneResourceShortDescDetail{}
	desc.ModelBaseShortDescDetail = *model.SResourceBase.GetShortDescV2(ctx)
	desc.Name = model.GetName()
	desc.Id = model.GetId()
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
	err := Metadata.SetValue(ctx, model, key, value, userCred)
	if err != nil {
		return errors.Wrap(err, "SetValue")
	}
	model.GetIStandaloneModel().OnMetadataUpdated(ctx, userCred)
	return nil
}

func (model *SStandaloneResourceBase) SetAllMetadata(ctx context.Context, dictstore map[string]interface{}, userCred mcclient.TokenCredential) error {
	err := Metadata.SetValuesWithLog(ctx, model, dictstore, userCred)
	if err != nil {
		return errors.Wrap(err, "SetValuesWithLog")
	}
	model.GetIStandaloneModel().OnMetadataUpdated(ctx, userCred)
	return nil
}

func (model *SStandaloneResourceBase) SetUserMetadataValues(ctx context.Context, dictstore map[string]interface{}, userCred mcclient.TokenCredential) error {
	err := Metadata.SetValuesWithLog(ctx, model, dictstore, userCred)
	if err != nil {
		return errors.Wrap(err, "SetValuesWithLog")
	}
	model.GetIStandaloneModel().OnMetadataUpdated(ctx, userCred)
	return nil
}

func (model *SStandaloneResourceBase) SetUserMetadataAll(ctx context.Context, dictstore map[string]interface{}, userCred mcclient.TokenCredential) error {
	err := Metadata.SetAll(ctx, model, dictstore, userCred, USER_TAG_PREFIX)
	if err != nil {
		return errors.Wrap(err, "SetAll")
	}
	model.GetIStandaloneModel().OnMetadataUpdated(ctx, userCred)
	return nil
}

func (model *SStandaloneResourceBase) SetCloudMetadataAll(ctx context.Context, dictstore map[string]interface{}, userCred mcclient.TokenCredential) error {
	err := Metadata.SetAll(ctx, model, dictstore, userCred, CLOUD_TAG_PREFIX)
	if err != nil {
		return errors.Wrap(err, "SetAll")
	}
	model.GetIStandaloneModel().OnMetadataUpdated(ctx, userCred)
	return nil
}

func (model *SStandaloneResourceBase) RemoveMetadata(ctx context.Context, key string, userCred mcclient.TokenCredential) error {
	err := Metadata.SetValue(ctx, model, key, "", userCred)
	if err != nil {
		return errors.Wrap(err, "SetValue")
	}
	model.GetIStandaloneModel().OnMetadataUpdated(ctx, userCred)
	return nil
}

func (model *SStandaloneResourceBase) RemoveAllMetadata(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := Metadata.RemoveAll(ctx, model, userCred)
	if err != nil {
		return errors.Wrap(err, "RemoveAll")
	}
	model.GetIStandaloneModel().OnMetadataUpdated(ctx, userCred)
	return nil
}

func (model *SStandaloneResourceBase) GetAllMetadata(userCred mcclient.TokenCredential) (map[string]string, error) {
	return Metadata.GetAll(model, nil, "", userCred)
}

func (model *SStandaloneResourceBase) GetAllUserMetadata() (map[string]string, error) {
	meta, err := Metadata.GetAll(model, nil, USER_TAG_PREFIX, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Metadata.GetAll")
	}
	ret := make(map[string]string)
	for k, v := range meta {
		ret[k[len(USER_TAG_PREFIX):]] = v
	}
	return ret, nil
}

func (model *SStandaloneResourceBase) AllowGetDetailsMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return IsAllowGetSpec(rbacutils.ScopeSystem, userCred, model, "metadata")
}

// 获取资源标签（元数据）
func (model *SStandaloneResourceBase) GetDetailsMetadata(ctx context.Context, userCred mcclient.TokenCredential, input apis.GetMetadataInput) (apis.GetMetadataOutput, error) {
	val, err := Metadata.GetAll(model, input.Field, input.Prefix, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "Metadata.GetAll")
	}
	if len(input.Prefix) > 0 {
		// trim prefix from key
		ret := make(map[string]string)
		for k, v := range val {
			ret[k[len(input.Prefix):]] = v
		}
		val = ret
	}
	return val, nil
}

func (model *SStandaloneResourceBase) AllowPerformMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return IsAllowPerform(rbacutils.ScopeSystem, userCred, model, "metadata")
}

// +onecloud:swagger-gen-ignore
func (model *SStandaloneResourceBase) PerformMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformMetadataInput) (jsonutils.JSONObject, error) {
	dictStore := make(map[string]interface{})
	for k, v := range input {
		// 已双下滑线开头的metadata是系统内置，普通用户不可添加，只能查看
		if strings.HasPrefix(k, SYS_TAG_PREFIX) && (userCred == nil || !IsAllowPerform(rbacutils.ScopeSystem, userCred, model, "metadata")) {
			return nil, httperrors.NewForbiddenError("not allow to set system key, please remove the underscore at the beginning")
		}
		dictStore[k] = v
	}
	err := model.SetAllMetadata(ctx, dictStore, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "SetAllMetadata")
	}
	return nil, nil
}

func (model *SStandaloneResourceBase) AllowPerformUserMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return IsAllowPerform(rbacutils.ScopeSystem, userCred, model, "user-metadata")
}

// 更新资源的用户标签
// +onecloud:swagger-gen-ignore
func (model *SStandaloneResourceBase) PerformUserMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformUserMetadataInput) (jsonutils.JSONObject, error) {
	dictStore := make(map[string]interface{})
	for k, v := range input {
		dictStore[USER_TAG_PREFIX+k] = v
	}
	err := model.SetUserMetadataValues(ctx, dictStore, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "SetUserMetadataValues")
	}
	return nil, nil
}

func (model *SStandaloneResourceBase) AllowPerformSetUserMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return IsAllowPerform(rbacutils.ScopeSystem, userCred, model, "set-user-metadata")
}

// 全量替换资源的所有用户标签
func (model *SStandaloneResourceBase) PerformSetUserMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformSetUserMetadataInput) (jsonutils.JSONObject, error) {
	dictStore := make(map[string]interface{})
	for k, v := range input {
		if len(k) > 64-len(USER_TAG_PREFIX) {
			return nil, httperrors.NewInputParameterError("input key too long > %d", 64-len(USER_TAG_PREFIX))
		}
		if len(v) > 65535 {
			return nil, httperrors.NewInputParameterError("input value too long > %d", 65535)
		}
		dictStore[USER_TAG_PREFIX+k] = v
	}
	err := model.SetUserMetadataAll(ctx, dictStore, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "SetUserMetadataAll")
	}
	return nil, nil
}

func (model *SStandaloneResourceBase) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	model.SResourceBase.PostUpdate(ctx, userCred, query, data)

	meta := make(map[string]string)
	err := data.Unmarshal(&meta, "__meta__")
	if err == nil {
		model.PerformMetadata(ctx, userCred, nil, meta)
	}
}

func (model *SStandaloneResourceBase) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	model.SResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

	meta := make(map[string]string)
	err := data.Unmarshal(&meta, "__meta__")
	if err == nil {
		model.PerformMetadata(ctx, userCred, nil, meta)
	}
}

func (model *SStandaloneResourceBase) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	if model.Deleted {
		model.RemoveAllMetadata(ctx, userCred)
	}
	model.SResourceBase.PostDelete(ctx, userCred)
}

// func (model *SStandaloneResourceBase) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
// 	return DeleteModel(ctx, userCred, model)
// }

func (model *SStandaloneResourceBase) ClearSchedDescCache() error {
	return nil
}

func (model *SStandaloneResourceBase) AppendDescription(userCred mcclient.TokenCredential, msg string) error {
	_, err := Update(model.GetIStandaloneModel(), func() error {
		if len(model.Description) > 0 {
			model.Description += ";"
		}
		model.Description += msg
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "db.Update")
	}
	OpsLog.LogEvent(model, "append_desc", msg, userCred)
	return nil
}

func (manager *SStandaloneResourceBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input apis.StandaloneResourceCreateInput) (apis.StandaloneResourceCreateInput, error) {
	var err error
	input.ResourceBaseCreateInput, err = manager.SResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.ResourceBaseCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SResourceBaseManager.ValidateCreateData")
	}
	input.Name = strings.TrimSpace(input.Name)
	if strings.ContainsAny(input.Name, "\n\r\t") {
		return input, errors.Wrap(httperrors.ErrInputParameter, "name should not contains any \\n\\r\\t")
	}
	return input, nil
}

func (model *SStandaloneResourceBase) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (apis.StandaloneResourceDetails, error) {
	return apis.StandaloneResourceDetails{}, nil
}

func (manager *SStandaloneResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.StandaloneResourceDetails {
	ret := make([]apis.StandaloneResourceDetails, len(objs))
	upperRet := manager.SResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	metaRet := manager.SMetadataResourceBaseModelManager.FetchCustomizeColumns(manager.GetIModelManager(), userCred, objs, fields)
	for i := range objs {
		ret[i] = apis.StandaloneResourceDetails{
			ResourceBaseDetails:  upperRet[i],
			MetadataResourceInfo: metaRet[i],
		}
	}
	return ret
}

func (manager *SStandaloneResourceBaseManager) GetMetadataHiddenKeys() []string {
	return nil
}

func (manager *SStandaloneResourceBaseManager) GetExportExtraKeys(ctx context.Context, keys stringutils2.SSortedStrings, rowMap map[string]string) *jsonutils.JSONDict {
	res := manager.SResourceBaseManager.GetExportExtraKeys(ctx, keys, rowMap)

	metaRes := manager.SMetadataResourceBaseModelManager.GetExportExtraKeys(keys, rowMap)
	if metaRes.Length() > 0 {
		res.Update(metaRes)
	}

	return res
}

func (manager *SStandaloneResourceBaseManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SResourceBaseManager.ListItemExportKeys")
	}

	q = manager.SMetadataResourceBaseModelManager.ListItemExportKeys(manager.GetIModelManager(), q, keys)

	return q, nil
}

func (model *SStandaloneResourceBase) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.StandaloneResourceBaseUpdateInput) (apis.StandaloneResourceBaseUpdateInput, error) {
	var err error
	input.ResourceBaseUpdateInput, err = model.SResourceBase.ValidateUpdateData(ctx, userCred, query, input.ResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SModelBase.ValidateUpdateData")
	}

	if len(input.Name) > 0 {
		input.Name = strings.TrimSpace(input.Name)
		if strings.ContainsAny(input.Name, "\n\r\t") {
			return input, errors.Wrap(httperrors.ErrInputParameter, "name should not contains any \\n\\r\\t")
		}
		err = alterNameValidator(model.GetIStandaloneModel(), input.Name)
		if err != nil {
			return input, errors.Wrap(err, "alterNameValidator")
		}
	}
	return input, nil
}

func (model *SStandaloneResourceBase) IsShared() bool {
	return false
}

func (model *SStandaloneResourceBase) OnMetadataUpdated(ctx context.Context, userCred mcclient.TokenCredential) {
	// noop
}
