package monitor

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

type AlertJointResourceBaseDetails struct {
	SAlertJointsBase
	Alert string `json:"alert"`
}

type AlertnotificationDetails struct {
	AlertJointResourceBaseDetails
	Notification string `json:"notification"`
}

type AlertJointCreateInput struct {
	apis.Meta

	AlertId string `json:"alert_id"`
}

type AlertnotificationCreateInput struct {
	AlertJointCreateInput

	NotificationId string               `json:"notification_id"`
	UsedBy         string               `json:"used_by"`
	Params         jsonutils.JSONObject `json:"params"`
}
