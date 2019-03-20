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

type SSchedtagManager struct {
	db.SStandaloneResourceBaseManager

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
	manager.BindJointManagers(
		HostschedtagManager,
		StorageschedtagManager,
	)
	return nil
}

func (manager *SSchedtagManager) BindJointManagers(ms ...ISchedtagJointManager) {
	for _, m := range ms {
		manager.jointsManager[m.GetMasterManager().KeywordPlural()] = m
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

func (manager *SSchedtagManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
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
	return manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
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
	if self.GetObjectCount() > 0 {
		return httperrors.NewNotEmptyError("Tag is associated with %s", self.ResourceType)
	}
	if self.getDynamicSchedtagCount() > 0 {
		return httperrors.NewNotEmptyError("tag has dynamic rules")
	}
	if self.getSchedPoliciesCount() > 0 {
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

func (self *SSchedtag) GetObjectCount() int {
	return self.GetJointManager().Query().Equals("schedtag_id", self.Id).Count()
}

func (self *SSchedtag) getSchedPoliciesCount() int {
	return SchedpolicyManager.Query().Equals("schedtag_id", self.Id).Count()
}

func (self *SSchedtag) getDynamicSchedtagCount() int {
	return DynamicschedtagManager.Query().Equals("schedtag_id", self.Id).Count()
}

func (self *SSchedtag) getMoreColumns(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	extra.Add(jsonutils.NewInt(int64(self.GetObjectCount())), fmt.Sprintf("%s_count", self.GetJointManager().GetMasterManager().Keyword()))
	extra.Add(jsonutils.NewInt(int64(self.getDynamicSchedtagCount())), "dynamic_schedtag_count")
	extra.Add(jsonutils.NewInt(int64(self.getSchedPoliciesCount())), "schedpolicy_count")
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
