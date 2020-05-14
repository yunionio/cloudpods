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
	monitor.LBUnused
}

func NewLBUnusedDriver() models.ISuggestSysRuleDriver {
	return &LBUnused{
		LBUnused: monitor.LBUnused{},
	}
}

func (_ *LBUnused) GetType() string {
	return monitor.LB_UN_USED
}

func (rule *LBUnused) GetResourceType() string {
	return string(monitor.LB_MONITOR_RES_TYPE)
}

func (rule *LBUnused) ValidateSetting(input *monitor.SSuggestSysAlertSetting) error {
	obj := new(monitor.LBUnused)
	input.LBUnused = obj
	return nil
}

func (rule *LBUnused) DoSuggestSysRule(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	doSuggestSysRule(ctx, userCred, isStart, rule)
}

func (rule *LBUnused) Run(instance *monitor.SSuggestSysAlertSetting) {
	oldAlert, err := getLastAlerts(rule)
	if err != nil {
		log.Errorln(err)
		return
	}
	newAlert, err := rule.getLatestAlerts(instance)
	if err != nil {
		log.Errorln(errors.Wrap(err, "getEIPUnused error"))
		return
	}
	DealAlertData(rule.GetType(), oldAlert, newAlert.Value())
}

func (rule *LBUnused) getLatestAlerts(instance *monitor.SSuggestSysAlertSetting) (*jsonutils.JSONArray, error) {
	session := auth.GetAdminSession(context.Background(), "", "")
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString("0"), "limit")
	query.Add(jsonutils.NewString("system"), "scope")
	lbs, err := modules.Loadbalancers.List(session, query)
	if err != nil {
		return nil, err
	}
	lbArr := jsonutils.NewArray()
	for _, lb := range lbs.Data {
		lbId, _ := lb.GetString("id")
		contains, err := containsLbBackEndGroups(lbId)
		if err != nil {
			return lbArr, err
		}
		if *contains {
			continue
		}

		suggestSysAlert, err := getSuggestSysAlertFromJson(lb, rule)
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

		problem := jsonutils.NewDict()
		problem.Add(jsonutils.NewString(rule.GetType()), "lb")
		suggestSysAlert.Problem = problem

		lbArr.Add(jsonutils.Marshal(suggestSysAlert))
	}
	return lbArr, nil
}

func containsLbBackEndGroups(lbId string) (*bool, error) {
	contains := false
	session := auth.GetAdminSession(context.Background(), "", "")
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString("0"), "limit")
	query.Add(jsonutils.NewString("system"), "scope")
	query.Add(jsonutils.NewString(lbId), "loadbalancer")
	groups, err := modules.LoadbalancerBackendGroups.List(session, query)
	if err != nil {
		return nil, err
	}
	for _, group := range groups.Data {
		groupId, _ := group.GetString("id")
		backEnds, err := containsLbBackEnd(groupId)
		if err != nil {
			return nil, err
		}
		if *backEnds {
			return backEnds, nil
		}
	}
	return &contains, nil
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
