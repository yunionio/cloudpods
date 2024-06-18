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
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type SCloudaccount struct {
	modulebase.ResourceManager
}

var (
	Cloudaccounts SCloudaccount
)

func init() {
	Cloudaccounts = SCloudaccount{
		modules.NewComputeManager("cloudaccount", "cloudaccounts",
			[]string{"ID", "Name", "Enabled", "Status", "Access_url",
				"balance", "currency", "error_count", "health_status",
				"Sync_Status", "Last_sync",
				"guest_count", "project_domain", "domain_id",
				"Provider", "Brand",
				"Enable_Auto_Sync", "Sync_Interval_Seconds",
				"Share_Mode", "is_public", "public_scope",
				"auto_create_project",
			},
			[]string{}),
	}

	modules.RegisterCompute(&Cloudaccounts)
}

func (self *SCloudaccount) GetProvider(ctx context.Context, s *mcclient.ClientSession, id string) (cloudprovider.ICloudProvider, error) {
	result, err := self.Get(s, id, jsonutils.Marshal(map[string]string{"scope": "system"}))
	if err != nil {
		return nil, errors.Wrap(err, "Cloudaccounts.Get")
	}
	account := &SCloudDelegate{}
	err = result.Unmarshal(account)
	if err != nil {
		return nil, errors.Wrap(err, "result.Unmarshal")
	}
	if !account.Enabled {
		log.Warningf("Cloud account %s is disabled", account.Name)
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
	})
}
