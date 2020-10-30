package modules

import "yunion.io/x/onecloud/pkg/mcclient/modulebase"

type SAlertDashBoardManager struct {
	*modulebase.ResourceManager
}

var (
	AlertDashBoardManager *SAlertDashBoardManager
)

func init() {
	AlertDashBoardManager = NewAlertDashBoardManager()
	register(AlertDashBoardManager)
}
func NewAlertDashBoardManager() *SAlertDashBoardManager {
	man := NewMonitorV2Manager("alertdashboard", "alertdashboards",
		[]string{"id", "name", "refresh", "alert_panel_details"},
		[]string{})
	return &SAlertDashBoardManager{
		ResourceManager: &man,
	}
}
