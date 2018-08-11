package modules

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type AlarmEventsManager struct {
	ResourceManager
}

func (this *AlarmEventsManager) DoBatchUpdate(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	path := "/alarm_events"

	return this._put(s, path, params, "alarm_events")
}

var (
	AlarmEvents AlarmEventsManager
)

func init() {
	AlarmEvents = AlarmEventsManager{NewMonitorManager("alarm_event", "alarm_events",
		[]string{"ID", "metric_name", "host_name", "host_ip", "alarm_condition", "template", "first_alarm_time", "last_alarm_time", "alarm_status", "alarm_times", "ack_time", "ack_status", "ack_wait_time", "upgrade_time", "upgrade_status", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark"},
		[]string{})}

	register(&AlarmEvents)
}
