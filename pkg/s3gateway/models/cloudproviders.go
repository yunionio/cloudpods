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

package models

import (
	"context"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/s3gateway/session"
	"yunion.io/x/onecloud/pkg/util/hashcache"
	"yunion.io/x/pkg/utils"
)

type SCloudproviderManagerDelegate struct {
	providers *hashcache.Cache
}

var CloudproviderManager *SCloudproviderManagerDelegate

func init() {
	CloudproviderManager = &SCloudproviderManagerDelegate{
		providers: hashcache.NewCache(2048, time.Minute*15),
	}
}

/*
{
	"account":"oH7Qrw75AqrI4BXn",
	"can_delete":false,
	"can_update":true,
	"cloudaccount":"testaliyun",
	"cloudaccount_id":"f1a927b4-a433-486e-86ae-e9aa36e447b1",
	"created_at":"2019-04-16T13:55:11.000000Z",
	"domain":"Default",
	"domain_id":"default",
	"eip_count":9,
	"enabled":true,
	"guest_count":1,
	"health_status":"normal",
	"host_count":61,
	"id":"57c84d93-8f06-4a85-8963-4ce42eabb339",
	"is_emulated":false,
	"last_sync":"2019-07-23T15:29:34.000000Z",
	"last_sync_end_at":"2019-07-23T15:33:34.000000Z",
	"loadbalancer_count":13,
	"name":"testaliyun",
	"project_count":0,
	"provider":"Aliyun",
	"secret":"Y5YFmuwVI4frJ8kVgWL0z5Kan/sJ3JMyjyFRxAXwXvsUKd8aNohPp2T/Kr1BqA==",
	"snapshot_count":6,
	"status":"connected",
	"storage_cache_count":20,
	"storage_count":141,
	"sync_region_count":20,
	"sync_status":"idle",
	"sync_status2":"idle",
	"tenant":"system",
	"tenant_id":"5d65667d112e47249ae66dbd7bc07030",
	"update_version":5003,
	"updated_at":"2019-08-18T14:47:42.000000Z",
	"vpc_count":13,
}
*/

type SCloudproviderDelegate struct {
	SBaseModelDelegate

	Enabled    bool
	Status     string
	SyncStatus string

	AccessUrl string
	Account   string
	Secret    string

	Provider string
	Brand    string
}

func (manager *SCloudproviderManagerDelegate) GetById(ctx context.Context, userCred mcclient.TokenCredential, id string) (*SCloudproviderDelegate, error) {
	val := manager.providers.AtomicGet(id)
	if !gotypes.IsNil(val) {
		return val.(*SCloudproviderDelegate), nil
	}
	s := session.GetSession(ctx, userCred)
	result, err := modules.Cloudproviders.Get(s, id, nil)
	if err != nil {
		return nil, errors.Wrap(err, "modules.Cloudproviders.Get")
	}
	provider := &SCloudproviderDelegate{}
	err = result.Unmarshal(provider)
	if err != nil {
		return nil, errors.Wrap(err, "result.Unmarshal")
	}
	manager.providers.AtomicSet(provider.Id, provider)
	return provider, nil
}

func (provider *SCloudproviderDelegate) getPassword() (string, error) {
	return utils.DescryptAESBase64(provider.Id, provider.Secret)
}

func (provider *SCloudproviderDelegate) getAccessUrl() string {
	return provider.AccessUrl
}

func (provider *SCloudproviderDelegate) GetProviderFactory() (cloudprovider.ICloudProviderFactory, error) {
	return cloudprovider.GetProviderFactory(provider.Provider)
}

func (provider *SCloudproviderDelegate) GetProvider() (cloudprovider.ICloudProvider, error) {
	if !provider.Enabled {
		return nil, errors.Error("Cloud provider is not enabled")
	}

	accessUrl := provider.getAccessUrl()
	passwd, err := provider.getPassword()
	if err != nil {
		return nil, err
	}
	return cloudprovider.GetProvider(provider.Id, provider.Name, accessUrl, provider.Account, passwd, provider.Provider)
}
