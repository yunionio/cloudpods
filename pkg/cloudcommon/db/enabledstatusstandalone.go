package db

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SEnabledStatusStandaloneResourceBase struct {
	SStatusStandaloneResourceBase

	Enabled bool `nullable:"false" default:"false" list:"user" create:"optional"` // = Column(Boolean, nullable=False, default=False)
}

type SEnabledStatusStandaloneResourceBaseManager struct {
	SStatusStandaloneResourceBaseManager
}

func NewEnabledStatusStandaloneResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SEnabledStatusStandaloneResourceBaseManager {
	return SEnabledStatusStandaloneResourceBaseManager{SStatusStandaloneResourceBaseManager: NewStatusStandaloneResourceBaseManager(dt, tableName, keyword, keywordPlural)}
}

func (self *SEnabledStatusStandaloneResourceBase) AllowPerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return IsAdminAllowPerform(userCred, self, "enable")
}

func (self *SEnabledStatusStandaloneResourceBase) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !self.Enabled {
		_, err := Update(self, func() error {
			self.Enabled = true
			return nil
		})
		if err != nil {
			log.Errorf("PerformEnable save update fail %s", err)
			return nil, err
		}
		OpsLog.LogEvent(self, ACT_ENABLE, "", userCred)
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_ENABLE, nil, userCred, true)
	}
	return nil, nil
}

func (self *SEnabledStatusStandaloneResourceBase) AllowPerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return IsAdminAllowPerform(userCred, self, "disable")
}

func (self *SEnabledStatusStandaloneResourceBase) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Enabled {
		_, err := Update(self, func() error {
			self.Enabled = false
			return nil
		})
		if err != nil {
			log.Errorf("PerformDisable save update fail %s", err)
			return nil, err
		}
		OpsLog.LogEvent(self, ACT_DISABLE, "", userCred)
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_DISABLE, nil, userCred, true)
	}
	return nil, nil
}
