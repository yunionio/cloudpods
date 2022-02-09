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
	"time"

	"golang.org/x/net/http/httpproxy"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudmon/options"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	k8s_modules "yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

type CloudReportBase struct {
	SProvider *SProvider
	Session   *mcclient.ClientSession
	Args      *options.ReportOptions
	Operator  string
	Impl      ICloudReportK8s
}

func (self *CloudReportBase) Report() error {
	return fmt.Errorf("No Implment the method")
}

func (self *CloudReportBase) GetResourceByOperator() ([]jsonutils.JSONObject, error) {
	var servers []jsonutils.JSONObject
	var err error
	switch MonType(self.Operator) {
	case REDIS:
		servers, err = self.GetAllserverOfThisProvider(&modules.ElasticCache, nil)
	case RDS:
		servers, err = self.GetAllserverOfThisProvider(&modules.DBInstance, nil)
	case OSS:
		servers, err = self.GetAllserverOfThisProvider(&modules.Buckets, nil)
	case ELB:
		query := jsonutils.NewDict()
		query.Add(jsonutils.NewString("0"), KEY_LIMIT)
		query.Add(jsonutils.NewString("true"), KEY_ADMIN)
		query.Add(jsonutils.NewString(self.SProvider.Provider), "provider")
		query.Add(jsonutils.NewString(self.SProvider.Id), "manager")
		servers, err = self.GetAllserverOfThisProvider(&modules.Loadbalancers, query)
	case K8S:
		query := jsonutils.NewDict()
		query.Add(jsonutils.NewString("0"), KEY_LIMIT)
		query.Add(jsonutils.NewString("true"), KEY_ADMIN)
		query.Add(jsonutils.NewString(self.SProvider.Provider), "provider")
		query.Add(jsonutils.NewString(self.SProvider.Id), "manager")
		servers, err = self.GetAllserverOfThisProvider(k8s_modules.KubeClusters, query)
	case SERVER:
		servers, err = self.GetAllserverOfThisProvider(&modules.Servers, nil)
	default:
		return []jsonutils.JSONObject{}, nil
	}
	return servers, err
}

func (self *CloudReportBase) GetAllserverOfThisProvider(manager modulebase.Manager, query *jsonutils.JSONDict) ([]jsonutils.JSONObject, error) {
	if query == nil {
		query = jsonutils.NewDict()
		query.Add(jsonutils.NewStringArray([]string{"running", "ready"}), "status")
		query.Add(jsonutils.NewString("0"), KEY_LIMIT)
		query.Add(jsonutils.NewString("true"), KEY_ADMIN)
		query.Add(jsonutils.NewString(self.SProvider.Provider), "provider")
		query.Add(jsonutils.NewString(self.SProvider.Id), "manager")
	}
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
	options, err := cloudAccout.Get("options")
	if err != nil {
		log.Errorf("get cloudAccout options err:%v", err)
	}
	cfg := cloudprovider.ProviderConfig{
		Id:        self.SProvider.Id,
		Name:      self.SProvider.Name,
		URL:       self.SProvider.AccessUrl,
		Account:   self.SProvider.Account,
		Secret:    secretDe,
		Vendor:    self.SProvider.Provider,
		ProxyFunc: proxyFunc,
	}
	if options != nil {
		cfg.Options = options.(*jsonutils.JSONDict)
		defaultRegion, _ := options.GetString("default_region")
		cfg.DefaultRegion = defaultRegion

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
			cloudregionExternalId := self.getCloudregionExternalId(server)
			if cloudregionExternalId == nil {
				return nil, nil, err
			}
			region_external_id = *cloudregionExternalId
		}
		if _, ok := extranleIdMap[region_external_id]; !ok {
			extranleIdMap[region_external_id] = ""
			region, err := providerInstance.GetIRegionById(region_external_id)
			if err != nil {
				name, _ := server.GetString("name")
				log.Errorf("name:%s,region_external_id:%s,err:%v", name, region_external_id, err)
				continue
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

func (self *CloudReportBase) getCloudregionExternalId(res jsonutils.JSONObject) *string {
	cloudregionId, _ := res.GetString("cloudregion_id")
	cloudregion, err := self.GetResourceById(cloudregionId, &modules.Cloudregions)
	if err != nil {
		return nil
	}
	external_id, _ := cloudregion.GetString("external_id")
	return &external_id
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
		metric, err := JsonToMetric(server.(*jsonutils.JSONDict), "", ServerTags, make([]string, 0))
		if err != nil {
			return metric, err
		}
		self.AddMetricTag(&metric, OtherVmTags)
		return metric, nil
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
	case K8S:
		return JsonToMetric(server.(*jsonutils.JSONDict), "", K8sTags, make([]string, 0))
	}
	return influxdb.SMetricData{}, fmt.Errorf("no found report operator")
}

func (self *CloudReportBase) CollectRegionMetric(region cloudprovider.ICloudRegion,
	servers []jsonutils.JSONObject) error {
	panic("NO implement CollectRegionMetricOfServer")
}

func (self *CloudReportBase) ListAllResources(manager modulebase.Manager,
	query *jsonutils.JSONDict) ([]jsonutils.JSONObject, error) {

	return ListAllResources(manager, self.Session, query)
}

func ListAllResources(manager modulebase.Manager, session *mcclient.ClientSession,
	query *jsonutils.JSONDict) ([]jsonutils.JSONObject, error) {
	offsetIndex := 0
	resources := make([]jsonutils.JSONObject, 0)
	tryTimes := 5
	i := 0
	for {
		i++
		query.Add(jsonutils.NewInt(int64(offsetIndex)), "offset")
		resList, err := manager.List(session, query)
		if err != nil {
			if i <= tryTimes {
				time.Sleep(3 * time.Second)
				continue
			}
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

func ListK8sClusterModuleResources(typ K8sClusterModuleType, clusterId string, session *mcclient.ClientSession, query *jsonutils.JSONDict) ([]jsonutils.JSONObject, error) {
	var manager modulebase.Manager
	switch typ {
	case K8S_MODULE_POD:
		manager = k8s_modules.Pods
	case K8S_MODULE_DEPLOY:
		manager = k8s_modules.Deployments
	case K8S_MODULE_NODE:
		manager = k8s_modules.K8sNodes
	case K8S_MODULE_DAEMONSET:
		manager = k8s_modules.DaemonSets
	default:
		return nil, fmt.Errorf("K8sClusterModuleType: %s is not support", string(typ))
	}
	if query == nil {
		query = jsonutils.NewDict()
	}
	query.Set("cluster", jsonutils.NewString(clusterId))
	query.Set("scope", jsonutils.NewString("system"))
	return ListAllResources(manager, session, query)
}

func (self *CloudReportBase) CollectRegionMetricOfK8sModules(region cloudprovider.ICloudRegion,
	clusters []jsonutils.JSONObject) error {
	errs := make([]error, 0)
	for _, cluster := range clusters {
		helper, err := GetK8sClusterHelper(self.SProvider.Provider)
		if err != nil {
			log.Errorf("GetK8sClusterHelper err: %v", err)
			return err
		}
		moduleHelpers := helper.MyModuleHelper()
		for _, moduleHelper := range moduleHelpers {
			err := self.Impl.CollectK8sModuleMetric(region, cluster, moduleHelper)
			if err != nil {
				errs = append(errs, errors.Errorf("k8s moduleType: %s collectK8sModuleMetric err: %v", moduleHelper.MyModuleType(), err))
			}
		}
	}
	return errors.NewAggregate(errs)
}
