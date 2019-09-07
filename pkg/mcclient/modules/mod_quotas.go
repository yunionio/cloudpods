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
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"

	"yunion.io/x/onecloud/pkg/mcclient"
)

type QuotaManager struct {
	modulebase.ResourceManager
}

func (this *QuotaManager) getURL(params jsonutils.JSONObject) string {
	url := fmt.Sprintf("/%s", this.URLPath())
	if params != nil {
		tenant, _ := params.GetString("tenant")
		if len(tenant) > 0 {
			url = fmt.Sprintf("%s/projects/%s", url, tenant)
		} else {
			domain := jsonutils.GetAnyString(params, []string{"domain", "project_domain"})
			if len(domain) > 0 {
				url = fmt.Sprintf("%s/domains/%s", url, domain)
			}
		}
	}
	return url
}

func (this *QuotaManager) GetQuota(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	computeQuota, err := modulebase.Get(this.ResourceManager, s, this.getURL(params), this.KeywordPlural)
	if err != nil {
		return nil, err
	}
	imageQuota, err := modulebase.Get(ImageQuotas.ResourceManager, s, ImageQuotas.getURL(params), ImageQuotas.KeywordPlural)
	if err != nil {
		return nil, err
	}
	computeQuotaDict := computeQuota.(*jsonutils.JSONDict)
	computeQuotaDict.Update(imageQuota)
	return computeQuotaDict, nil
}

func getQuotaKey(quota jsonutils.JSONObject) string {
	domainId, _ := quota.GetString("domain_id")
	tenantId, _ := quota.GetString("tenant_id")
	platform, _ := quota.GetString("platform")
	return fmt.Sprintf("%s-%s-%s", domainId, tenantId, platform)
}

func quotaListToMap(list []jsonutils.JSONObject) map[string]jsonutils.JSONObject {
	ret := make(map[string]jsonutils.JSONObject)
	for i := range list {
		key := getQuotaKey(list[i])
		ret[key] = list[i]
	}
	return ret
}

func mergeQuotaList(list1 []jsonutils.JSONObject, list2 []jsonutils.JSONObject) []jsonutils.JSONObject {
	list2map := quotaListToMap(list2)
	for i := range list1 {
		key := getQuotaKey(list1[i])
		if quota, ok := list2map[key]; ok {
			list1[i].(*jsonutils.JSONDict).Update(quota)
		}
	}
	return list1
}

func (this *QuotaManager) GetQuotaList(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	var reqUrl string
	domainId := jsonutils.GetAnyString(params, []string{"domain", "project_domain", "domain_id", "project_domain_id"})
	if len(domainId) > 0 {
		reqUrl = "/quotas/projects?project_domain=" + domainId
	} else {
		reqUrl = "/quotas/domains"
	}
	computeQuotaList, err := modulebase.List(this.ResourceManager, s, reqUrl, this.KeywordPlural)
	if err != nil {
		return nil, err
	}
	imageQuotaList, err := modulebase.List(ImageQuotas.ResourceManager, s, reqUrl, this.KeywordPlural)
	if err != nil {
		return nil, err
	}
	data := mergeQuotaList(computeQuotaList.Data, imageQuotaList.Data)
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewArray(data...), "data")
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
		log.Debugf("set compute quota %s", body)
		_, err = modulebase.Post(this.ResourceManager, s, url, body, this.KeywordPlural)
		if err != nil {
			log.Errorf("set compute quota fail %s %s", data, err)
			return nil, err
		}
	}
	data = quotas.CopyIncludes("image", "action", "cascade")
	if data.Size() > 0 {
		body := jsonutils.NewDict()
		body.Add(data, ImageQuotas.KeywordPlural)
		log.Debugf("set image quota %s", body)
		_, err = modulebase.Post(ImageQuotas.ResourceManager, s, url, body, ImageQuotas.KeywordPlural)
		if err != nil {
			log.Errorf("set quota fail %s %s", data, err)
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
