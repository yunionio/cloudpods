package models

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SActionlogManager struct {
	db.SOpsLogManager
}

type SActionlog struct {
	db.SOpsLog

	StartTime time.Time `nullable:"false" list:"user" create:"optional"` // = Column(DateTime, nullable=False)
	Success   bool      `default:"true" list:"user" create:"required"`   // = Column(Boolean, default=True)
	// Action    string    `width:"32" charset:"utf8" nullable:"false" list:"user"` //= Column(VARCHAR(32, charset='utf8'), nullable=False)
}

var ActonLog *SActionlogManager

func init() {
	ActonLog = &SActionlogManager{db.SOpsLogManager{db.NewModelBaseManager(SActionlog{}, "action_tbl", "action", "actions")}}
}

func (action *SActionlog) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	now := time.Now().UTC()
	action.OpsTime = now
	if action.StartTime.IsZero() {
		action.StartTime = now
	}
	return nil
}
