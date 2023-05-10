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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/onecloud/pkg/util/tagutils"
)

type UUIDGenerator func() string

var (
	DefaultUUIDGenerator = stringutils.UUID4
)

type SStandaloneAnonResourceBase struct {
	SResourceBase

	// 资源UUID
	Id string `width:"128" charset:"ascii" primary:"true" list:"user" create:"optional" json:"id"`

	// 资源描述信息
	Description string `width:"256" charset:"utf8" get:"user" list:"user" update:"user" create:"optional" json:"description"`

	// 是否是模拟资源, 部分从公有云上同步的资源并不真实存在, 例如宿主机
	// list 接口默认不会返回这类资源，除非显示指定 is_emulate=true 过滤参数
	IsEmulated bool `nullable:"false" default:"false" list:"admin" create:"admin_optional" json:"is_emulated"`
}

func (model *SStandaloneAnonResourceBase) BeforeInsert() {
	if len(model.Id) == 0 {
		model.Id = DefaultUUIDGenerator()
	}
}

type SStandaloneAnonResourceBaseManager struct {
	SResourceBaseManager
	SMetadataResourceBaseModelManager
}

func NewStandaloneAnonResourceBaseManager(
	dt interface{},
	tableName string,
	keyword string,
	keywordPlural string,
) SStandaloneAnonResourceBaseManager {
	return SStandaloneAnonResourceBaseManager{
		SResourceBaseManager: NewResourceBaseManager(dt, tableName, keyword, keywordPlural),
	}
}

func (manager *SStandaloneAnonResourceBaseManager) CreateByInsertOrUpdate() bool {
	return false
}

func (manager *SStandaloneAnonResourceBaseManager) IsStandaloneManager() bool {
	return true
}

func (manager *SStandaloneAnonResourceBaseManager) GetIStandaloneModelManager() IStandaloneModelManager {
	return manager.GetVirtualObject().(IStandaloneModelManager)
}

func (manager *SStandaloneAnonResourceBaseManager) FilterById(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	return q.Equals("id", idStr)
}

func (manager *SStandaloneAnonResourceBaseManager) FilterByName(q *sqlchemy.SQuery, name string) *sqlchemy.SQuery {
	return q.FilterByFalse()
}

func (manager *SStandaloneAnonResourceBaseManager) FilterByNotId(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	return q.NotEquals("id", idStr)
}

func (manager *SStandaloneAnonResourceBaseManager) FilterByOwner(q *sqlchemy.SQuery, man FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	if userCred != nil {
		result := policy.PolicyManager.Allow(scope, userCred, consts.GetServiceType(), man.KeywordPlural(), policy.PolicyActionList)
		if !result.ObjectTags.IsEmpty() {
			policyTagFilters := tagutils.STagFilters{}
			policyTagFilters.AddFilters(result.ObjectTags)
			q = ObjectIdQueryWithTagFilters(q, "id", man.Keyword(), policyTagFilters)
		}
	}
	return q
}

func (manager *SStandaloneAnonResourceBaseManager) FilterByHiddenSystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	q = manager.SResourceBaseManager.FilterByHiddenSystemAttributes(q, userCred, query, scope)
	showEmulated := jsonutils.QueryBoolean(query, "show_emulated", false)
	if showEmulated {
		var isAllow bool
		// TODO, add tagfilter
		allowScope, _ := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionList, "show_emulated")
		if !scope.HigherThan(allowScope) {
			isAllow = true
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

func (manager *SStandaloneAnonResourceBaseManager) FetchById(idStr string) (IModel, error) {
	return FetchById(manager.GetIStandaloneModelManager(), idStr)
}

func (manager *SStandaloneAnonResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input apis.StandaloneAnonResourceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ResourceBaseListInput)
	if err != nil {
		return q, errors.Wrap(err, "SResourceBaseManager.ListItemFilte")
	}

	// show_emulated is handled by FilterByHiddenSystemAttributes

	if len(input.Ids) > 0 {
		q = q.In("id", input.Ids)
	}

	q = manager.SMetadataResourceBaseModelManager.ListItemFilter(manager.GetIModelManager(), q, input.MetadataResourceListInput)

	return q, nil
}

func (manager *SStandaloneAnonResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
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

func (manager *SStandaloneAnonResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input apis.StandaloneAnonResourceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.ResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SResourceBaseManager.OrderByExtraFields")
	}

	q = manager.SMetadataResourceBaseModelManager.OrderByExtraFields(manager.GetIModelManager(), q, input.MetadataResourceListInput)

	return q, nil
}

func (model *SStandaloneAnonResourceBase) StandaloneModelManager() IStandaloneModelManager {
	return model.GetModelManager().(IStandaloneModelManager)
}

func (model SStandaloneAnonResourceBase) GetId() string {
	return model.Id
}

func (model *SStandaloneAnonResourceBase) GetIStandaloneModel() IStandaloneModel {
	return model.GetVirtualObject().(IStandaloneModel)
}

func (model *SStandaloneAnonResourceBase) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := model.SResourceBase.GetShortDesc(ctx)
	desc.Add(jsonutils.NewString(model.GetId()), "id")
	/*if len(model.ExternalId) > 0 {
		desc.Add(jsonutils.NewString(model.ExternalId), "external_id")
	}*/
	return desc
}

func (model *SStandaloneAnonResourceBase) GetShortDescV2(ctx context.Context) *apis.StandaloneAnonResourceShortDescDetail {
	desc := &apis.StandaloneAnonResourceShortDescDetail{}
	desc.ModelBaseShortDescDetail = *model.SResourceBase.GetShortDescV2(ctx)
	desc.Id = model.GetId()
	return desc
}

/*
 * userCred: optional
 */
func (model *SStandaloneAnonResourceBase) GetMetadata(ctx context.Context, key string, userCred mcclient.TokenCredential) string {
	return Metadata.GetStringValue(ctx, model, key, userCred)
}

func (model *SStandaloneAnonResourceBase) GetMetadataJson(ctx context.Context, key string, userCred mcclient.TokenCredential) jsonutils.JSONObject {
	return Metadata.GetJsonValue(ctx, model, key, userCred)
}

func isUserMetadata(key string) bool {
	return strings.HasPrefix(key, USER_TAG_PREFIX)
}

func containsUserMetadata(dict map[string]interface{}) bool {
	for k := range dict {
		if isUserMetadata(k) {
			return true
		}
	}
	return false
}

func (model *SStandaloneAnonResourceBase) SetMetadata(ctx context.Context, key string, value interface{}, userCred mcclient.TokenCredential) error {
	err := Metadata.SetValue(ctx, model, key, value, userCred)
	if err != nil {
		return errors.Wrap(err, "SetValue")
	}
	if isUserMetadata(key) {
		model.GetIStandaloneModel().OnMetadataUpdated(ctx, userCred)
	}
	return nil
}

func (model *SStandaloneAnonResourceBase) SetAllMetadata(ctx context.Context, dictstore map[string]interface{}, userCred mcclient.TokenCredential) error {
	err := Metadata.SetValuesWithLog(ctx, model, dictstore, userCred)
	if err != nil {
		return errors.Wrap(err, "SetValuesWithLog")
	}
	if containsUserMetadata(dictstore) {
		model.GetIStandaloneModel().OnMetadataUpdated(ctx, userCred)
	}
	return nil
}

func (model *SStandaloneAnonResourceBase) SetUserMetadataValues(ctx context.Context, dictstore map[string]interface{}, userCred mcclient.TokenCredential) error {
	err := Metadata.SetValuesWithLog(ctx, model, dictstore, userCred)
	if err != nil {
		return errors.Wrap(err, "SetValuesWithLog")
	}
	model.GetIStandaloneModel().OnMetadataUpdated(ctx, userCred)
	return nil
}

func (model *SStandaloneAnonResourceBase) SetUserMetadataAll(ctx context.Context, dictstore map[string]interface{}, userCred mcclient.TokenCredential) error {
	err := Metadata.SetAll(ctx, model, dictstore, userCred, USER_TAG_PREFIX)
	if err != nil {
		return errors.Wrap(err, "SetAll")
	}
	model.GetIStandaloneModel().OnMetadataUpdated(ctx, userCred)
	return nil
}

func (model *SStandaloneAnonResourceBase) SetCloudMetadataAll(ctx context.Context, dictstore map[string]interface{}, userCred mcclient.TokenCredential) error {
	err := Metadata.SetAll(ctx, model, dictstore, userCred, CLOUD_TAG_PREFIX)
	if err != nil {
		return errors.Wrap(err, "SetAll")
	}
	userTags := map[string]interface{}{}
	for k, v := range dictstore {
		userTags[strings.Replace(k, CLOUD_TAG_PREFIX, USER_TAG_PREFIX, 1)] = v
	}
	return Metadata.SetAll(ctx, model, userTags, userCred, USER_TAG_PREFIX)
}

func (model *SStandaloneAnonResourceBase) SetClassMetadataValues(ctx context.Context, dictstore map[string]interface{}, userCred mcclient.TokenCredential) error {
	err := Metadata.SetValuesWithLog(ctx, model, dictstore, userCred)
	if err != nil {
		return errors.Wrap(err, "SetValuesWithLog")
	}
	return nil
}

func (model *SStandaloneAnonResourceBase) SetClassMetadataAll(ctx context.Context, dictstore map[string]string, userCred mcclient.TokenCredential) error {
	afterCheck := make(map[string]interface{}, len(dictstore))
	for k, v := range dictstore {
		if !strings.HasPrefix(k, CLASS_TAG_PREFIX) {
			afterCheck[CLASS_TAG_PREFIX+k] = v
		} else {
			afterCheck[k] = v
		}
	}
	err := Metadata.SetAll(ctx, model, afterCheck, userCred, CLASS_TAG_PREFIX)
	if err != nil {
		return errors.Wrap(err, "SetAll")
	}
	return nil
}

func (model *SStandaloneAnonResourceBase) InheritTo(ctx context.Context, dest IClassMetadataSetter) error {
	return InheritFromTo(ctx, model, dest)
}

type IClassMetadataSetter interface {
	// a setter should first be a owner
	IClassMetadataOwner

	SetClassMetadataAll(context.Context, map[string]string, mcclient.TokenCredential) error
}

func InheritFromTo(ctx context.Context, src IClassMetadataOwner, dest IClassMetadataSetter) error {
	metadata, err := src.GetAllClassMetadata()
	if err != nil {
		return errors.Wrap(err, "GetAllClassMetadata")
	}
	if len(metadata) == 0 {
		return nil
	}
	curMeta, err := dest.GetAllClassMetadata()
	if err != nil {
		return errors.Wrap(err, "GetAllClassMetadata dest")
	}
	if len(curMeta) > 0 {
		// check conflict
		for k, v := range curMeta {
			if sv, ok := metadata[k]; ok {
				if sv != v {
					// duplicate value for identical key
					return errors.Wrapf(httperrors.ErrConflict, "destination has another value for class key %s", k)
				}
			} else {
				// no such class key
				return errors.Wrapf(httperrors.ErrConflict, "destination has extra class key %s", k)
			}
		}
	}
	userCred := auth.AdminCredential()
	return dest.SetClassMetadataAll(ctx, metadata, userCred)
}

type IClassMetadataOwner interface {
	GetAllClassMetadata() (map[string]string, error)
}

type AllMetadataOwner map[string]string

func (w AllMetadataOwner) GetAllClassMetadata() (map[string]string, error) {
	ret := make(map[string]string)
	for k, v := range w {
		if strings.HasPrefix(k, CLASS_TAG_PREFIX) {
			ret[k[len(CLASS_TAG_PREFIX):]] = v
		}
	}
	return ret, nil
}

type ClassMetadataOwner map[string]string

func (w ClassMetadataOwner) GetAllClassMetadata() (map[string]string, error) {
	return w, nil
}

func IsInSameClass(ctx context.Context, cmo1, cmo2 IClassMetadataOwner) (bool, error) {
	pureTags, err := cmo1.GetAllClassMetadata()
	if err != nil {
		return false, errors.Wrap(err, "GetAllPureMetadata")
	}
	pureTagsP, err := cmo2.GetAllClassMetadata()
	if err != nil {
		return false, errors.Wrap(err, "GetAllPureMetadata")
	}
	if len(pureTags) != len(pureTagsP) {
		return false, nil
	}
	for k, v := range pureTags {
		if vp, ok := pureTagsP[k]; !ok || vp != v {
			return false, nil
		}
	}
	return true, nil
}

func RequireSameClass(ctx context.Context, cmo1, cmo2 IClassMetadataOwner) error {
	same, err := IsInSameClass(ctx, cmo1, cmo2)
	if err != nil {
		return errors.Wrap(err, "IsInSameClass")
	}
	if !same {
		tag1, _ := cmo1.GetAllClassMetadata()
		tag2, _ := cmo2.GetAllClassMetadata()
		return errors.Wrapf(httperrors.ErrConflict, "inconsist class metadata: %s %s", tag1, tag2)
	}
	return nil
}

func (model *SStandaloneAnonResourceBase) IsInSameClass(ctx context.Context, pModel *SStandaloneAnonResourceBase) (bool, error) {
	return IsInSameClass(ctx, model, pModel)
}

func (model *SStandaloneAnonResourceBase) SetSysCloudMetadataAll(ctx context.Context, dictstore map[string]interface{}, userCred mcclient.TokenCredential) error {
	err := Metadata.SetAll(ctx, model, dictstore, userCred, SYS_CLOUD_TAG_PREFIX)
	if err != nil {
		return errors.Wrap(err, "SetAll")
	}
	return nil
}

func (model *SStandaloneAnonResourceBase) RemoveMetadata(ctx context.Context, key string, userCred mcclient.TokenCredential) error {
	err := Metadata.SetValue(ctx, model, key, "", userCred)
	if err != nil {
		return errors.Wrap(err, "SetValue")
	}
	return nil
}

func (model *SStandaloneAnonResourceBase) RemoveAllMetadata(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := Metadata.RemoveAll(ctx, model, userCred)
	if err != nil {
		return errors.Wrap(err, "RemoveAll")
	}
	return nil
}

func (model *SStandaloneAnonResourceBase) GetAllMetadata(ctx context.Context, userCred mcclient.TokenCredential) (map[string]string, error) {
	return Metadata.GetAll(ctx, model, nil, "", userCred)
}

func (model *SStandaloneAnonResourceBase) GetAllUserMetadata() (map[string]string, error) {
	meta, err := Metadata.GetAll(nil, model, nil, USER_TAG_PREFIX, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Metadata.GetAll")
	}
	ret := make(map[string]string)
	for k, v := range meta {
		ret[k[len(USER_TAG_PREFIX):]] = v
	}
	return ret, nil
}

func (model *SStandaloneAnonResourceBase) GetAllCloudMetadata() (map[string]string, error) {
	meta, err := Metadata.GetAll(nil, model, nil, CLOUD_TAG_PREFIX, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Metadata.GetAll")
	}
	ret := make(map[string]string)
	for k, v := range meta {
		ret[k[len(CLOUD_TAG_PREFIX):]] = v
	}
	return ret, nil
}

func (model *SStandaloneAnonResourceBase) GetAllClassMetadata() (map[string]string, error) {
	meta, err := Metadata.GetAll(nil, model, nil, CLASS_TAG_PREFIX, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Metadata.GetAll")
	}
	ret := make(map[string]string)
	for k, v := range meta {
		ret[k[len(CLASS_TAG_PREFIX):]] = v
	}
	return ret, nil
}

// 获取资源标签（元数据）
func (model *SStandaloneAnonResourceBase) GetDetailsMetadata(ctx context.Context, userCred mcclient.TokenCredential, input apis.GetMetadataInput) (apis.GetMetadataOutput, error) {
	val, err := Metadata.GetAll(ctx, model, input.Field, input.Prefix, userCred)
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

// +onecloud:swagger-gen-ignore
func (model *SStandaloneAnonResourceBase) PerformMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformMetadataInput) (jsonutils.JSONObject, error) {
	dictStore := make(map[string]interface{})
	for k, v := range input {
		// 已双下滑线开头的metadata是系统内置，普通用户不可添加，只能查看
		if strings.HasPrefix(k, SYS_TAG_PREFIX) && (userCred == nil || !IsAllowPerform(ctx, rbacscope.ScopeSystem, userCred, model, "metadata")) {
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

// 更新资源的用户标签
// +onecloud:swagger-gen-ignore
func (model *SStandaloneAnonResourceBase) PerformUserMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformUserMetadataInput) (jsonutils.JSONObject, error) {
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

// 全量替换资源的所有用户标签
func (model *SStandaloneAnonResourceBase) PerformSetUserMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformSetUserMetadataInput) (jsonutils.JSONObject, error) {
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

// 更新资源的 class 标签
func (model *SStandaloneAnonResourceBase) PerformClassMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformClassMetadataInput) (jsonutils.JSONObject, error) {
	dictStore := make(map[string]interface{})
	for k, v := range input {
		dictStore[CLASS_TAG_PREFIX+k] = v
	}
	err := model.SetUserMetadataValues(ctx, dictStore, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "SetUserMetadataValues")
	}
	return nil, nil
}

// 全量替换资源的所有 class 标签
func (model *SStandaloneAnonResourceBase) PerformSetClassMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformSetClassMetadataInput) (jsonutils.JSONObject, error) {
	dictStore := make(map[string]string)
	for k, v := range input {
		if len(k) > 64-len(CLASS_TAG_PREFIX) {
			return nil, httperrors.NewInputParameterError("input key too long > %d", 64-len(CLASS_TAG_PREFIX))
		}
		if len(v) > 65535 {
			return nil, httperrors.NewInputParameterError("input value too long > %d", 65535)
		}
		dictStore[k] = v
	}
	err := model.SetClassMetadataAll(ctx, dictStore, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "SetUserMetadataAll")
	}
	return nil, nil
}

func (model *SStandaloneAnonResourceBase) GetDetailsClassMetadata(ctx context.Context, userCred mcclient.TokenCredential, input apis.GetClassMetadataInput) (apis.GetClassMetadataOutput, error) {
	return model.GetAllClassMetadata()
}

type sPolicyTags struct {
	PolicyObjectTags  tagutils.TTagSetList `json:"policy_object_tags"`
	PolicyProjectTags tagutils.TTagSetList `json:"policy_project_tags"`
	PolicyDomainTags  tagutils.TTagSetList `json:"policy_domain_tags"`
}

func (model *SStandaloneAnonResourceBase) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	model.SResourceBase.PostUpdate(ctx, userCred, query, data)

	meta := make(map[string]string)
	err := data.Unmarshal(&meta, "__meta__")
	if err == nil {
		model.PerformMetadata(ctx, userCred, nil, meta)
	}

	model.applyPolicyTags(ctx, userCred, data)
}

func (model *SStandaloneAnonResourceBase) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	model.SResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

	meta := make(map[string]string)
	err := data.Unmarshal(&meta, "__meta__")
	if err == nil {
		model.PerformMetadata(ctx, userCred, nil, meta)
	}

	model.applyPolicyTags(ctx, userCred, data)
}

func (model *SStandaloneAnonResourceBase) applyPolicyTags(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) {
	tags := sPolicyTags{}
	data.Unmarshal(&tags)
	log.Debugf("applyPolicyTags: %s", jsonutils.Marshal(tags))
	if len(tags.PolicyObjectTags) > 0 {
		model.PerformMetadata(ctx, userCred, nil, tagutils.Tagset2MapString(tags.PolicyObjectTags.Flattern()))
	}
	if model.Keyword() == "project" && len(tags.PolicyProjectTags) > 0 {
		model.PerformMetadata(ctx, userCred, nil, tagutils.Tagset2MapString(tags.PolicyProjectTags.Flattern()))
	} else if model.Keyword() == "domain" && len(tags.PolicyDomainTags) > 0 {
		model.PerformMetadata(ctx, userCred, nil, tagutils.Tagset2MapString(tags.PolicyDomainTags.Flattern()))
	}
}

func (model *SStandaloneAnonResourceBase) ClearSchedDescCache() error {
	return nil
}

func (model *SStandaloneAnonResourceBase) AppendDescription(userCred mcclient.TokenCredential, msg string) error {
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

func (manager *SStandaloneAnonResourceBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input apis.StandaloneAnonResourceCreateInput) (apis.StandaloneAnonResourceCreateInput, error) {
	var err error
	input.ResourceBaseCreateInput, err = manager.SResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.ResourceBaseCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SResourceBaseManager.ValidateCreateData")
	}
	return input, nil
}

func (manager *SStandaloneAnonResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.StandaloneAnonResourceDetails {
	ret := make([]apis.StandaloneAnonResourceDetails, len(objs))
	upperRet := manager.SResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	metaRet := manager.SMetadataResourceBaseModelManager.FetchCustomizeColumns(manager.GetIModelManager(), userCred, objs, fields)
	for i := range objs {
		ret[i] = apis.StandaloneAnonResourceDetails{
			ResourceBaseDetails:  upperRet[i],
			MetadataResourceInfo: metaRet[i],
		}
	}
	return ret
}

func (manager *SStandaloneAnonResourceBaseManager) GetMetadataHiddenKeys() []string {
	return nil
}

func (manager *SStandaloneAnonResourceBaseManager) GetExportExtraKeys(ctx context.Context, keys stringutils2.SSortedStrings, rowMap map[string]string) *jsonutils.JSONDict {
	res := manager.SResourceBaseManager.GetExportExtraKeys(ctx, keys, rowMap)

	metaRes := manager.SMetadataResourceBaseModelManager.GetExportExtraKeys(keys, rowMap)
	if metaRes.Length() > 0 {
		res.Update(metaRes)
	}

	return res
}

func (manager *SStandaloneAnonResourceBaseManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SResourceBaseManager.ListItemExportKeys")
	}

	q = manager.SMetadataResourceBaseModelManager.ListItemExportKeys(manager.GetIModelManager(), q, keys)

	return q, nil
}

func (model *SStandaloneAnonResourceBase) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.StandaloneAnonResourceBaseUpdateInput) (apis.StandaloneAnonResourceBaseUpdateInput, error) {
	var err error
	input.ResourceBaseUpdateInput, err = model.SResourceBase.ValidateUpdateData(ctx, userCred, query, input.ResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SModelBase.ValidateUpdateData")
	}
	return input, nil
}

func (model *SStandaloneAnonResourceBase) IsShared() bool {
	return false
}

func (model *SStandaloneAnonResourceBase) OnMetadataUpdated(ctx context.Context, userCred mcclient.TokenCredential) {
	// noop
}

type SGetResourceTagValuePairsInput struct {
	apis.MetadataBaseFilterInput

	// Return keys only
	KeyOnly *bool `json:"key_only"`

	// Order by key of tags
	OrderByTagKey string `json:"order_by_tag_key"`
}

func (manager *SStandaloneAnonResourceBaseManager) GetPropertyTagValuePairs(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	return GetPropertyTagValuePairs(
		manager.GetIStandaloneModelManager(),
		manager.Keyword(),
		"id",
		ctx,
		userCred,
		query,
	)
}

func GetPropertyTagValuePairs(
	manager IModelManager,
	tagObjType string,
	tagIdField string,
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	input := SGetResourceTagValuePairsInput{}
	err := query.Unmarshal(&input)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}

	sq := Metadata.Query().SubQuery()
	keyOnly := (input.KeyOnly != nil && *input.KeyOnly)

	queryKeys := []string{tagIdField}
	if tagIdField != "id" {
		queryKeys = append(queryKeys, "id")
	}
	objQ := manager.Query(queryKeys...)
	objQ, err = ListItemQueryFilters(manager, ctx, objQ, userCred, query, policy.PolicyActionList)
	if err != nil {
		return nil, errors.Wrap(err, "ListItemQueryFilters")
	}
	objSQ := objQ.SubQuery()

	var queryFields []sqlchemy.IQueryField
	if keyOnly {
		queryFields = []sqlchemy.IQueryField{
			sq.Field("key"),
			sqlchemy.COUNT("count", objSQ.Field("id")),
		}
	} else {
		queryFields = []sqlchemy.IQueryField{
			sq.Field("key"),
			sq.Field("value"),
			sqlchemy.COUNT("count", objSQ.Field("id")),
		}
	}
	q := sq.Query()

	q = q.Join(objSQ, sqlchemy.AND(
		sqlchemy.Equals(q.Field("obj_type"), tagObjType),
		sqlchemy.Equals(q.Field("obj_id"), objSQ.Field(tagIdField)),
	))

	q = q.AppendField(queryFields...)

	q = Metadata.metadataBaseFilter(q, input.MetadataBaseFilterInput)

	if keyOnly {
		q = q.GroupBy(q.Field("key"))
	} else {
		q = q.GroupBy(q.Field("key"), q.Field("value"))
	}
	if input.OrderByTagKey == string(sqlchemy.SQL_ORDER_DESC) {
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

	metadatas := make([]struct {
		Key   string
		Value string
		Count int64
	}, 0)
	err = q.All(&metadatas)
	if err != nil {
		return nil, errors.Wrap(err, "Query.All")
	}

	return jsonutils.Marshal(metadatas), nil
}

type SGetResourceTagValueTreeInput struct {
	Keys    []string `json:"key"`
	ShowMap *bool    `json:"show_map"`
}

func (manager *SStandaloneAnonResourceBaseManager) GetPropertyTagValueTree(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	return GetPropertyTagValueTree(
		manager.GetIStandaloneModelManager(),
		manager.Keyword(),
		"id",
		ctx,
		userCred,
		query,
	)
}

func GetPropertyTagValueTree(
	manager IModelManager,
	tagObjType string,
	tagIdField string,
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	input := SGetResourceTagValueTreeInput{}
	err := query.Unmarshal(&input)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}

	objSubQ := manager.Query().SubQuery()
	objQ := objSubQ.Query(objSubQ.Field(tagIdField), sqlchemy.COUNT("_sub_count_"))
	objQ, err = ListItemQueryFilters(manager, ctx, objQ, userCred, query, policy.PolicyActionList)
	if err != nil {
		return nil, errors.Wrap(err, "ListItemQueryFilters")
	}
	objQ = objQ.GroupBy(objSubQ.Field(tagIdField))
	q := objQ.SubQuery().Query(sqlchemy.SUM(tagValueCountKey, objQ.Field("_sub_count_")))
	metadataSQ := Metadata.Query().Equals("obj_type", tagObjType).In("key", input.Keys).SubQuery()
	groupBy := make([]interface{}, 0)
	for i, key := range input.Keys {
		valueFieldName := tagValueKey(i)
		subq := metadataSQ.Query().Equals("key", key).SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field(tagIdField), subq.Field("obj_id")))
		q = q.AppendField(
			sqlchemy.NewFunction(
				sqlchemy.NewCase().When(sqlchemy.IsNull(subq.Field("value")), sqlchemy.NewStringField(tagutils.NoValue)).Else(subq.Field("value")),
				valueFieldName,
			),
		)
		groupBy = append(groupBy, q.Field(valueFieldName))
	}
	q = q.GroupBy(groupBy...)
	valueMap, err := q.AllStringMap()
	if err != nil {
		return nil, errors.Wrap(err, "AllStringAmp")
	}

	if input.ShowMap != nil && *input.ShowMap {
		return jsonutils.Marshal(valueMap), nil
	}

	tree := constructTree(valueMap, input.Keys)
	return jsonutils.Marshal(tree), nil
}
