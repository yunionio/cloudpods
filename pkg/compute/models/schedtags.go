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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
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
	GetMasterIdKey(db.IJointModelManager) string
}

type ISchedtagJointModel interface {
	db.IJointModel
	GetSchedtagId() string
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
		HostManager:    HostschedtagManager,
		StorageManager: StorageschedtagManager,
		NetworkManager: NetworkschedtagManager,
	})
	return nil
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

type SSchedtag struct {
	db.SStandaloneResourceBase
	db.SScopedResourceBase

	DefaultStrategy string `width:"16" charset:"ascii" nullable:"true" default:"" list:"user" update:"admin" create:"admin_optional"` // Column(VARCHAR(16, charset='ascii'), nullable=True, default='')
	ResourceType    string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"required"`                                 // Column(VARCHAR(16, charset='ascii'), nullable=True, default='')
}

func (manager *SSchedtagManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (manager *SSchedtagManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	if resType := jsonutils.GetAnyString(query, []string{"type", "resource_type"}); resType != "" {
		q = q.Equals("resource_type", resType)
	}
	return manager.SResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
}

func (self *SSchedtag) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SSchedtagManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SSchedtag) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SSchedtag) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (manager *SSchedtagManager) ValidateSchedtags(userCred mcclient.TokenCredential, schedtags map[string]string) (map[string]string, error) {
	ret := make(map[string]string)
	for tag, act := range schedtags {
		schedtagObj, err := manager.FetchByIdOrName(nil, tag)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError("Invalid schedtag %s", tag)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		act = strings.ToLower(act)
		schedtag := schedtagObj.(*SSchedtag)
		if !utils.IsInStringArray(act, STRATEGY_LIST) {
			return nil, httperrors.NewInputParameterError("invalid strategy %s", act)
		}
		ret[schedtag.Name] = act
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

func (manager *SSchedtagManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	defStrategy, _ := data.GetString("default_strategy")
	if len(defStrategy) > 0 {
		err := validateDefaultStrategy(defStrategy)
		if err != nil {
			return nil, err
		}
	}
	// set resourceType to hosts if not provided by client
	resourceType, _ := data.GetString("resource_type")
	if resourceType == "" {
		resourceType = HostManager.KeywordPlural()
		data.Set("resource_type", jsonutils.NewString(resourceType))
	}
	if !utils.IsInStringArray(resourceType, manager.GetResourceTypes()) {
		return nil, httperrors.NewInputParameterError("Not support resource_type %s", resourceType)
	}
	return manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
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

func (self *SSchedtag) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	defStrategy, _ := data.GetString("default_strategy")
	if len(defStrategy) > 0 {
		err := validateDefaultStrategy(defStrategy)
		if err != nil {
			return nil, err
		}
	}
	return self.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (self *SSchedtag) ValidateDeleteCondition(ctx context.Context) error {
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
	return self.SStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

/*
func (self *SSchedtag) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return userCred.IsSystemAdmin()
}

func (self *SSchedtag) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return userCred.IsSystemAdmin()
}*/

func (self *SSchedtag) GetObjects(objs interface{}) error {
	q := self.GetObjectQuery()
	masterMan := self.GetJointManager().GetMasterManager()
	err := db.FetchModelObjects(masterMan, q, objs)
	if err != nil {
		return err
	}
	return nil
}

func (self *SSchedtag) GetObjectQuery() *sqlchemy.SQuery {
	jointMan := self.GetJointManager()
	masterMan := jointMan.GetMasterManager()
	objs := masterMan.Query().SubQuery()
	objschedtags := jointMan.Query().SubQuery()
	q := objs.Query()
	q = q.Join(objschedtags, sqlchemy.AND(sqlchemy.Equals(objschedtags.Field(jointMan.GetMasterIdKey(jointMan)), objs.Field("id")),
		sqlchemy.IsFalse(objschedtags.Field("deleted"))))
	q = q.Filter(sqlchemy.IsTrue(objs.Field("enabled")))
	q = q.Filter(sqlchemy.Equals(objschedtags.Field("schedtag_id"), self.Id))
	return q
}

func (self *SSchedtag) GetJointManager() ISchedtagJointManager {
	return SchedtagManager.jointsManager[self.ResourceType]
}

func (self *SSchedtag) GetObjectCount() (int, error) {
	return self.GetJointManager().Query().Equals("schedtag_id", self.Id).CountWithError()
}

func (self *SSchedtag) getSchedPoliciesCount() (int, error) {
	return SchedpolicyManager.Query().Equals("schedtag_id", self.Id).CountWithError()
}

func (self *SSchedtag) getDynamicSchedtagCount() (int, error) {
	return DynamicschedtagManager.Query().Equals("schedtag_id", self.Id).CountWithError()
}

func (self *SSchedtag) getMoreColumns(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	extra = self.SScopedResourceBase.GetMoreColumns(extra)
	cnt, _ := self.GetObjectCount()
	extra.Add(jsonutils.NewInt(int64(cnt)), fmt.Sprintf("%s_count", self.GetJointManager().GetMasterManager().Keyword()))
	cnt, _ = self.getDynamicSchedtagCount()
	extra.Add(jsonutils.NewInt(int64(cnt)), "dynamic_schedtag_count")
	cnt, _ = self.getSchedPoliciesCount()
	extra.Add(jsonutils.NewInt(int64(cnt)), "schedpolicy_count")
	return extra
}

func (self *SSchedtag) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreColumns(extra)
}

func (self *SSchedtag) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return self.getMoreColumns(extra), nil
}

/*func (self *SSchedtag) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)
}*/

func (self *SSchedtag) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := self.SStandaloneResourceBase.GetShortDesc(ctx)
	desc.Add(jsonutils.NewString(self.DefaultStrategy), "default")
	return desc
}

func GetResourceJointSchedtags(obj IModelWithSchedtag) ([]ISchedtagJointModel, error) {
	jointMan := obj.GetSchedtagJointManager()
	q := jointMan.Query().Equals(jointMan.GetMasterIdKey(jointMan), obj.GetId())
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
	q = q.Filter(sqlchemy.Equals(objschedtags.Field(jointMan.GetMasterIdKey(jointMan)), masterId))
	err := db.FetchModelObjects(SchedtagManager, q, &tags)
	if err != nil {
		log.Errorf("GetSchedtags error: %s", err)
		return nil
	}
	return tags
}

func AllowPerformSetResourceSchedtag(obj db.IModel, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, obj, "set-schedtag")
}

func PerformSetResourceSchedtag(obj IModelWithSchedtag, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	schedtags := jsonutils.GetArrayOfPrefix(data, "schedtag")
	setTagsId := []string{}
	for idx := 0; idx < len(schedtags); idx++ {
		schedtagIdent, _ := schedtags[idx].GetString()
		tag, err := SchedtagManager.FetchByIdOrName(userCred, schedtagIdent)
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
			if newTagObj, err := db.NewModelObject(jointMan); err != nil {
				return nil, httperrors.NewGeneralError(err)
			} else {
				objectKey := jointMan.GetMasterIdKey(jointMan)
				createData := jsonutils.NewDict()
				createData.Add(jsonutils.NewString(setTagId), "schedtag_id")
				createData.Add(jsonutils.NewString(obj.GetId()), objectKey)
				if err := createData.Unmarshal(newTagObj); err != nil {
					return nil, httperrors.NewGeneralError(fmt.Errorf("Create %s joint schedtag error: %v", jointMan.Keyword(), err))
				}
				if err := newTagObj.GetModelManager().TableSpec().Insert(newTagObj); err != nil {
					return nil, httperrors.NewGeneralError(err)
				}
			}
		}
	}
	obj.ClearSchedDescCache()
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

func (s *SSchedtag) AllowPerformSetScope(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return true
}

func (s *SSchedtag) PerformSetScope(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return SchedtagManager.PerformSetScope(ctx, s, userCred, data)
}
