package monitor

import (
	"time"

	"yunion.io/x/onecloud/pkg/apis"
)

type MonitorResourceJointListInput struct {
	MonitorResourceId string  `json:"monitor_resource_id"`
	AlertId           string  `json:"alert_id"`
	JointId           []int64 `json:"joint_id"`
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
