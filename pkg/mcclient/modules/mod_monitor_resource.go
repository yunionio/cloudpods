package modules

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

var (
	MonitorResourceManager      *SMonitorResourceManager
	MonitorResourceAlertManager *SMonitorResourceAlertManager
)

type SMonitorResourceManager struct {
	*modulebase.ResourceManager
}

type SMonitorResourceAlertManager struct {
	*modulebase.JointResourceManager
}

func init() {
	MonitorResourceManager = NewMonitorResourceManager()
	MonitorResourceAlertManager = NewAlertResourceAlertManager()

	register(MonitorResourceManager)
	register(MonitorResourceAlertManager)
}

func NewMonitorResourceManager() *SMonitorResourceManager {
	man := NewMonitorV2Manager("monitorresource", "monitorresources",
		[]string{"id", "name", "res_type", "res_id", "alert_state"},
		[]string{})
	return &SMonitorResourceManager{
		ResourceManager: &man,
	}
}

func NewAlertResourceAlertManager() *SMonitorResourceAlertManager {
	man := NewJointMonitorV2Manager("monitorresourcealert", "monitorresourcealerts",
		[]string{"monitor_resource_id", "alert_id", "res_name", "res_type", "alert_name", "alert_state", "send_state", "level",
			"trigger_time", "data"},
		[]string{},
		MonitorResourceManager, CommonAlertManager)
	return &SMonitorResourceAlertManager{&man}
}
