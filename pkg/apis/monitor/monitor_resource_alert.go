package monitor

import (
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

type MonitorResourceJointListInput struct {
	apis.JointResourceBaseListInput
	MonitorResourceId string  `json:"monitor_resource_id"`
	AlertId           string  `json:"alert_id"`
	JointId           []int64 `json:"joint_id"`
	Alerting          bool    `json:"alertinng"`
	SendState         string  `json:"send_state"`
	ResType           string  `json:"res_type"`
}

type MonitorResourceJointCreateInput struct {
	apis.Meta
	MonitorResourceId string `json:"monitor_resource_id"`
	AlertId           string `json:"alert_id"`

	AlertRecordId string    `width:"36" charset:"ascii" list:"user"  update:"user"`
	AlertState    string    `width:"18" charset:"ascii" list:"user"  update:"user"`
	TriggerTime   time.Time `list:"user"  update:"user" json:"trigger_time"`
	Data          EvalMatch `json:"data"`
}

type MonitorResourceJointDetails struct {
	ResName   string               `json:"res_name"`
	ResType   string               `json:"res_type"`
	AlertName string               `json:"alert_name"`
	AlertRule jsonutils.JSONObject `json:"alert_rule"`
	Level     string               `json:"level"`
	SendState string               `json:"send_state"`
}
