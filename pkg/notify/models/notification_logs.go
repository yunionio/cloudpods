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
	"time"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

type SNotificationLogManager struct {
	db.SLogBaseManager
}

var NotificationLogManager *SNotificationLogManager

func InitNotificationLog() {
	NotificationLogManager = &SNotificationLogManager{
		SLogBaseManager: db.NewLogBaseManager(SNotificationLog{}, "notification_logs_tbl", "notification", "notifications", "created_at", consts.OpsLogWithClickhouse),
	}
	NotificationLogManager.SetVirtualObject(NotificationLogManager)
}

// 站内信
type SNotificationLog struct {
	db.SLogBase

	ContactType string `width:"128" nullable:"true" create:"optional" list:"user" get:"user"`
	// swagger:ignore
	Topic    string `width:"128" nullable:"true" create:"required" list:"user" get:"user"`
	Priority string `width:"16" nullable:"true" create:"optional" list:"user" get:"user"`
	// swagger:ignore
	Message string `create:"required"`
	// swagger:ignore
	TopicType  string    `json:"topic_type" width:"20" nullable:"true" update:"user" list:"user"`
	ReceivedAt time.Time `nullable:"true" list:"user" get:"user"`
	EventId    string    `width:"128" nullable:"true"`

	SendTimes int
}
