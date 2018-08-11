package modules

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type QuotaManager struct {
	ResourceManager
}

func (this *QuotaManager) getURL(params jsonutils.JSONObject) string {
	url := fmt.Sprintf("/%s", this.URLPath())
	if params != nil {
		tenant, _ := params.GetString("tenant")
		if len(tenant) > 0 {
			url = fmt.Sprintf("%s/%s", url, tenant)
			user, _ := params.GetString("user")
			if len(user) > 0 {
				url = fmt.Sprintf("%s/%s", url, user)
			}
		}
	}
	return url
}

func (this *QuotaManager) GetQuota(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return this._get(s, this.getURL(params), this.KeywordPlural)
}

func (this *QuotaManager) DoQuotaSet(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	url := this.getURL(params)
	quotas, ok := params.(*jsonutils.JSONDict)
	if !ok {
		return nil, fmt.Errorf("Invalid input")
	}
	data := quotas.Copy("tenant", "user")
	body := jsonutils.NewDict()
	body.Add(data, "quotas")
	return this._post(s, url, body, this.KeywordPlural)
}

func (this *QuotaManager) DoQuotaCheck(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	url := this.getURL(params)
	url = fmt.Sprintf("%s/check_quota", url)
	quotas, ok := params.(*jsonutils.JSONDict)
	if !ok {
		return nil, fmt.Errorf("Invalid input")
	}
	data := quotas.Copy("tenant", "user")
	body := jsonutils.NewDict()
	body.Add(data, "quotas")
	return this._post(s, url, body, this.KeywordPlural)
}

var (
	Quotas QuotaManager
)

func init() {
	Quotas = QuotaManager{NewComputeManager("quota", "quotas",
		[]string{},
		[]string{})}
	registerCompute(&Quotas)
}
