package modules

import "yunion.io/x/onecloud/pkg/mcclient/modulebase"

var (
	MonitorResourceManager *SMonitorResourceManager
)

type SMonitorResourceManager struct {
	*modulebase.ResourceManager
}

func init() {
	MonitorResourceManager = NewMonitorResourceManager()

	register(MonitorResourceManager)
}

func NewMonitorResourceManager() *SMonitorResourceManager {
	man := NewMonitorV2Manager("monitorresource", "monitorresources",
		[]string{"id", "name", "res_type", "res_id", "alert_state"},
		[]string{})
	return &SMonitorResourceManager{
		ResourceManager: &man,
	}
}
