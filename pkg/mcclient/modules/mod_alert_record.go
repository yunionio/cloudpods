package modules

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type SAlertRecordManager struct {
	*modulebase.ResourceManager
}

var (
	AlertRecordManager       *SAlertRecordManager
	AlertRecordShieldManager *SAlertRecordManager
)

func init() {
	AlertRecordManager = NewAlertRecordManager()
	AlertRecordShieldManager = NewAlertRecordShieldManager()
	register(AlertRecordManager)
	register(AlertRecordShieldManager)
}

func NewAlertRecordManager() *SAlertRecordManager {
	man := NewMonitorV2Manager("alertrecord", "alertrecords",
		[]string{"id", "alert_name", "res_type", "level", "state", "res_num", "eval_data"},
		[]string{})
	return &SAlertRecordManager{
		ResourceManager: &man,
	}
}

func NewAlertRecordShieldManager() *SAlertRecordManager {
	man := NewMonitorV2Manager("alertrecordshield", "alertrecordshields",
		[]string{"id", "res_name", "alert_name", "res_type", "alert_id", "start_time", "end_time"},
		[]string{})
	return &SAlertRecordManager{
		ResourceManager: &man,
	}
}
