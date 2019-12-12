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
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type QuotaManager struct {
	modulebase.ResourceManager
}

func (this *QuotaManager) getURL(params jsonutils.JSONObject) string {
	url := fmt.Sprintf("/%s", this.URLPath())
	query := jsonutils.NewDict()
	if params != nil {
		tenant, _ := params.GetString("tenant")
		if len(tenant) > 0 {
			url = fmt.Sprintf("%s/projects/%s", url, tenant)
		} else {
			domain := jsonutils.GetAnyString(params, []string{"domain", "project_domain"})
			if len(domain) > 0 {
				url = fmt.Sprintf("%s/domains/%s", url, domain)
			} else {
				scope, _ := params.GetString("scope")
				if len(scope) > 0 {
					query.Add(jsonutils.NewString(scope), "scope")
				}
			}
		}
		refresh := jsonutils.QueryBoolean(params, "refresh", false)
		if refresh {
			query.Add(jsonutils.JSONTrue, "refresh")
		}
	}
	if query.Size() > 0 {
		url += "?" + query.QueryString()
	}
	return url
}

func (this *QuotaManager) GetQuota(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	quotas, err := modulebase.Get(this.ResourceManager, s, this.getURL(params), this.KeywordPlural)
	if err != nil {
		return nil, err
	}
	ret := jsonutils.NewDict()
	ret.Add(quotas, "data")
	return ret, nil
}

func (this *QuotaManager) DoCleanPendingUsage(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	url := this.getURL(params)
	results, err := modulebase.Delete(this.ResourceManager, s, url, nil, "")
	if err != nil {
		return nil, err
	}
	return results, nil
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
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewArray(computeQuotaList.Data...), "data")
	return ret, nil
}

func (this *QuotaManager) DoQuotaSet(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	url := this.getURL(params)
	body := jsonutils.NewDict()
	body.Add(params, this.KeywordPlural)
	results, err := modulebase.Post(this.ResourceManager, s, url, body, this.KeywordPlural)
	if err != nil {
		return nil, err
	}
	ret := jsonutils.NewDict()
	ret.Add(results, "data")
	return ret, nil
}

var (
	Quotas        QuotaManager
	ProjectQuotas QuotaManager
	RegionQuotas  QuotaManager
	ZoneQuotas    QuotaManager
	ImageQuotas   QuotaManager

	quotaColumns = []string{}
	/*quotaColumns = []string{
		"domain", "domain_id",
		"tenant", "tenant_id",
		"provider",
		"brand",
		"cloud_env",
		"account", "account_id",
		"manager", "manager_id",
		"region", "region_id",
		"zone", "zone_id",
		"hypervisor",
	}*/
)

func init() {
	Quotas = QuotaManager{NewComputeManager("quota", "quotas",
		quotaColumns,
		[]string{})}
	registerCompute(&Quotas)

	ProjectQuotas = QuotaManager{NewComputeManager("project_quota", "project_quotas",
		quotaColumns,
		[]string{})}
	registerCompute(&ProjectQuotas)

	RegionQuotas = QuotaManager{NewComputeManager("region_quota", "region_quotas",
		quotaColumns,
		[]string{})}
	registerCompute(&RegionQuotas)

	ZoneQuotas = QuotaManager{NewComputeManager("zone_quota", "zone_quotas",
		quotaColumns,
		[]string{})}
	registerCompute(&ZoneQuotas)

	ImageQuotas = QuotaManager{NewImageManager("quota", "quotas",
		quotaColumns,
		[]string{})}
	// registerV2(&ImageQuotas)
}
