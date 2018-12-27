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
	computeQuota, err := this._get(s, this.getURL(params), this.KeywordPlural)
	if err != nil {
		return nil, err
	}
	imageQuota, err := ImageQuotas._get(s, ImageQuotas.getURL(params), ImageQuotas.KeywordPlural)
	if err != nil {
		return nil, err
	}
	computeQuotaDict := computeQuota.(*jsonutils.JSONDict)
	computeQuotaDict.Update(imageQuota)
	return computeQuotaDict, nil
}

func (this *QuotaManager) doPost(s *mcclient.ClientSession, params jsonutils.JSONObject, url string) (jsonutils.JSONObject, error) {
	quotas, ok := params.(*jsonutils.JSONDict)
	if !ok {
		return nil, fmt.Errorf("Invalid input")
	}
	data := quotas.CopyExcludes("tenant", "user", "image")
	var err error
	if data.Size() > 0 {
		body := jsonutils.NewDict()
		body.Add(data, this.KeywordPlural)
		_, err = this._post(s, url, body, this.KeywordPlural)
		if err != nil {
			return nil, err
		}
	}
	data = quotas.CopyIncludes("image")
	if data.Size() > 0 {
		body := jsonutils.NewDict()
		body.Add(data, ImageQuotas.KeywordPlural)
		_, err = ImageQuotas._post(s, url, body, ImageQuotas.KeywordPlural)
		if err != nil {
			return nil, err
		}
	}
	return jsonutils.NewDict(), nil
}

func (this *QuotaManager) DoQuotaSet(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	url := this.getURL(params)
	return this.doPost(s, params, url)
}

func (this *QuotaManager) DoQuotaCheck(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	url := this.getURL(params)
	url = fmt.Sprintf("%s/check_quota", url)
	return this.doPost(s, params, url)
}

var (
	Quotas      QuotaManager
	ImageQuotas QuotaManager
)

func init() {
	Quotas = QuotaManager{NewComputeManager("quota", "quotas",
		[]string{},
		[]string{})}
	registerCompute(&Quotas)

	ImageQuotas = QuotaManager{NewImageManager("quota", "quotas",
		[]string{},
		[]string{})}
	// registerV2(&ImageQuotas)
}
