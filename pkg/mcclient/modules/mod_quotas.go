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
			url = fmt.Sprintf("%s/projects/%s", url, tenant)
		} else {
			domain, _ := params.GetString("domain")
			if len(domain) > 0 {
				url = fmt.Sprintf("%s/domains/%s", url, domain)
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

func (this *QuotaManager) GetQuotaList(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	var reqUrl string
	domainId := jsonutils.GetAnyString(params, []string{"domain", "project_domain", "domain_id", "project_domain_id"})
	if len(domainId) > 0 {
		reqUrl = "/quotas/projects?domain_id=" + domainId
	} else {
		reqUrl = "/quotas/domains"
	}
	computeQuotaList, err := this._list(s, reqUrl, this.KeywordPlural)
	if err != nil {
		return nil, err
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewArray(computeQuotaList.Data...), "data")
	return ret, nil
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
