// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SAlertDashBoardManager struct {
	db.SEnabledResourceBaseManager
	db.SStatusStandaloneResourceBaseManager
	db.SScopedResourceBaseManager
}

type SAlertDashBoard struct {
	//db.SVirtualResourceBase
	db.SEnabledResourceBase
	db.SStatusStandaloneResourceBase
	db.SScopedResourceBase

	Refresh string `nullable:"false" list:"user" create:"required" update:"user"`
}

var AlertDashBoardManager *SAlertDashBoardManager

func init() {
	AlertDashBoardManager = &SAlertDashBoardManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SAlertDashBoard{},
			"alertdashboard_tbl",
			"alertdashboard",
			"alertdashboards",
		),
	}

	AlertDashBoardManager.SetVirtualObject(AlertDashBoardManager)
}

func (manager *SAlertDashBoardManager) NamespaceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeSystem
}

func (manager *SAlertDashBoardManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}
	q, err = manager.SScopedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SScopedResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}

func (man *SAlertDashBoardManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input monitor.AlertDashBoardListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SScopedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.ScopedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SScopedResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (man *SAlertDashBoardManager) ValidateCreateData(
	ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject,
	data monitor.AlertDashBoardCreateInput) (monitor.AlertDashBoardCreateInput, error) {
	if len(data.Refresh) == 0 {
		data.Refresh = "1m"
	}
	if _, err := time.ParseDuration(data.Refresh); err != nil {
		return data, httperrors.NewInputParameterError("Invalid refresh format: %s", data.Refresh)
	}

	generateName, err := db.GenerateName(ctx, man, ownerId, data.Name)
	if err != nil {
		return data, err
	}
	data.Name = generateName
	return data, nil
}

func (dash *SAlertDashBoard) CustomizeCreate(
	ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) error {
	//return dash.SScopedResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
	return nil
}

func (dash *SAlertDashBoard) PostCreate(ctx context.Context,
	userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject, data jsonutils.JSONObject) {
	_, err := dash.PerformSetScope(ctx, userCred, query, data)
	if err != nil {
		log.Errorln(errors.Wrap(err, "dash PerformSetScope"))
	}
}

func (dash *SAlertDashBoard) PerformSetScope(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	domainId := jsonutils.GetAnyString(data, []string{"domain_id", "domain", "project_domain_id", "project_domain"})
	projectId := jsonutils.GetAnyString(data, []string{"project_id", "project"})
	if len(domainId) == 0 && len(projectId) == 0 {
		scope, _ := data.GetString("scope")
		if len(scope) != 0 {
			switch rbacscope.TRbacScope(scope) {
			case rbacscope.ScopeSystem:

			case rbacscope.ScopeDomain:
				domainId = userCred.GetProjectDomainId()
				data.(*jsonutils.JSONDict).Set("domain_id", jsonutils.NewString(domainId))
			case rbacscope.ScopeProject:
				projectId = userCred.GetProjectId()
				data.(*jsonutils.JSONDict).Set("project_id", jsonutils.NewString(projectId))
			}
		}
	}
	return db.PerformSetScope(ctx, dash, userCred, data)
}

func (man *SAlertDashBoardManager) ListItemFilter(
	ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query monitor.AlertDashBoardListInput,
) (*sqlchemy.SQuery, error) {
	q, err := AlertManager.ListItemFilter(ctx, q, userCred, query.AlertListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (man *SAlertDashBoardManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.AlertDashBoardDetails {
	rows := make([]monitor.AlertDashBoardDetails, len(objs))
	alertRows := AlertManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i].AlertDetails = alertRows[i]
		rows[i], _ = objs[i].(*SAlertDashBoard).GetMoreDetails(rows[i])
	}
	return rows
}

func (dash *SAlertDashBoard) GetMoreDetails(out monitor.AlertDashBoardDetails) (monitor.AlertDashBoardDetails, error) {
	panels, err := dash.getAttachPanels()
	if err != nil {
		return out, errors.Wrapf(err, "dashboard:%s GetMoreDetails error", dash.Name)
	}
	out.AlertPanelDetails = make([]monitor.AlertPanelDetail, len(panels))
	for i, panel := range panels {
		//out.AlertPanelDetail[i].PanelDetails = monitor.PanelDetails{}
		out.AlertPanelDetails[i].PanelDetails, err = panel.GetMoreDetails(out.AlertPanelDetails[i].PanelDetails)
		if err != nil {
			return out, errors.Wrapf(err, "dashboard:%s get panel:%s details error", dash.Name, panel.Name)
		}
		out.AlertPanelDetails[i].Setting = panel.Settings
		out.AlertPanelDetails[i].PanelId = panel.Id
		out.AlertPanelDetails[i].PanelName = panel.Name
	}
	return out, nil
}

func (dash *SAlertDashBoard) getJointPanels() ([]SAlertDashboardPanel, error) {
	joints := make([]SAlertDashboardPanel, 0)
	q := AlertDashBoardPanelManager.Query().Equals(AlertDashBoardPanelManager.GetMasterFieldName(), dash.Id)
	err := db.FetchModelObjects(AlertDashBoardPanelManager, q, &joints)
	if err != nil {
		return joints, errors.Wrapf(err, "get dash:%s joint panel err", dash.Name)
	}
	return joints, err
}

func (dash *SAlertDashBoard) getAttachPanels() ([]SAlertPanel, error) {
	panels := make([]SAlertPanel, 0)
	q := AlertPanelManager.Query()
	sq := AlertDashBoardPanelManager.Query().Equals("dashboard_id", dash.Id).SubQuery()
	q = q.Join(sq, sqlchemy.Equals(q.Field("id"), sq.Field("panel_id"))).Asc(sq.Field("index")).Desc("created_at")
	err := db.FetchModelObjects(AlertPanelManager, q, &panels)
	if err != nil {
		return panels, errors.Wrapf(err, "dashboard:%s get attach panels error", dash.Name)
	}
	return panels, nil
}

func (dash *SAlertDashBoard) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data *jsonutils.JSONDict,
) (*jsonutils.JSONDict, error) {
	if refresh, err := data.GetString("refresh"); err == nil && len(refresh) != 0 {
		if _, err := time.ParseDuration(refresh); err != nil {
			return data, httperrors.NewInputParameterError("Invalid refresh format: %s", refresh)
		}

	}
	return data, nil
}

func (dash *SAlertDashBoard) CustomizeDelete(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	panels, err := dash.getAttachPanels()
	if err != nil {
		return errors.Wrapf(err, "dash:%s,when exec customizedelete to get attach panels error", dash.Name)
	}
	for _, panel := range panels {
		if err := panel.CustomizeDelete(ctx, userCred, query, data); err != nil {
			return errors.Wrap(err, "dashboard exec panel CustomizeDelete error")
		}
		if err := panel.Delete(ctx, userCred); err != nil {
			return errors.Wrap(err, "dashboard exec panel Delete error")
		}
	}
	return nil
}

func (manager *SAlertDashBoardManager) getDashboardByid(id string) (*SAlertDashBoard, error) {
	iModel, err := db.FetchById(manager, id)
	if err != nil {
		return nil, err
	}
	return iModel.(*SAlertDashBoard), nil
}

func (dash *SAlertDashBoard) PerformClonePanel(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, input monitor.AlertClonePanelInput) (jsonutils.JSONObject, error) {

	panel, err := AlertPanelManager.getPanelByid(input.PanelId)
	if err != nil {
		return nil, errors.Wrapf(err, "getPanelByid:%s err", input.PanelId)
	}
	panelCloned, err := panel.ClonePanel(ctx, dash.GetId(), input)
	if err != nil {
		return nil, errors.Wrap(err, "ClonePanel err")
	}
	panelDetails := monitor.PanelDetails{}
	panelDetails, err = panelCloned.GetMoreDetails(panelDetails)
	if err != nil {
		return nil, errors.Wrapf(err, "panelCloned:%s GetMoreDetails err", panelCloned.Id)
	}
	output := jsonutils.Marshal(panelCloned)
	output.(*jsonutils.JSONDict).Update(jsonutils.Marshal(&panelDetails))

	return output, nil
}

func (dash *SAlertDashBoard) PerformSetPanelOrder(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input monitor.AlertPanelSetOrderInput,
) (jsonutils.JSONObject, error) {
	panels, err := dash.getJointPanels()
	if err != nil {
		return nil, errors.Wrapf(err, "getJointPanels")
	}
	order := map[string]int{}
	for _, v := range input.Order {
		order[v.PanelId] = v.Index
	}
	for i := range panels {
		panel := panels[i]
		index, ok := order[panel.PanelId]
		if ok {
			_, err := db.Update(&panel, func() error {
				panel.Index = index
				return nil
			})
			if err != nil {
				return nil, errors.Wrapf(err, "update index")
			}
		}
	}
	return nil, nil
}

func (dash *SAlertDashBoard) PerformCloneDashboard(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, input monitor.AlertCloneDashboardInput) (jsonutils.JSONObject, error) {
	iModel, err := db.NewModelObject(AlertDashBoardManager)
	if err != nil {
		return nil, errors.Wrap(err, "AlertDashBoardManager NewModelObject err")
	}
	dashJson := jsonutils.Marshal(dash)
	dashJson.Unmarshal(iModel)
	iModel.(*SAlertDashBoard).Id = ""
	name, err := db.GenerateName2(ctx, AlertDashBoardManager, nil, input.CloneName, iModel, 1)
	if err != nil {
		return nil, errors.Wrap(err, "clonePanel GenerateName err")
	}
	iModel.(*SAlertDashBoard).Name = name
	err = AlertDashBoardManager.TableSpec().Insert(ctx, iModel)
	if err != nil {
		return nil, errors.Wrapf(err, "insert clone dashboard err by id:%s", dash.GetId())
	}
	alertPanels, err := dash.getAttachPanels()
	if err != nil {
		return nil, errors.Wrapf(err, "dashboard:%s getAttachPanels err", dash.Id)
	}
	for _, panel := range alertPanels {
		newDashId := iModel.(*SAlertDashBoard).GetId()
		_, err := panel.ClonePanel(ctx, newDashId, monitor.AlertClonePanelInput{})
		if err != nil {
			return nil, errors.Wrapf(err, "ClonePanel %s for dashboard %s", panel.GetId(), newDashId)
		}
	}
	boardDetails, err := iModel.(*SAlertDashBoard).GetMoreDetails(monitor.AlertDashBoardDetails{})
	if err != nil {
		return nil, errors.Wrap(err, "GetMoreDetails of dashboard")
	}
	output := jsonutils.Marshal(iModel)
	output.(*jsonutils.JSONDict).Update((jsonutils.Marshal(&boardDetails)))
	return output, nil
}
