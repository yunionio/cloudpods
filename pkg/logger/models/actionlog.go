package models

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

var WhiteListMap = make(map[string]bool)

func InitActionWhiteList() {
	for _, value := range logclient.WhiteList {
		WhiteListMap[value] = true
	}
}

func IsInActionWhiteList(key string) bool {
	return WhiteListMap[key]
}

type SActionlogManager struct {
	db.SOpsLogManager
}

type SActionlog struct {
	db.SOpsLog

	StartTime time.Time `nullable:"false" list:"user" create:"optional"`                          // = Column(DateTime, nullable=False)
	Success   bool      `list:"user" create:"required"`                                           // = Column(Boolean, default=True)
	Service   string    `width:"32" charset:"utf8" nullable:"true" list:"user" create:"optional"` //= Column(VARCHAR(32, charset='utf8'), nullable=False)
}

var ActonLog *SActionlogManager
var logQueue = make(chan *SActionlog, 50)

func init() {
	InitActionWhiteList()
	ActonLog = &SActionlogManager{db.SOpsLogManager{db.NewModelBaseManager(SActionlog{}, "action_tbl", "action", "actions")}}
}

func (action *SActionlog) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	now := time.Now().UTC()
	action.OpsTime = now
	if action.StartTime.IsZero() {
		action.StartTime = now
	}
	return action.SOpsLog.CustomizeCreate(ctx, userCred, ownerProjId, query, data)
}

// Websockets 不再拉取 ActionLog 的消息，因此注释掉如下代码
// 可以保留，以便有需求时，再次打开
// func (manager *SActionlogManager) OnCreateComplete(ctx context.Context, items []db.IModel, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
//	actionLog := items[0].(*SActionlog)
//	if IsInActionWhiteList(actionLog.Action) {
//		select {
//		case logQueue <- actionLog:
//			return
//		default:
//			log.Warningf("Log queue full, insert failed, log ignored: %s", actionLog.Action)
//		}
//	}
// }

func StartNotifyToWebsocketWorker() {
	go func() {
		for {
			actionLog := <-logQueue
			params := jsonutils.Marshal(actionLog)
			s := auth.GetAdminSession(context.Background(), "", "")
			_, err := modules.Websockets.PerformClassAction(s, "action-log", params)
			if err != nil {
				log.Errorf("Send action log error %s", err)
			}
		}
	}()
}
