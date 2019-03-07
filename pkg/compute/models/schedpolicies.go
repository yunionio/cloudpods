package models

import (
	"context"
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/conditionparser"
)

type SSchedpolicyManager struct {
	db.SStandaloneResourceBaseManager
}

var SchedpolicyManager *SSchedpolicyManager

func init() {
	SchedpolicyManager = &SSchedpolicyManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SSchedpolicy{},
			"schedpolicies_tbl",
			"schedpolicy",
			"schedpolicies",
		),
	}
}

// sched policy is called before calling scheduler, add additional preferences for schedtags
type SSchedpolicy struct {
	db.SStandaloneResourceBase

	Condition  string `width:"256" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
	SchedtagId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
	Strategy   string `width:"32" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`

	Enabled bool `nullable:"false" default:"true" create:"optional" list:"user" update:"user"`
}

func validateSchedpolicyInputData(data *jsonutils.JSONDict, create bool) error {
	err := validateDynamicSchedtagInputData(data, create)
	if err != nil {
		return err
	}

	strategyStr := jsonutils.GetAnyString(data, []string{"strategy"})
	if len(strategyStr) == 0 && create {
		return httperrors.NewMissingParameterError("strategy")
	}

	if len(strategyStr) > 0 && !utils.IsInStringArray(strategyStr, STRATEGY_LIST) {
		return httperrors.NewInputParameterError("invalid strategy %s", strategyStr)
	}

	return nil
}

func (self *SSchedpolicyManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SSchedpolicyManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SSchedpolicy) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SSchedpolicy) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SSchedpolicy) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (manager *SSchedpolicyManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	err := validateSchedpolicyInputData(data, true)
	if err != nil {
		return nil, err
	}

	return manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (self *SSchedpolicy) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	err := validateSchedpolicyInputData(data, false)
	if err != nil {
		return nil, err
	}

	return self.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (self *SSchedpolicy) getSchedtag() *SSchedtag {
	obj, err := SchedtagManager.FetchById(self.SchedtagId)
	if err != nil {
		log.Errorf("fail to fetch sched tag by id %s", err)
		return nil
	}
	return obj.(*SSchedtag)
}

func (self *SSchedpolicy) getMoreColumns(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	schedtag := self.getSchedtag()
	if schedtag != nil {
		extra.Add(jsonutils.NewString(schedtag.GetName()), "schedtag")
	}
	return extra
}

func (self *SSchedpolicy) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreColumns(extra)
}

func (self *SSchedpolicy) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return self.getMoreColumns(extra), nil
}

func (manager *SSchedpolicyManager) getAllEnabledPolicies() []SSchedpolicy {
	policies := make([]SSchedpolicy, 0)

	q := SchedpolicyManager.Query().IsTrue("enabled")
	err := db.FetchModelObjects(manager, q, &policies)
	if err != nil {
		log.Errorf("getAllEnabledPolicies fail %s", err)
		return nil
	}

	return policies
}

func (self *SSchedpolicy) AllowPerformEvaluate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "evaluate")
}

func (self *SSchedpolicy) PerformEvaluate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	serverStr := jsonutils.GetAnyString(data, []string{"server", "server_id", "guest", "guest_id"})
	serverObj, err := GuestManager.FetchByIdOrName(userCred, serverStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError("server %s not found", serverStr)
		} else {
			return nil, httperrors.NewGeneralError(err)
		}
	}

	server := serverObj.(*SGuest)
	desc := server.getSchedDesc()

	params := jsonutils.NewDict()
	params.Add(desc, "server")

	meet, err := conditionparser.Eval(self.Condition, params)
	if err != nil {
		return nil, err
	}
	result := jsonutils.NewDict()
	result.Add(desc, "server")
	if meet {
		result.Add(jsonutils.JSONTrue, "result")
	} else {
		result.Add(jsonutils.JSONFalse, "result")
	}
	return result, nil
}

func ApplySchedPolicies(params *jsonutils.JSONDict) *jsonutils.JSONDict {
	policies := SchedpolicyManager.getAllEnabledPolicies()
	if policies == nil {
		log.Errorf("getAllEnabledPolicies fail")
		return params
	}

	schedtags := make(map[string]string)

	if params.Contains("aggregate_strategy") {
		err := params.Unmarshal(&schedtags, "aggregate_strategy")
		if err != nil {
			log.Errorf("unmarshall aggregate_strategy fail %s", err)
			return params
		}
		log.Infof("original sched tag %#v", schedtags)
	}

	input := jsonutils.NewDict()
	input.Add(params, "server")

	for i := 0; i < len(policies); i += 1 {
		meet, err := conditionparser.Eval(policies[i].Condition, input)
		if err == nil && meet {
			st := policies[i].getSchedtag()
			if st != nil {
				schedtags[st.Name] = policies[i].Strategy
			}
		}
	}

	newSchedtags := jsonutils.Marshal(schedtags)
	log.Infof("updated sched tag %s", newSchedtags)

	ret := jsonutils.NewDict()
	ret.Add(newSchedtags, "aggregate_strategy")

	return ret
}
