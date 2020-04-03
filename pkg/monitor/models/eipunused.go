package models

import (
	"context"
	"database/sql"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type EIPUnused struct {
	Status string
}

func (_ *EIPUnused) GetType() string {
	return EIPUsed
}

func NewEIPUsedDriver() IDriver {
	return &EIPUnused{
		Status: "",
	}
}

func (dri *EIPUnused) ValidateSetting(input monitor.SuggestSysRuleCreateInput) error {
	eipUnsed := new(EIPUnused)
	return input.Setting.Unmarshal(eipUnsed)
}

func (rule *EIPUnused) Run(instance *SSuggestSysAlartSetting) {
	oldAlert := make([]DSuggestSysAlert, 0)
	q := SuggestSysAlertManager.Query()
	q.Equals("type", EIPUsed)
	err := db.FetchModelObjects(SuggestSysAlertManager, q, &oldAlert)
	if err != nil && err != sql.ErrNoRows {
		log.Errorln(errors.Wrap(err, "db.FetchModelObjects"))
		return
	}
	newAlert := rule.getEIPUnused(instance)

	DealAlertDatas(oldAlert, newAlert.Value())
}

func (rule *EIPUnused) getEIPUnused(instance *SSuggestSysAlartSetting) *jsonutils.JSONArray {
	//处理逻辑
	session := auth.GetAdminSession(context.Background(), "", "")
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString("scope"), "system")
	rtn, err := modules.Elasticips.List(session, query)
	if err != nil {
		log.Errorln(err)
		return jsonutils.NewArray()
	}
	EIPUnsedArr := jsonutils.NewArray()
	for _, row := range rtn.Data {
		if row.ContainsIgnoreCases("associate_type") || row.ContainsIgnoreCases("associate_id") {
			continue
		}
		alertData := jsonutils.DeepCopy(row).(*jsonutils.JSONDict)
		id, _ := alertData.GetString("id")
		alertData.Add(jsonutils.NewString(id), "res_id")
		alertData.Remove("id")

		alertData.Add(jsonutils.Marshal(rule), "monitorConfig")
		if instance != nil {
			alertData.Add(jsonutils.Marshal(instance.EIPUnsed), "monitorConfig")
		}

		problem := jsonutils.NewDict()
		problem.Add(jsonutils.NewString(rule.GetType()), "eip")
		alertData.Add(problem, "problem")

		alertData.Add(jsonutils.NewString("释放未使用的EIP"), "suggest")
		alertData.Add(jsonutils.NewString(rule.GetType()), "type")

		alertData.Add(jsonutils.NewString(DRIVER_ACTION), "action")

		alertData.Add(row, "res_meta")
		EIPUnsedArr.Add(alertData)
	}
	return EIPUnsedArr
}

func DealAlertDatas(oldAlerts []DSuggestSysAlert, newAlerts []jsonutils.JSONObject) {
	oldMap := make(map[string]DSuggestSysAlert, 0)
	for _, alert := range oldAlerts {
		oldMap[alert.ResId] = alert
	}

	for _, newAlert := range newAlerts {
		res_id, _ := newAlert.GetString("res_id")
		if oldAlert, ok := oldMap[res_id]; ok {
			//更新的alert
			_, err := db.Update(&oldAlert, func() error {
				err := newAlert.Unmarshal(&oldAlert)
				if err != nil {
					errMsg := fmt.Sprintf("unmarshal fail: %s", err)
					log.Errorf(errMsg)
				}
				return nil
			})
			if err != nil {
				log.Errorln("更新alert失败", err)
			}
			delete(oldMap, res_id)
		} else {
			//新增的alert
			adminCredential := auth.AdminCredential()
			_, err := db.DoCreate(SuggestSysAlertManager, context.Background(), adminCredential, nil, newAlert,
				adminCredential)
			if err != nil {
				log.Errorln(err)
			}
		}
	}

	for _, oldAlert := range oldMap {
		err := oldAlert.Delete(context.Background(), auth.AdminCredential())
		if err != nil {
			log.Errorln("删除旧alert数据失败", err)
		}
	}
}

//cronjob
func (rule *EIPUnused) DoSuggestSysRule(ctx context.Context, userCred mcclient.TokenCredential,
	isStart bool) {
	SuggestSysRuleManager.fetchSuggestSysAlartSettings(EIPUsed)
	driver := SuggestSysRuleManager.drivers[EIPUsed]
	driver.Run(SuggestSysRuleManager.suggestSysAlartSettings[EIPUsed])
}
