package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"database/sql"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/conditionparser"
)

type SDynamicschedtagManager struct {
	db.SStandaloneResourceBaseManager
}

var DynamicschedtagManager *SDynamicschedtagManager

func init() {
	DynamicschedtagManager = &SDynamicschedtagManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SDynamicschedtag{},
			"dynamicschedtags_tbl",
			"dynamicschedtag",
			"dynamicschedtags",
		),
	}
}

// dynamic schedtag is called before scan host candidates, dynamically adding additional schedtag to hosts
// condition examples:
//  host.sys_load > 1.5 || host.mem_used_percent > 0.7 => "high_load"
//
type SDynamicschedtag struct {
	db.SStandaloneResourceBase

	Condition  string `width:"256" charset:"ascii" nullable:"false" list:"user" create:"required" update:"admin"`
	SchedtagId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" update:"admin"`

	Enabled bool `nullable:"false" default:"true" create:"optional" list:"user" update:"user"`
}

func (self *SDynamicschedtagManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SDynamicschedtagManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SDynamicschedtag) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SDynamicschedtag) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SDynamicschedtag) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func validateDynamicSchedtagInputData(data *jsonutils.JSONDict, create bool) error {
	condStr := jsonutils.GetAnyString(data, []string{"condition"})
	if len(condStr) == 0 && create {
		return httperrors.NewInputParameterError("empty condition")
	}
	if len(condStr) > 0 && !conditionparser.IsValid(condStr) {
		return httperrors.NewInputParameterError("invalid condition")
	}

	schedStr := jsonutils.GetAnyString(data, []string{"schedtag", "schedtag_id"})
	if len(schedStr) == 0 && create {
		return httperrors.NewInputParameterError("missing schedtag")
	}
	if len(schedStr) > 0 {
		schedObj, err := SchedtagManager.FetchByIdOrName(nil, schedStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return httperrors.NewResourceNotFoundError("schedtag %s not found", schedStr)
			} else {
				log.Errorf("fetch schedtag %s fail %s", schedStr, err)
				return httperrors.NewGeneralError(err)
			}
		}
		data.Set("schedtag_id", jsonutils.NewString(schedObj.GetId()))
	}

	return nil
}

func (manager *SDynamicschedtagManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	err := validateDynamicSchedtagInputData(data, true)
	if err != nil {
		return nil, err
	}
	return manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (self *SDynamicschedtag) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	err := validateDynamicSchedtagInputData(data, false)
	if err != nil {
		return nil, err
	}

	return self.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (self *SDynamicschedtag) getSchedtag() *SSchedtag {
	obj, err := SchedtagManager.FetchById(self.SchedtagId)
	if err != nil {
		log.Errorf("fail to fetch sched tag by id %s", err)
		return nil
	}
	return obj.(*SSchedtag)
}

func (self *SDynamicschedtag) getMoreColumns(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	schedtag := self.getSchedtag()
	if schedtag != nil {
		extra.Add(jsonutils.NewString(schedtag.GetName()), "schedtag")
	}
	return extra
}

func (self *SDynamicschedtag) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreColumns(extra)
}

func (self *SDynamicschedtag) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	return self.getMoreColumns(extra)
}

func (manager *SDynamicschedtagManager) getAllEnabledDynamicSchedtags() []SDynamicschedtag {
	rules := make([]SDynamicschedtag, 0)

	q := DynamicschedtagManager.Query().IsTrue("enabled")
	err := db.FetchModelObjects(manager, q, &rules)
	if err != nil {
		log.Errorf("getAllEnabledDynamicSchedtags fail %s", err)
		return nil
	}

	return rules
}

func (self *SDynamicschedtag) AllowPerformEvaluate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "evaluate")
}

func (self *SDynamicschedtag) PerformEvaluate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
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
	srvDesc := server.getSchedDesc()

	hostStr := jsonutils.GetAnyString(data, []string{"host", "host_id"})
	hostObj, err := HostManager.FetchByIdOrName(userCred, hostStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError("host %s not found", serverStr)
		} else {
			return nil, httperrors.NewGeneralError(err)
		}
	}

	host := hostObj.(*SHost)
	// TODO: to fill host scheduling information
	hostDesc := jsonutils.Marshal(host)

	params := jsonutils.NewDict()
	params.Add(srvDesc, "server")
	params.Add(hostDesc, "host")

	meet, err := conditionparser.Eval(self.Condition, params)
	if err != nil {
		return nil, err
	}
	result := jsonutils.NewDict()
	result.Add(srvDesc, "server")
	result.Add(hostDesc, "host")

	if meet {
		result.Add(jsonutils.JSONTrue, "result")
	} else {
		result.Add(jsonutils.JSONFalse, "result")
	}
	return result, nil
}
