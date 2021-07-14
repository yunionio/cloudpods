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

package common

import (
	"fmt"
	"net/http"
	"net/url"

	"golang.org/x/net/http/httpproxy"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

type CloudReportBase struct {
	SProvider *SProvider
	Session   *mcclient.ClientSession
	Args      *ReportOptions
	Operator  string
}

func (self *CloudReportBase) Report() error {
	return fmt.Errorf("No Implment the method")
}

func (self *CloudReportBase) GetAllserverOfThisProvider(manager modulebase.Manager) ([]jsonutils.JSONObject, error) {
	query := jsonutils.NewDict()

	query.Add(jsonutils.NewStringArray([]string{"running", "ready"}), "status")
	query.Add(jsonutils.NewString("0"), KEY_LIMIT)
	query.Add(jsonutils.NewString("true"), KEY_ADMIN)
	query.Add(jsonutils.NewString(self.SProvider.Provider), "provider")
	query.Add(jsonutils.NewString(self.SProvider.Id), "manager")
	return self.ListAllResources(manager, query)
}

func (self *CloudReportBase) GetAllHostOfThisProvider(manager modulebase.Manager) ([]jsonutils.JSONObject, error) {
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString("running"), "status")
	query.Add(jsonutils.NewString("0"), KEY_LIMIT)
	query.Add(jsonutils.NewString("true"), KEY_ADMIN)
	query.Add(jsonutils.NewString(self.SProvider.Provider), "provider")
	query.Add(jsonutils.NewString(self.SProvider.Id), "manager")
	return self.ListAllResources(manager, query)
}

func (self *CloudReportBase) GetAllCloudAccount(manager modulebase.Manager) ([]jsonutils.JSONObject, error) {
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString("0"), KEY_LIMIT)
	query.Add(jsonutils.NewBool(true), DETAILS)
	query.Add(jsonutils.NewString("true"), KEY_ADMIN)
	query.Add(jsonutils.NewString(fmt.Sprintf("brand.in(%s,%s,%s,%s)", compute.CLOUD_PROVIDER_ALIYUN,
		compute.CLOUD_PROVIDER_QCLOUD, compute.CLOUD_PROVIDER_HUAWEI, compute.CLOUD_PROVIDER_JDCLOUD)), "filter")
	return self.ListAllResources(manager, query)
}

func (self *CloudReportBase) GetAllStorage(manager modulebase.Manager) ([]jsonutils.JSONObject, error) {
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString("0"), KEY_LIMIT)
	query.Add(jsonutils.NewBool(true), DETAILS)
	query.Add(jsonutils.NewString("system"), KEY_SCOPE)
	return self.ListAllResources(manager, query)
}

func (self *CloudReportBase) ListAllResource(manager modulebase.Manager,
	query jsonutils.JSONObject) ([]jsonutils.JSONObject, error) {
	resources := make([]jsonutils.JSONObject, 0)
	offsetIndex := 0
	for {
		query.(*jsonutils.JSONDict).Add(jsonutils.NewInt(int64(offsetIndex)), "offset")
		resList, err := manager.List(self.Session, query)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resList.Data...)
		offsetIndex = offsetIndex + len(resList.Data)
		if offsetIndex >= resList.Total {
			break
		}
	}
	return resources, nil
}

func (self *CloudReportBase) InitProviderInstance() (cloudprovider.ICloudProvider, error) {
	//secret进行Descrypt
	secretDe, _ := utils.DescryptAESBase64(self.SProvider.Id, self.SProvider.Secret)
	var proxyFunc httputils.TransportProxyFunc
	{
		cfg := &httpproxy.Config{
			HTTPProxy:  self.SProvider.ProxySetting.HTTPProxy,
			HTTPSProxy: self.SProvider.ProxySetting.HTTPSProxy,
			NoProxy:    self.SProvider.ProxySetting.NoProxy,
		}
		cfgProxyFunc := cfg.ProxyFunc()
		proxyFunc = func(req *http.Request) (*url.URL, error) {
			return cfgProxyFunc(req.URL)
		}
	}
	cloudAccout, err := self.getCloudAccount(&modules.Cloudaccounts)
	if err != nil {
		return nil, errors.Wrap(err, "getCloudAccount error")
	}
	endpoints := cloudprovider.SApsaraEndpoints{}
	options, err := cloudAccout.Get("options")
	if err == nil {
		err := options.Unmarshal(&endpoints)
		if err != nil {
			log.Errorf("Unmarshal SApsaraEndpoints err:%v", err)
		}
	} else {
		log.Errorf("get cloudAccout options err:%v", err)
	}
	cfg := cloudprovider.ProviderConfig{
		Id:               self.SProvider.Id,
		Name:             self.SProvider.Name,
		URL:              self.SProvider.AccessUrl,
		Account:          self.SProvider.Account,
		Secret:           secretDe,
		Vendor:           self.SProvider.Provider,
		ProxyFunc:        proxyFunc,
		SApsaraEndpoints: endpoints,
	}
	return cloudprovider.GetProvider(cfg)
}

func (self *CloudReportBase) getCloudAccount(manager modulebase.Manager) (jsonutils.JSONObject, error) {
	return self.GetResourceById(self.SProvider.CloudaccountId, manager)
}

func (self *CloudReportBase) GetResourceById(id string, manager modulebase.Manager) (jsonutils.JSONObject, error) {
	return manager.Get(self.Session, id, jsonutils.NewDict())
}

func (self *CloudReportBase) GetAllRegionOfServers(servers []jsonutils.JSONObject,
	providerInstance cloudprovider.ICloudProvider) ([]cloudprovider.
	ICloudRegion, map[string][]jsonutils.JSONObject, error) {
	extranleIdMap := make(map[string]string)
	regionServerList := make([]cloudprovider.ICloudRegion, 0)
	regionServerMap := make(map[string][]jsonutils.JSONObject)
	for i, server := range servers {
		region_external_id, err := server.GetString("region_external_id")
		if err != nil {
			return nil, nil, err
		}
		if _, ok := extranleIdMap[region_external_id]; !ok {
			extranleIdMap[region_external_id] = ""
			region, err := providerInstance.GetIRegionById(region_external_id)
			if err != nil {
				return nil, nil, err
			}
			regionServerList = append(regionServerList, region)
			regionServers := make([]jsonutils.JSONObject, 0)
			regionServers = append(regionServers, servers[i])
			regionServerMap[region_external_id] = regionServers
		} else {
			regionservers := regionServerMap[region_external_id]
			regionservers = append(regionservers, server)
			regionServerMap[region_external_id] = regionservers
		}
	}
	return regionServerList, regionServerMap, nil
}

func (self *CloudReportBase) AddMetricTag(metric *influxdb.SMetricData, tags map[string]string) {
	for key, value := range tags {
		metric.Tags = append(metric.Tags, influxdb.SKeyValue{
			Key:   key,
			Value: value,
		})
	}
}

func (self *CloudReportBase) NewMetricFromJson(server jsonutils.JSONObject) (influxdb.SMetricData, error) {
	switch MonType(self.Operator) {
	case SERVER:
		return JsonToMetric(server.(*jsonutils.JSONDict), "", ServerTags, make([]string, 0))
	case HOST:
		return JsonToMetric(server.(*jsonutils.JSONDict), "", HostTags, make([]string, 0))
	case REDIS:
		return JsonToMetric(server.(*jsonutils.JSONDict), "", RedisTags, make([]string, 0))
	case RDS:
		return JsonToMetric(server.(*jsonutils.JSONDict), "", RdsTags, make([]string, 0))
	case OSS:
		return JsonToMetric(server.(*jsonutils.JSONDict), "", OssTags, make([]string, 0))
	case ELB:
		return JsonToMetric(server.(*jsonutils.JSONDict), "", ElbTags, make([]string, 0))
	case CLOUDACCOUNT:
		return JsonToMetric(server.(*jsonutils.JSONDict), "", CloudAccountTags, CloudAccountFields)
	case STORAGE:
		return JsonToMetric(server.(*jsonutils.JSONDict), "", StorageTags, make([]string, 0))
	case ALERT_RECORD:
		return JsonToMetric(server.(*jsonutils.JSONDict), "", AlertRecordHistoryTags, AlertRecordHistoryFields)
	}
	return influxdb.SMetricData{}, fmt.Errorf("no found report operator")
}

func (self *CloudReportBase) CollectRegionMetric(region cloudprovider.ICloudRegion,
	servers []jsonutils.JSONObject) error {
	panic("NO implement CollectRegionMetricOfServer")
}

func (self *CloudReportBase) ListAllResources(manager modulebase.Manager,
	query *jsonutils.JSONDict) ([]jsonutils.JSONObject, error) {
	offsetIndex := 0
	resources := make([]jsonutils.JSONObject, 0)
	for {
		query.Add(jsonutils.NewInt(int64(offsetIndex)), "offset")
		resList, err := manager.List(self.Session, query)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resList.Data...)
		offsetIndex = offsetIndex + len(resList.Data)
		if offsetIndex >= resList.Total {
			break
		}
	}
	return resources, nil
}
