package models

import (
	"context"
	"database/sql"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
)

type SchedStrategyType string

const (
	STRATEGY_REQUIRE = "require"
	STRATEGY_EXCLUDE = "exclude"
	STRATEGY_PREFER  = "prefer"
	STRATEGY_AVOID   = "avoid"

	// # container used aggregate
	CONTAINER_AGGREGATE = "container"
)

var STRATEGY_LIST = []string{STRATEGY_REQUIRE, STRATEGY_EXCLUDE, STRATEGY_PREFER, STRATEGY_AVOID}

type SSchedtagManager struct {
	db.SStandaloneResourceBaseManager
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
	}
}

type SSchedtag struct {
	db.SStandaloneResourceBase

	DefaultStrategy string `width:"16" charset:"ascii" nullable:"true" default:"" list:"user" update:"admin" create:"admin_optional"` // Column(VARCHAR(16, charset='ascii'), nullable=True, default='')
}

func (manager *SSchedtagManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SSchedtag) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SSchedtagManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsAdminAllow(consts.GetServiceType(), self.KeywordPlural(), policy.PolicyActionCreate)
}

func (self *SSchedtag) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return userCred.IsAdminAllow(consts.GetServiceType(), self.KeywordPlural(), policy.PolicyActionUpdate)
}

func (self *SSchedtag) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsAdminAllow(consts.GetServiceType(), self.KeywordPlural(), policy.PolicyActionDelete)
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
	return manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
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
	if self.GetHostCount() > 0 {
		return httperrors.NewNotEmptyError("Tag is associated with hosts")
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

func (self *SSchedtag) GetHosts() []SHost {
	q := self.GetHostQuery()
	hosts := make([]SHost, 0)
	err := db.FetchModelObjects(HostManager, q, &hosts)
	if err != nil {
		log.Errorf("GetHosts query fail %s", err)
		return nil
	}
	return hosts
}

func (self *SSchedtag) GetHostQuery() *sqlchemy.SQuery {
	hosts := HostManager.Query().SubQuery()
	hostschedtags := HostschedtagManager.Query().SubQuery()
	q := hosts.Query()
	q = q.Join(hostschedtags, sqlchemy.AND(sqlchemy.Equals(hostschedtags.Field("host_id"), hosts.Field("id")),
		sqlchemy.IsFalse(hostschedtags.Field("deleted"))))
	q = q.Filter(sqlchemy.IsTrue(hosts.Field("enabled")))
	q = q.Filter(sqlchemy.Equals(hostschedtags.Field("schedtag_id"), self.Id))
	return q
}

func (self *SSchedtag) GetHostCount() int {
	return HostschedtagManager.Query().Equals("schedtag_id", self.Id).Count()
}

func (self *SSchedtag) getSchedPoliciesCount() int {
	return SchedpolicyManager.Query().Equals("schedtag_id", self.Id).Count()
}

func (self *SSchedtag) getDynamicSchedtagCount() int {
	return DynamicschedtagManager.Query().Equals("schedtag_id", self.Id).Count()
}

func (self *SSchedtag) getMoreColumns(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	extra.Add(jsonutils.NewInt(int64(self.GetHostCount())), "host_count")
	extra.Add(jsonutils.NewInt(int64(self.getDynamicSchedtagCount())), "dynamic_schedtag_count")
	extra.Add(jsonutils.NewInt(int64(self.getSchedPoliciesCount())), "schedpolicy_count")
	return extra
}

func (self *SSchedtag) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreColumns(extra)
}

func (self *SSchedtag) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	return self.getMoreColumns(extra)
}

/*func (self *SSchedtag) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)
}*/

func (self *SSchedtag) GetShortDesc() *jsonutils.JSONDict {
	desc := jsonutils.NewDict()
	desc.Add(jsonutils.NewString(self.Id), "id")
	desc.Add(jsonutils.NewString(self.Name), "name")
	desc.Add(jsonutils.NewString(self.DefaultStrategy), "default")
	return desc
}
