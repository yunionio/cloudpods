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

package oldmodels

import (
	"time"
)

type SNotificationManager struct {
	SStatusStandaloneResourceBaseManager
}

var NotificationManager *SNotificationManager

func init() {
	NotificationManager = &SNotificationManager{
		SStatusStandaloneResourceBaseManager: NewStatusStandaloneResourceBaseManager(
			SNotification{},
			"notify_t_notification",
			"oldnotification",
			"oldnotifications",
		),
	}
	NotificationManager.SetVirtualObject(NotificationManager)
}

type SNotification struct {
	SStatusStandaloneResourceBase

	UID         string    `width:"128" nullable:"false" create:"required"`
	ContactType string    `width:"16" nullable:"false" create:"required" list:"user" index:"true"`
	Topic       string    `width:"128" nullable:"true" create:"optional" list:"user"`
	Priority    string    `width:"16" nullable:"true" create:"optional" list:"user"`
	Msg         string    `create:"required"`
	ReceivedAt  time.Time `nullable:"true" list:"user" create:"optional"`
	SendAt      time.Time `nullable:"false"`
	SendBy      string    `width:"128" nullable:"false"`
	// ClusterID identify message with same topic, msg, priority
	ClusterID string `width:"128" charset:"ascii" primary:"true" create:"optional" list:"user" get:"user"`
}
