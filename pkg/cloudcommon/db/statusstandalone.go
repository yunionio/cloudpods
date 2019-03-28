package db

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/mcclient"
)

type SStatusStandaloneResourceBase struct {
	SStandaloneResourceBase

	Status string `width:"36" charset:"ascii" nullable:"false" default:"init" list:"user" create:"optional"`
}

type SStatusStandaloneResourceBaseManager struct {
	SStandaloneResourceBaseManager
}

func NewStatusStandaloneResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SStatusStandaloneResourceBaseManager {
	return SStatusStandaloneResourceBaseManager{SStandaloneResourceBaseManager: NewStandaloneResourceBaseManager(dt, tableName, keyword, keywordPlural)}
}

func (model *SStatusStandaloneResourceBase) SetStatus(userCred mcclient.TokenCredential, status string, reason string) error {
	if model.Status == status {
		return nil
	}
	oldStatus := model.Status
	_, err := Update(model, func() error {
		model.Status = status
		return nil
	})
	if err != nil {
		return err
	}
	if userCred != nil {
		notes := fmt.Sprintf("%s=>%s", oldStatus, status)
		if len(reason) > 0 {
			notes = fmt.Sprintf("%s: %s", notes, reason)
		}
		OpsLog.LogEvent(model, ACT_UPDATE_STATUS, notes, userCred)
	}
	return nil
}

func (model *SStatusStandaloneResourceBase) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return IsAdminAllowPerform(userCred, model, "status")
}

func (model *SStatusStandaloneResourceBase) PerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	status, err := data.GetString("status")
	if err != nil {
		return nil, err
	}
	reason, _ := data.GetString("reason")
	err = model.SetStatus(userCred, status, reason)
	return nil, err
}

func (model *SStatusStandaloneResourceBase) IsInStatus(status ...string) bool {
	return utils.IsInStringArray(model.Status, status)
}
