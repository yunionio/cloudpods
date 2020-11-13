package suggestsysdrivers

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	compute_api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/monitor/dbinit"
	"yunion.io/x/onecloud/pkg/monitor/models"
)

type SecGroupRuleInServer struct {
	*baseDriver
}

func NewSecGroupRuleInServerDriver() models.ISuggestSysRuleDriver {
	return &SecGroupRuleInServer{
		baseDriver: newBaseDriver(
			monitor.SECGROUPRULEINSERVER_ALLIN,
			monitor.SECGROUPRULEINSERVER_MONITOR_RES_TYPE,
			monitor.SECGROUPRULEINSERVER_DRIVER_ACTION,
			monitor.SECGROUPRULEINSERVER_MONITOR_SUGGEST,
			*dbinit.SecGroupRuleInCreateInput,
		),
	}
}

func (drv *SecGroupRuleInServer) ValidateSetting(input *monitor.SSuggestSysAlertSetting) error {
	return nil
}

func (drv *SecGroupRuleInServer) DoSuggestSysRule(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	doSuggestSysRule(ctx, userCred, isStart, drv)
}

func (drv *SecGroupRuleInServer) Run(rule *models.SSuggestSysRule, setting *monitor.SSuggestSysAlertSetting) {
	Run(drv, rule, setting)
}

func (drv *SecGroupRuleInServer) GetLatestAlerts(rule *models.SSuggestSysRule,
	setting *monitor.SSuggestSysAlertSetting) ([]jsonutils.JSONObject, error) {
	secGroupIdArr, err := drv.getSecGroupIdsInThisRule()
	if err != nil {
		return nil, err
	}
	servers, err := drv.getServersBySecGroupIds(secGroupIdArr)
	if err != nil {
		return nil, err
	}
	secGroupRuleInServerArr := make([]jsonutils.JSONObject, 0)
	for _, server := range servers {
		suggestSysAlert, err := getSuggestSysAlertFromJson(server, drv)
		if err != nil {
			return nil, err
		}
		suggestSysAlert.Name = fmt.Sprintf("%s-%s", suggestSysAlert.Name, string(drv.GetType()))
		suggestSysAlert.Amount = 0
		secGroupRuleInServerArr = append(secGroupRuleInServerArr, jsonutils.Marshal(suggestSysAlert))
	}
	return secGroupRuleInServerArr, nil

}

func (drv *SecGroupRuleInServer) getSecGroupIdsInThisRule() ([]string, error) {
	param := jsonutils.NewDict()
	param.Add(jsonutils.NewString("in"), "direction")
	secGroups, err := ListAllResources(&modules.SecGroups, param)
	if err != nil {
		return nil, err
	}
	secGroupIdArr := make([]string, 0)
	for _, secGroup := range secGroups {
		secGroupDetail := new(compute_api.SecgroupDetails)
		secGroup.Unmarshal(secGroupDetail)
		if secGroupDetail.GuestCnt == 0 {
			continue
		}
		for _, inRule := range secGroupDetail.InRules {
			if inRule.CIDR == monitor.SECGROUPRULEINSERVER_CIDR && len(inRule.Ports) == 0 &&
				strings.ToLower(inRule.Protocol) != monitor.SECGROUPRULEINSERVER_FILTER_PROTOCOL {
				secGroupIdArr = append(secGroupIdArr, secGroupDetail.Id)
				break
			}
		}
	}
	return secGroupIdArr, nil
}

func (drv *SecGroupRuleInServer) getServersBySecGroupIds(secGroupIdArr []string) ([]jsonutils.JSONObject, error) {
	param := jsonutils.NewDict()
	param.Add(jsonutils.NewString("hypervisor.notin(baremetal,container)"), "filter.0")
	servers := make([]jsonutils.JSONObject, 0)
	for _, secGroupId := range secGroupIdArr {
		param.Set("secgroup_id", jsonutils.NewString(secGroupId))
		serversPart, err := ListAllResources(&modules.Servers, param)
		if err != nil {
			return nil, errors.Wrap(err, "SecGroupRuleInServer getServers error")
		}
		servers = append(servers, serversPart...)
	}
	return servers, nil
}

func (drv *SecGroupRuleInServer) StartResolveTask(ctx context.Context, userCred mcclient.TokenCredential,
	suggestSysAlert *models.SSuggestSysAlert, params *jsonutils.JSONDict) error {
	log.Println("SecGroupRuleInServer StartResolveTask do nothing")
	return nil
}

func (s SecGroupRuleInServer) Resolve(data *models.SSuggestSysAlert) error {
	log.Println("InfluxdbBaseDriver Resolve do nothing")
	return nil
}
