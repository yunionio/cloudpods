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

type DiskUnused struct {
	monitor.DiskUnused
}

func NewDiskUnusedDriver() models.ISuggestSysRuleDriver {
	return &DiskUnused{
		DiskUnused: monitor.DiskUnused{},
	}
}

func (rule *DiskUnused) GetType() string {
	return monitor.DISK_UN_USED
}

func (rule *DiskUnused) GetResourceType() string {
	return string(monitor.DISK_MONITOR_RES_TYPE)
}

func (rule *DiskUnused) DoSuggestSysRule(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	doSuggestSysRule(ctx, userCred, isStart, rule)
}

func (rule *DiskUnused) Run(instance *monitor.SSuggestSysAlertSetting) {
	oldAlert, err := getLastAlerts(rule)
	if err != nil {
		log.Errorln(err)
		return
	}
	newAlerts, err := rule.getLatestAlerts(instance)
	if err != nil {
		log.Errorln(errors.Wrap(err, "DiskUnused getLatestAlerts error"))
		return
	}
	DealAlertData(rule.GetType(), oldAlert, newAlerts.Value())
}

func (rule *DiskUnused) getLatestAlerts(instance *monitor.SSuggestSysAlertSetting) (*jsonutils.JSONArray, error) {
	session := auth.GetAdminSession(context.Background(), "", "")
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString("0"), "limit")
	query.Add(jsonutils.NewBool(true), "unused")
	query.Add(jsonutils.NewString("system"), "scope")
	disks, err := modules.Disks.List(session, query)
	if err != nil {
		return nil, err
	}
	DiskUnusedArr := jsonutils.NewArray()
	for _, disk := range disks.Data {
		suggestSysAlert, err := getSuggestSysAlertFromJson(disk, rule)
		if err != nil {
			return DiskUnusedArr, errors.Wrap(err, "getEIPUnused's alertData Unmarshal error")
		}

		input := &monitor.SSuggestSysAlertSetting{
			DiskUnused: &monitor.DiskUnused{},
		}
		suggestSysAlert.MonitorConfig = jsonutils.Marshal(input)
		if instance != nil {
			suggestSysAlert.MonitorConfig = jsonutils.Marshal(instance)
		}

		problem := jsonutils.NewDict()
		problem.Add(jsonutils.NewString(rule.GetType()), "disk")
		suggestSysAlert.Problem = problem
		DiskUnusedArr.Add(jsonutils.Marshal(suggestSysAlert))
	}
	return DiskUnusedArr, nil
}

func (rule *DiskUnused) ValidateSetting(input *monitor.SSuggestSysAlertSetting) error {
	obj := new(monitor.DiskUnused)
	input.DiskUnused = obj
	return nil
}

func (rule *DiskUnused) StartResolveTask(ctx context.Context, userCred mcclient.TokenCredential,
	suggestSysAlert *models.SSuggestSysAlert, params *jsonutils.JSONDict) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ResolveUnusedTask", suggestSysAlert, userCred, params, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (rule *DiskUnused) Resolve(data *models.SSuggestSysAlert) error {
	session := auth.GetAdminSession(context.Background(), "", "")
	_, err := modules.Disks.Delete(session, data.ResId, jsonutils.NewDict())
	if err != nil {
		log.Errorln("delete unused eip error", err)
		return err
	}
	return nil
}
