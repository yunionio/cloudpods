package models

import (
	"context"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	AlertDashBoardPanelManager *SAlertDashboardPanelManager
)

type SAlertDashboardPanelManager struct {
	db.SJointResourceBaseManager
}

func init() {
	db.InitManager(func() {
		AlertDashBoardPanelManager = &SAlertDashboardPanelManager{
			SJointResourceBaseManager: db.NewJointResourceBaseManager(
				SAlertDashboardPanel{},
				"alertdashboardpanel_tbl",
				"alertdashboardpanel",
				"alertdashboardpanels",
				AlertDashBoardManager,
				AlertPanelManager,
			),
		}
		AlertDashBoardPanelManager.SetVirtualObject(AlertDashBoardPanelManager)
		AlertDashBoardPanelManager.TableSpec().AddIndex(true, AlertDashBoardPanelManager.GetMasterFieldName(), AlertDashBoardPanelManager.GetSlaveFieldName())
	})
}

type SAlertDashboardPanel struct {
	db.SVirtualJointResourceBase

	DashboardId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	PanelId     string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
}

func (man *SAlertDashboardPanelManager) GetMasterFieldName() string {
	return "dashboard_Id"
}

func (man *SAlertDashboardPanelManager) GetSlaveFieldName() string {
	return "panel_id"
}

func (joint *SAlertDashboardPanel) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, joint)
}
