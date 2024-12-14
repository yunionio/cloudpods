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

package models

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SchedStrategyType string

const (
	STRATEGY_REQUIRE = api.STRATEGY_REQUIRE
	STRATEGY_EXCLUDE = api.STRATEGY_EXCLUDE
	STRATEGY_PREFER  = api.STRATEGY_PREFER
	STRATEGY_AVOID   = api.STRATEGY_AVOID

	// # container used aggregate
	CONTAINER_AGGREGATE = api.CONTAINER_AGGREGATE
)

var STRATEGY_LIST = api.STRATEGY_LIST

type ISchedtagJointManager interface {
	db.IJointModelManager
	GetResourceIdKey(db.IJointModelManager) string
}

type ISchedtagJointModel interface {
	db.IJointModel
	GetSchedtagId() string
	GetResourceId() string
	GetDetails(base api.SchedtagJointResourceDetails, resourceName string, isList bool) interface{}
}

type IModelWithSchedtag interface {
	db.IModel
	GetSchedtagJointManager() ISchedtagJointManager
	ClearSchedDescCache() error
}

type SSchedtagManager struct {
	db.SStandaloneResourceBaseManager
	db.SScopedResourceBaseManager

	jointsManager map[string]ISchedtagJointManager
}

var SchedtagManager *SSchedtagManager

func init() {
	SchedtagManager = &SSchedtagManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SSchedtag{},
			"aggregates_tbl",
			"schedtag",
			"schedtags",
		),
		jointsManager: make(map[string]ISchedtagJointManager),
	}
	SchedtagManager.SetVirtualObject(SchedtagManager)
}

func (manager *SSchedtagManager) InitializeData() error {
	// set old schedtags resource_type to hosts
	schedtags := []SSchedtag{}
	q := manager.Query().IsNullOrEmpty("resource_type")
	err := db.FetchModelObjects(manager, q, &schedtags)
	if err != nil {
		return err
	}
	for _, tag := range schedtags {
		tmp := &tag
		db.Update(tmp, func() error {
			tmp.ResourceType = HostManager.KeywordPlural()
			return nil
		})
	}
	manager.BindJointManagers(map[db.IModelManager]ISchedtagJointManager{
		HostManager:          HostschedtagManager,
		StorageManager:       StorageschedtagManager,
		NetworkManager:       NetworkschedtagManager,
		CloudproviderManager: CloudproviderschedtagManager,
		ZoneManager:          ZoneschedtagManager,
		CloudregionManager:   CloudregionschedtagManager,
	})
	return nil
}

func (manager *SSchedtagManager) NamespaceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeSystem
}

func (manager *SSchedtagManager) BindJointManagers(ms map[db.IModelManager]ISchedtagJointManager) {
	for m, schedtagM := range ms {
		manager.jointsManager[m.KeywordPlural()] = schedtagM
	}
}

func (manager *SSchedtagManager) GetResourceTypes() []string {
	ret := []string{}
	for key := range manager.jointsManager {
		ret = append(ret, key)
	}
	return ret
}

func (manager *SSchedtagManager) GetJointManager(resTypePlural string) ISchedtagJointManager {
	return manager.jointsManager[resTypePlural]
}

type SSchedtag struct {
	db.SStandaloneResourceBase
	db.SScopedResourceBase

	DefaultStrategy string `width:"16" charset:"ascii" nullable:"true" default:"" list:"user" update:"admin" create:"admin_optional"` // Column(VARCHAR(16, charset='ascii'), nullable=True, default='')
	ResourceType    string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"required"`                                 // Column(VARCHAR(16, charset='ascii'), nullable=True, default='')
}

func (m *SSchedtagManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	if ownerId == nil {
		return q
	}
	switch scope {
	case rbacscope.ScopeDomain:
		q = q.Filter(sqlchemy.OR(
			// share to system
			sqlchemy.IsNullOrEmpty(q.Field("domain_id")),
			// share to this domain or its sub-projects
			sqlchemy.Equals(q.Field("domain_id"), ownerId.GetProjectDomainId()),
		))
	case rbacscope.ScopeProject:
		q = q.Filter(sqlchemy.OR(
			// share to system
			sqlchemy.AND(
				sqlchemy.IsNullOrEmpty(q.Field("domain_id")),
				sqlchemy.IsNullOrEmpty(q.Field("tenant_id")),
			),
			// share to project's parent domain
			sqlchemy.AND(
				sqlchemy.Equals(q.Field("domain_id"), ownerId.GetProjectDomainId()),
				sqlchemy.IsNullOrEmpty(q.Field("tenant_id")),
			),
			// share to this project
			sqlchemy.AND(
				sqlchemy.Equals(q.Field("domain_id"), ownerId.GetProjectDomainId()),
				sqlchemy.Equals(q.Field("tenant_id"), ownerId.GetProjectId()),
			),
		))
	}
	return q
}

func (manager *SSchedtagManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	return db.ApplyListItemExportKeys(ctx, q, userCred, keys,
		&manager.SStandaloneResourceBaseManager,
		&manager.SScopedResourceBaseManager,
	)
}

// 调度标签列表
func (manager *SSchedtagManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.SchedtagListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SScopedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ScopedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SScopedResourceBaseManager.ListItemFilter")
	}

	if len(query.ResourceType) > 0 {
		q = q.In("resource_type", query.ResourceType)
	}

	if len(query.DefaultStrategy) > 0 {
		q = q.In("default_strategy", query.DefaultStrategy)
	}

	if len(query.CloudproviderId) > 0 {
		hostSubq := HostManager.Query("id").Equals("manager_id", query.CloudproviderId).SubQuery()
		hostSchedtagQ := HostschedtagManager.Query("schedtag_id")
		hostSchedtagSubq := hostSchedtagQ.Join(hostSubq, sqlchemy.Equals(hostSchedtagQ.Field("host_id"), hostSubq.Field("id"))).SubQuery()
		q = q.Join(hostSchedtagSubq, sqlchemy.Equals(q.Field("id"), hostSchedtagSubq.Field("schedtag_id")))
	}

	return q, nil
}

func (manager *SSchedtagManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.SchedtagListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SScopedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ScopedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SScopedResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SSchedtagManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SScopedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SSchedtagManager) ValidateSchedtags(ctx context.Context, userCred mcclient.TokenCredential, schedtags []*api.SchedtagConfig) ([]*api.SchedtagConfig, error) {
	ret := make([]*api.SchedtagConfig, len(schedtags))
	for idx, tag := range schedtags {
		schedtagObj, err := manager.FetchByIdOrName(ctx, userCred, tag.Id)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError("Invalid schedtag %s", tag.Id)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		strategy := strings.ToLower(tag.Strategy)
		schedtag := schedtagObj.(*SSchedtag)
		if !utils.IsInStringArray(strategy, STRATEGY_LIST) {
			return nil, httperrors.NewInputParameterError("invalid strategy %s", strategy)
		}
		tag.Id = schedtag.GetId()
		tag.ResourceType = schedtag.ResourceType
		ret[idx] = tag
	}
	return ret, nil
}

func validateDefaultStrategy(defStrategy string) error {
	if !utils.IsInStringArray(defStrategy, STRATEGY_LIST) {
		return httperrors.NewInputParameterError("Invalid default stragegy %s", defStrategy)
	}
	if defStrategy == STRATEGY_REQUIRE {
		return httperrors.NewInputParameterError("Cannot set default strategy of %s", defStrategy)
	}
	return nil
}

func (manager *SSchedtagManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.SchedtagCreateInput) (*jsonutils.JSONDict, error) {
	if len(input.DefaultStrategy) > 0 {
		err := validateDefaultStrategy(input.DefaultStrategy)
		if err != nil {
			return nil, err
		}
	}
	// set resourceType to hosts if not provided by client
	if input.ResourceType == "" {
		input.ResourceType = HostManager.KeywordPlural()
	}
	if !utils.IsInStringArray(input.ResourceType, manager.GetResourceTypes()) {
		return nil, httperrors.NewInputParameterError("Not support resource_type %s", input.ResourceType)
	}

	var err error
	input.ScopedResourceCreateInput, err = manager.SScopedResourceBaseManager.ValidateCreateData(manager, ctx, userCred, ownerId, query, input.ScopedResourceCreateInput)
	if err != nil {
		return nil, err
	}

	input.StandaloneResourceCreateInput, err = manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StandaloneResourceCreateInput)
	if err != nil {
		return nil, err
	}

	return input.JSON(input), nil
}

func (manager *SSchedtagManager) GetResourceSchedtags(resType string) ([]SSchedtag, error) {
	jointMan := manager.jointsManager[resType]
	if jointMan == nil {
		return nil, fmt.Errorf("Not found joint manager by resource type: %s", resType)
	}
	tags := make([]SSchedtag, 0)
	if err := manager.Query().Equals("resource_type", resType).All(&tags); err != nil {
		return nil, err
	}
	return tags, nil
}

func (self *SSchedtag) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.SScopedResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (self *SSchedtag) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	defStrategy, _ := data.GetString("default_strategy")
	if len(defStrategy) > 0 {
		err := validateDefaultStrategy(defStrategy)
		if err != nil {
			return nil, err
		}
	}
	input := apis.StandaloneResourceBaseUpdateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	input, err = self.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBase.ValidateUpdateData")
	}
	data.Update(jsonutils.Marshal(input))

	return data, nil
}

func (self *SSchedtag) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	cnt, err := self.GetObjectCount()
	if err != nil {
		return httperrors.NewInternalServerError("GetObjectCount fail %s", err)
	}
	if cnt > 0 {
		return httperrors.NewNotEmptyError("Tag is associated with %s", self.ResourceType)
	}
	cnt, err = self.getDynamicSchedtagCount()
	if err != nil {
		return httperrors.NewInternalServerError("getDynamicSchedtagCount fail %s", err)
	}
	if cnt > 0 {
		return httperrors.NewNotEmptyError("tag has dynamic rules")
	}
	cnt, err = self.getSchedPoliciesCount()
	if err != nil {
		return httperrors.NewInternalServerError("getSchedPoliciesCount fail %s", err)
	}
	if cnt > 0 {
		return httperrors.NewNotEmptyError("tag is associate with sched policies")
	}
	return self.SStandaloneResourceBase.ValidateDeleteCondition(ctx, nil)
}

// GetObjectPtr wraps the given value with pointer: V => *V, *V => **V, etc.
func GetObjectPtr(obj interface{}) interface{} {
	v := reflect.ValueOf(obj)
	pt := reflect.PtrTo(v.Type())
	pv := reflect.New(pt.Elem())
	pv.Elem().Set(v)

	return pv.Interface()
}

func (self *SSchedtag) GetResources() ([]IModelWithSchedtag, error) {
	objs := make([]interface{}, 0)
	q := self.GetObjectQuery()
	masterMan, err := self.GetResourceManager()
	if err != nil {
		return nil, err
	}
	if err := db.FetchModelObjects(masterMan, q, &objs); err != nil {
		return nil, err
	}

	ret := make([]IModelWithSchedtag, len(objs))
	for i := range objs {
		obj := objs[i]
		ret[i] = GetObjectPtr(obj).(IModelWithSchedtag)
	}
	return ret, nil
}

func (self *SSchedtag) GetObjectQuery() *sqlchemy.SQuery {
	jointMan := self.GetJointManager()
	masterMan := jointMan.GetMasterManager()
	objs := masterMan.Query().SubQuery()
	objschedtags := jointMan.Query().SubQuery()
	q := objs.Query()
	q = q.Join(objschedtags, sqlchemy.AND(sqlchemy.Equals(objschedtags.Field(jointMan.GetResourceIdKey(jointMan)), objs.Field("id")),
		sqlchemy.IsFalse(objschedtags.Field("deleted"))))
	// q = q.Filter(sqlchemy.IsTrue(objs.Field("enabled")))
	q = q.Filter(sqlchemy.Equals(objschedtags.Field("schedtag_id"), self.Id))
	return q
}

func (self *SSchedtag) GetJointManager() ISchedtagJointManager {
	return SchedtagManager.GetJointManager(self.ResourceType)
}

func (s *SSchedtag) GetResourceManager() (db.IStandaloneModelManager, error) {
	jResMan := s.GetJointManager()
	if jResMan == nil {
		return nil, errors.Errorf("Not found bind joint resource manager by type %q", s.ResourceType)
	}

	return jResMan.GetMasterManager(), nil
}

func (self *SSchedtag) GetObjectCount() (int, error) {
	q := self.GetObjectQuery()
	return q.CountWithError()
}

func (self *SSchedtag) getSchedPoliciesCount() (int, error) {
	return SchedpolicyManager.Query().Equals("schedtag_id", self.Id).CountWithError()
}

func (self *SSchedtag) getDynamicSchedtagCount() (int, error) {
	return DynamicschedtagManager.Query().Equals("schedtag_id", self.Id).CountWithError()
}

func (self *SSchedtag) getMoreColumns(out api.SchedtagDetails) api.SchedtagDetails {
	out.ProjectId = self.SScopedResourceBase.ProjectId
	cnt, _ := self.GetObjectCount()
	keyword := self.GetJointManager().GetMasterManager().Keyword()
	switch keyword {
	case HostManager.Keyword():
		out.HostCount = cnt
	case GuestManager.Keyword():
		out.ServerCount = cnt
	default:
		out.OtherCount = cnt
		out.JoinModelKeyword = keyword
	}
	out.DynamicSchedtagCount, _ = self.getDynamicSchedtagCount()
	out.SchedpolicyCount, _ = self.getSchedPoliciesCount()

	// resource_count = row.host_count || row.other_count || '0'
	if out.HostCount > 0 {
		out.ResourceCount = out.HostCount
	} else if out.OtherCount > 0 {
		out.ResourceCount = out.OtherCount
	} else {
		out.ResourceCount = 0
	}
	return out
}

func (manager *SSchedtagManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.SchedtagDetails {
	rows := make([]api.SchedtagDetails, len(objs))

	stdRows := manager.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	scopedRows := manager.SScopedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.SchedtagDetails{
			StandaloneResourceDetails: stdRows[i],
			ScopedResourceBaseInfo:    scopedRows[i],
		}
		rows[i] = objs[i].(*SSchedtag).getMoreColumns(rows[i])
	}

	return rows
}

/*func (self *SSchedtag) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)
}*/

func (self *SSchedtag) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := self.SStandaloneResourceBase.GetShortDesc(ctx)
	desc.Add(jsonutils.NewString(self.DefaultStrategy), "default")
	return desc
}

func (self *SSchedtag) GetShortDescV2(ctx context.Context) api.SchedtagShortDescDetails {
	desc := api.SchedtagShortDescDetails{}
	desc.StandaloneResourceShortDescDetail = self.SStandaloneResourceBase.GetShortDescV2(ctx)
	desc.Default = self.DefaultStrategy
	return desc
}

func GetResourceJointSchedtags(obj IModelWithSchedtag) ([]ISchedtagJointModel, error) {
	jointMan := obj.GetSchedtagJointManager()
	q := jointMan.Query().Equals(jointMan.GetResourceIdKey(jointMan), obj.GetId())
	jointTags := make([]ISchedtagJointModel, 0)
	rows, err := q.Rows()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		item, err := db.NewModelObject(jointMan)
		if err != nil {
			return nil, err
		}
		err = q.Row2Struct(rows, item)
		if err != nil {
			return nil, err
		}
		jointTags = append(jointTags, item.(ISchedtagJointModel))
	}

	return jointTags, nil
}

func GetSchedtags(jointMan ISchedtagJointManager, masterId string) []SSchedtag {
	tags := make([]SSchedtag, 0)
	schedtags := SchedtagManager.Query().SubQuery()
	objschedtags := jointMan.Query().SubQuery()
	q := schedtags.Query()
	q = q.Join(objschedtags, sqlchemy.AND(sqlchemy.Equals(objschedtags.Field("schedtag_id"), schedtags.Field("id")),
		sqlchemy.IsFalse(objschedtags.Field("deleted"))))
	q = q.Filter(sqlchemy.Equals(objschedtags.Field(jointMan.GetResourceIdKey(jointMan)), masterId))
	err := db.FetchModelObjects(SchedtagManager, q, &tags)
	if err != nil {
		log.Errorf("GetSchedtags error: %s", err)
		return nil
	}
	return tags
}

func PerformSetResourceSchedtag(obj IModelWithSchedtag, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	schedtags := jsonutils.GetArrayOfPrefix(data, "schedtag")
	setTagsId := []string{}
	for idx := 0; idx < len(schedtags); idx++ {
		schedtagIdent, _ := schedtags[idx].GetString()
		tag, err := SchedtagManager.FetchByIdOrName(ctx, userCred, schedtagIdent)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewNotFoundError("Schedtag %s not found", schedtagIdent)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		schedtag := tag.(*SSchedtag)
		if schedtag.ResourceType != obj.KeywordPlural() {
			return nil, httperrors.NewInputParameterError("Schedtag %s ResourceType is %s, not match %s", schedtag.GetName(), schedtag.ResourceType, obj.KeywordPlural())
		}
		setTagsId = append(setTagsId, schedtag.GetId())
	}
	oldTags, err := GetResourceJointSchedtags(obj)
	if err != nil {
		return nil, httperrors.NewGeneralError(fmt.Errorf("Get old joint schedtags: %v", err))
	}
	for _, oldTag := range oldTags {
		if !utils.IsInStringArray(oldTag.GetSchedtagId(), setTagsId) {
			if err := oldTag.Detach(ctx, userCred); err != nil {
				return nil, httperrors.NewGeneralError(err)
			}
		}
	}
	var oldTagIds []string
	for _, tag := range oldTags {
		oldTagIds = append(oldTagIds, tag.GetSchedtagId())
	}
	jointMan := obj.GetSchedtagJointManager()
	for _, setTagId := range setTagsId {
		if !utils.IsInStringArray(setTagId, oldTagIds) {
			if _, err := InsertJointResourceSchedtag(ctx, jointMan, obj.GetId(), setTagId); err != nil {
				return nil, errors.Wrapf(err, "InsertJointResourceSchedtag %s %s", obj.GetId(), setTagId)
			}
		}
	}
	if err := obj.ClearSchedDescCache(); err != nil {
		log.Errorf("Resource %s/%s ClearSchedDescCache error: %v", obj.Keyword(), obj.GetId(), err)
	}
	logclient.AddActionLogWithContext(ctx, obj, logclient.ACT_SET_SCHED_TAG, nil, userCred, true)
	return nil, nil
}

func DeleteResourceJointSchedtags(obj IModelWithSchedtag, ctx context.Context, userCred mcclient.TokenCredential) error {
	jointTags, err := GetResourceJointSchedtags(obj)
	if err != nil {
		return fmt.Errorf("Get %s schedtags error: %v", obj.Keyword(), err)
	}
	for _, tag := range jointTags {
		tag.Delete(ctx, userCred)
	}
	return nil
}

func GetSchedtagsDetailsToResource(obj IModelWithSchedtag, ctx context.Context, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	schedtags := GetSchedtags(obj.GetSchedtagJointManager(), obj.GetId())
	if schedtags != nil && len(schedtags) > 0 {
		info := make([]jsonutils.JSONObject, len(schedtags))
		for i := 0; i < len(schedtags); i += 1 {
			info[i] = schedtags[i].GetShortDesc(ctx)
		}
		extra.Add(jsonutils.NewArray(info...), "schedtags")
	}
	return extra
}

func GetSchedtagsDetailsToResourceV2(obj IModelWithSchedtag, ctx context.Context) []api.SchedtagShortDescDetails {
	info := []api.SchedtagShortDescDetails{}
	schedtags := GetSchedtags(obj.GetSchedtagJointManager(), obj.GetId())
	if schedtags != nil && len(schedtags) > 0 {
		for i := 0; i < len(schedtags); i += 1 {
			desc := schedtags[i].GetShortDescV2(ctx)
			info = append(info, desc)
		}
	}
	return info
}

func (s *SSchedtag) PerformSetScope(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return db.PerformSetScope(ctx, s, userCred, data)
}

func (s *SSchedtag) GetJointResourceTag(resId string) (ISchedtagJointModel, error) {
	jMan := s.GetJointManager()
	jObj, err := db.FetchJointByIds(jMan, resId, s.GetId(), nil)
	if err != nil {
		return nil, err
	}
	return jObj.(ISchedtagJointModel), nil
}

func (s *SSchedtag) PerformSetResource(ctx context.Context, userCred mcclient.TokenCredential, _ jsonutils.JSONObject, input *api.SchedtagSetResourceInput) (jsonutils.JSONObject, error) {
	if input == nil {
		return nil, nil
	}

	setResIds := make(map[string]IModelWithSchedtag, 0)
	resMan, err := s.GetResourceManager()
	if err != nil {
		return nil, errors.Wrap(err, "get resource manager")
	}

	// get need set resource ids
	for i := 0; i < len(input.ResourceIds); i++ {
		resId := input.ResourceIds[i]
		res, err := resMan.FetchByIdOrName(ctx, userCred, resId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewNotFoundError("Resource %s %s not found", s.ResourceType, resId)
			}
			return nil, errors.Wrapf(err, "Fetch resource %s by id or name", s.ResourceType)
		}
		setResIds[res.GetId()] = res.(IModelWithSchedtag)
	}

	// unbind current resources if not in input
	curRess, err := s.GetResources()
	if err != nil {
		return nil, errors.Wrap(err, "Get current bind resources")
	}
	curResIds := make([]string, 0)
	for i := range curRess {
		res := curRess[i]
		if _, ok := setResIds[res.GetId()]; ok {
			curResIds = append(curResIds, res.GetId())
			continue
		}
		jObj, err := s.GetJointResourceTag(res.GetId())
		if err != nil {
			return nil, errors.Wrapf(err, "Get joint resource tag by id %s", res.GetId())
		}
		if err := jObj.Detach(ctx, userCred); err != nil {
			return nil, errors.Wrap(err, "detach joint tag")
		}
	}

	// bind input resources
	jointMan := s.GetJointManager()
	for resId := range setResIds {
		if utils.IsInStringArray(resId, curResIds) {
			// already binded
			continue
		}

		if _, err := InsertJointResourceSchedtag(ctx, jointMan, resId, s.GetId()); err != nil {
			return nil, errors.Wrapf(err, "InsertJointResourceSchedtag %s %s", resId, s.GetId())
		}

		res := setResIds[resId]
		if err := res.ClearSchedDescCache(); err != nil {
			log.Errorf("Resource %s/%s ClearSchedDescCache error: %v", res.Keyword(), res.GetId(), err)
		}
	}

	return nil, nil
}
