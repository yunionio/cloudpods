package suggestsysdrivers

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

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
	i, count := 1, 0
	param := jsonutils.NewDict()
	param.Add(jsonutils.NewString("hypervisor.notin(baremetal,container)"), "filter.0")
	filterSecGroupIds := make([]string, 0)
	jump := false
	for {
		tmp := count + 50
		if tmp > len(secGroupIdArr) {
			tmp = len(secGroupIdArr)
			jump = true
		}
		filterSecGroupIds = secGroupIdArr[count:tmp]
		param.Add(jsonutils.NewString(fmt.Sprintf("`secgroup.in(%s)", strings.Join(filterSecGroupIds, ","))),
			fmt.Sprintf("filter.%d", i))
		if jump {
			break
		}
		i++
		count = tmp
	}
	return ListAllResources(&modules.Servers, param)
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
