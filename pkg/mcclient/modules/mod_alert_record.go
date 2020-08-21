package modules

import "yunion.io/x/onecloud/pkg/mcclient/modulebase"

type SAlertRecordManager struct {
	*modulebase.ResourceManager
}

var (
	AlertRecordManager *SAlertRecordManager
)

func init() {
	AlertRecordManager = NewAlertRecordManager()
	register(AlertRecordManager)
}

func NewAlertRecordManager() *SAlertRecordManager {
	man := NewMonitorV2Manager("alertrecord", "alertrecords",
		[]string{"id", "alert_id", "level", "state", "eval_data"},
		[]string{})
	return &SAlertRecordManager{
		ResourceManager: &man,
	}
}
