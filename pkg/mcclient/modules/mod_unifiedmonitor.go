package modules

import "yunion.io/x/onecloud/pkg/mcclient/modulebase"

var (
	UnifiedMonitorManager *SUnifiedMonitorManager
)

type SUnifiedMonitorManager struct {
	*modulebase.ResourceManager
}

func init() {
	UnifiedMonitorManager = NewUnifiedMonitorManager()
	for _, m := range []modulebase.IBaseManager{
		UnifiedMonitorManager,
	} {
		Register(m)
	}
}

func NewUnifiedMonitorManager() *SUnifiedMonitorManager {
	man := NewMonitorV2Manager("unifiedmonitor", "unifiedmonitors",
		[]string{},
		[]string{})
	return &SUnifiedMonitorManager{
		ResourceManager: &man,
	}
}
