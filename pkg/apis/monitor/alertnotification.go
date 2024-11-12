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

package monitor

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	AlertNotificationUsedByMeterAlert     = "meter_alert"
	AlertNotificationUsedByNodeAlert      = "node_alert"
	AlertNotificationUsedByCommonAlert    = "common_alert"
	AlertNotificationUsedByMigrationAlert = "migration_alert"
)

type AlertJointResourceBaseDetails struct {
	apis.JointResourceBaseDetails
	Alert string `json:"alert"`
}

type AlertnotificationDetails struct {
	AlertJointResourceBaseDetails
	Notification string `json:"notification"`
	Frequency    int64  `json:"frequency"`
}

type AlertnotificationCreateInput struct {
	AlertJointCreateInput

	NotificationId string               `json:"notification_id"`
	UsedBy         string               `json:"used_by"`
	Params         jsonutils.JSONObject `json:"params"`
}

type AlertNotificationListInput struct {
	AlertJointListInput
}
