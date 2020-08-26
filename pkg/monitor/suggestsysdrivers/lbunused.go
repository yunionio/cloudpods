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

package suggestsysdrivers

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/monitor/models"
)

type LBUnused struct {
	*baseDriver
}

func NewLBUnusedDriver() models.ISuggestSysRuleDriver {
	return &LBUnused{
		baseDriver: newBaseDriver(
			monitor.LB_UNUSED,
			monitor.LB_MONITOR_RES_TYPE,
			monitor.DELETE_DRIVER_ACTION,
			monitor.LB_MONITOR_SUGGEST,
		),
	}
}

func (drv *LBUnused) ValidateSetting(input *monitor.SSuggestSysAlertSetting) error {
	obj := new(monitor.LBUnused)
	input.LBUnused = obj
	return nil
}

func (drv *LBUnused) DoSuggestSysRule(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	doSuggestSysRule(ctx, userCred, isStart, drv)
}

func (drv *LBUnused) Run(rule *models.SSuggestSysRule, setting *monitor.SSuggestSysAlertSetting) {
	Run(drv, rule, setting)
}

func (drv *LBUnused) GetLatestAlerts(rule *models.SSuggestSysRule, instance *monitor.SSuggestSysAlertSetting) ([]jsonutils.JSONObject, error) {
	session := auth.GetAdminSession(context.Background(), "", "")
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString("0"), "limit")
	query.Add(jsonutils.NewString("system"), "scope")
	lbs, err := modules.Loadbalancers.List(session, query)
	if err != nil {
		return nil, err
	}
	lbArr := make([]jsonutils.JSONObject, 0)
	for _, lb := range lbs.Data {
		lbId, _ := lb.GetString("id")

		contains, problems, err := containsLbBackEndGroups(lbId)
		if err != nil {
			log.Errorln(err)
			continue
		}
		if *contains {
			continue
		}
		contains, err = getLbListeners(lbId)
		if err != nil {
			log.Errorln(err)
			continue
		}
		if *contains {
			continue
		}
		problems = append(problems, monitor.SuggestAlertProblem{
			Type:        "listener",
			Description: monitor.LB_UNUSED_NLISTENER,
		})
		suggestSysAlert, err := getSuggestSysAlertFromJson(lb, drv)
		if err != nil {
			return lbArr, errors.Wrap(err, "getLatestAlerts's alertData Unmarshal error")
		}

		input := &monitor.SSuggestSysAlertSetting{
			LBUnused: &monitor.LBUnused{},
		}
		suggestSysAlert.MonitorConfig = jsonutils.Marshal(input)
		if instance != nil {
			suggestSysAlert.MonitorConfig = jsonutils.Marshal(instance)
		}
		suggestSysAlert.Problem = jsonutils.Marshal(problems)

		lbArr = append(lbArr, jsonutils.Marshal(suggestSysAlert))
	}
	return lbArr, nil
}

func getLbListeners(lbId string) (*bool, error) {
	contains := false
	session := auth.GetAdminSession(context.Background(), "", "")
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString("0"), "limit")
	query.Add(jsonutils.NewString("system"), "scope")
	query.Add(jsonutils.NewString(lbId), "loadbalancer")
	listeners, err := modules.LoadbalancerListeners.List(session, query)
	if err != nil {
		return nil, err
	}
	if listeners != nil && len(listeners.Data) > 0 {
		for _, listener := range listeners.Data {
			status, _ := listener.GetString("status")
			if status == "enabled" {
				contains = true
				break
			}
		}
	}
	return &contains, nil
}

func containsLbBackEndGroups(lbId string) (*bool, []monitor.SuggestAlertProblem, error) {
	contains := false

	problems := make([]monitor.SuggestAlertProblem, 0)
	session := auth.GetAdminSession(context.Background(), "", "")
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString("0"), "limit")
	query.Add(jsonutils.NewString("system"), "scope")
	query.Add(jsonutils.NewString(lbId), "loadbalancer")
	groups, err := modules.LoadbalancerBackendGroups.List(session, query)
	if err != nil {
		return nil, problems, err
	}
	for _, group := range groups.Data {
		groupId, _ := group.GetString("id")
		backEnds, err := containsLbBackEnd(groupId)
		if err != nil {
			return nil, problems, err
		}
		if *backEnds {
			return backEnds, problems, nil
		}
	}
	if len(groups.Data) == 0 {
		problems = append(problems, monitor.SuggestAlertProblem{
			Type:        "backendgroup",
			Description: monitor.LB_UNUSED_NBCGROUP,
		})
	}
	problems = append(problems, monitor.SuggestAlertProblem{
		Type:        "backend",
		Description: monitor.LB_UNUSED_NBC,
	})
	return &contains, problems, nil
}

func containsLbBackEnd(groupId string) (*bool, error) {
	contains := false

	session := auth.GetAdminSession(context.Background(), "", "")
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString("0"), "limit")
	query.Add(jsonutils.NewString("system"), "scope")
	query.Add(jsonutils.NewString(groupId), "backend_group")
	backEnds, err := modules.LoadbalancerBackends.List(session, query)
	if err != nil {
		return nil, err
	}
	if len(backEnds.Data) > 0 {
		contains = true
		return &contains, nil
	}
	return &contains, nil
}

func (rule *LBUnused) StartResolveTask(ctx context.Context, userCred mcclient.TokenCredential,
	suggestSysAlert *models.SSuggestSysAlert,
	params *jsonutils.JSONDict) error {
	suggestSysAlert.SetStatus(userCred, monitor.SUGGEST_ALERT_START_DELETE, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ResolveUnusedTask", suggestSysAlert, userCred, params, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (rule *LBUnused) Resolve(data *models.SSuggestSysAlert) error {
	session := auth.GetAdminSession(context.Background(), "", "")
	_, err := modules.Loadbalancers.Delete(session, data.ResId, jsonutils.NewDict())
	if err != nil {
		log.Errorln("delete unused lb error", err)
		return err
	}
	return nil
}
