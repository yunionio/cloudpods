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

package modules

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type AlarmEventsManager struct {
	modulebase.ResourceManager
}

func (this *AlarmEventsManager) DoBatchUpdate(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	path := "/alarm_events"

	return modulebase.Put(this.ResourceManager, s, path, params, "alarm_events")
}

var (
	AlarmEvents AlarmEventsManager
)

func init() {
	AlarmEvents = AlarmEventsManager{NewServiceTreeManager("alarm_event", "alarm_events",
		[]string{"ID", "metric_name", "host_name", "host_ip", "alarm_condition", "template", "first_alarm_time", "last_alarm_time", "alarm_status", "alarm_times", "ack_time", "ack_status", "ack_wait_time", "upgrade_time", "upgrade_status", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark"},
		[]string{})}

	register(&AlarmEvents)
}
