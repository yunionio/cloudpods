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

package compute

import (
	"context"
	"net/http"
	"net/url"

	"golang.org/x/net/http/httpproxy"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/utils"

	proxyapi "yunion.io/x/onecloud/pkg/apis/cloudcommon/proxy"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type SCloudprovider struct {
	modulebase.ResourceManager
}

var (
	Cloudproviders SCloudprovider
)

func init() {
	Cloudproviders = SCloudprovider{
		modules.NewComputeManager("cloudprovider", "cloudproviders",
			[]string{"ID", "Name", "Enabled", "Status", "Access_url", "Account",
				"Sync_Status", "Last_sync", "Last_sync_end_at",
				"health_status",
				"Provider", "guest_count", "host_count", "vpc_count",
				"storage_count", "storage_cache_count", "eip_count",
				"tenant_id", "tenant"},
			[]string{}),
	}

	modules.RegisterCompute(&Cloudproviders)
}

type SCloudDelegate struct {
	Id             string
	Name           string
	Enabled        bool
	Status         string
	SyncStatus     string
	CloudaccountId string

	AccessUrl string
	Account   string
	Secret    string

	Provider string
	Brand    string

	ReadOnly bool

	Options struct {
		cloudprovider.SHCSOEndpoints
		Account       string
		Password      string
		DefaultRegion string
	}
	ProxySetting proxyapi.SProxySetting
}

func (account *SCloudDelegate) getPassword() (string, error) {
	return utils.DescryptAESBase64(account.Id, account.Secret)
}

func (account *SCloudDelegate) getAccessUrl() string {
	return account.AccessUrl
}

func (account *SCloudDelegate) getOptions(ctx context.Context, s *mcclient.ClientSession) (string, jsonutils.JSONObject) {
	regionId, ret := "", jsonutils.NewDict()
	resp, _ := Cloudaccounts.GetById(s, account.CloudaccountId, jsonutils.Marshal(map[string]string{"scope": "system"}))
	if !gotypes.IsNil(resp) {
		options, _ := resp.Get("options")
		ret.Update(options)
		regionId, _ = resp.GetString("region_id")
		if len(regionId) == 0 {
			regionId, _ = ret.GetString("default_region")
		}
	}
	return regionId, ret
}

func (self *SCloudprovider) GetProvider(ctx context.Context, s *mcclient.ClientSession, id string) (cloudprovider.ICloudProvider, error) {
	result, err := self.Get(s, id, jsonutils.Marshal(map[string]string{"scope": "system"}))
	if err != nil {
		return nil, errors.Wrap(err, "Cloudprovider.Get")
	}
	account := &SCloudDelegate{}
	err = result.Unmarshal(account)
	if err != nil {
		return nil, errors.Wrap(err, "result.Unmarshal")
	}
	if !account.Enabled {
		log.Warningf("Cloud provider %s is disabled", account.Name)
	}

	accessUrl := account.getAccessUrl()
	passwd, err := account.getPassword()
	if err != nil {
		return nil, err
	}
	var proxyFunc httputils.TransportProxyFunc
	{
		cfg := &httpproxy.Config{
			HTTPProxy:  account.ProxySetting.HTTPProxy,
			HTTPSProxy: account.ProxySetting.HTTPSProxy,
			NoProxy:    account.ProxySetting.NoProxy,
		}
		cfgProxyFunc := cfg.ProxyFunc()
		proxyFunc = func(req *http.Request) (*url.URL, error) {
			return cfgProxyFunc(req.URL)
		}
	}
	regionId, options := account.getOptions(ctx, s)
	return cloudprovider.GetProvider(cloudprovider.ProviderConfig{
		Id:        account.Id,
		Name:      account.Name,
		Vendor:    account.Provider,
		URL:       accessUrl,
		Account:   account.Account,
		Secret:    passwd,
		ProxyFunc: proxyFunc,

		ReadOnly: account.ReadOnly,

		RegionId: regionId,
		Options:  options.(*jsonutils.JSONDict),

		AccountId: account.Id,
	})
}
