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

package remotefile

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	CLOUD_PROVIDER_REMOTEFILE = api.CLOUD_PROVIDER_REMOTEFILE
)

type RemoteFileClientConfig struct {
	cpcfg cloudprovider.ProviderConfig

	url      string
	username string
	password string

	client *http.Client

	projects  []SProject
	regions   []SRegion
	zones     []SZone
	wires     []SWire
	networks  []SNetwork
	storages  []SStorage
	disks     []SDisk
	hosts     []SHost
	vpcs      []SVpc
	vms       []SInstance
	buckets   []SBucket
	eips      []SEip
	rds       []SDBInstance
	lbs       []SLoadbalancer
	misc      []SMisc
	secgroups []SSecurityGroup
	metrics   []map[cloudprovider.TMetricType]map[string]interface{}

	debug bool
}

func NewRemoteFileClientConfig(url, username, password string) *RemoteFileClientConfig {
	cfg := &RemoteFileClientConfig{
		url:      url,
		username: username,
		password: password,
	}
	return cfg
}

func (cfg *RemoteFileClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *RemoteFileClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *RemoteFileClientConfig) Debug(debug bool) *RemoteFileClientConfig {
	cfg.debug = debug
	return cfg
}

type SRemoteFileClient struct {
	*RemoteFileClientConfig
	lock sync.Mutex
}

func NewRemoteFileClient(cfg *RemoteFileClientConfig) (*SRemoteFileClient, error) {
	cli := &SRemoteFileClient{
		RemoteFileClientConfig: cfg,
	}
	cli.client = httputils.GetDefaultClient()
	if cli.cpcfg.ProxyFunc != nil {
		httputils.SetClientProxyFunc(cli.client, cli.cpcfg.ProxyFunc)
	}
	_, err := cli.GetRegions()
	return cli, err
}

func (cli *SRemoteFileClient) GetCloudRegionExternalIdPrefix() string {
	return fmt.Sprintf("%s/%s/", CLOUD_PROVIDER_REMOTEFILE, cli.cpcfg.Id)
}

func (cli *SRemoteFileClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	subAccount := cloudprovider.SSubAccount{
		Account: cli.cpcfg.Id,
		Name:    cli.cpcfg.Name,
		Id:      cli.cpcfg.Id,

		HealthStatus: api.CLOUD_PROVIDER_HEALTH_NORMAL,
	}
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (self *SRemoteFileClient) _url(res string) string {
	return fmt.Sprintf("%s/%s.json", self.url, res)
}

func (self *SRemoteFileClient) get(res string) (jsonutils.JSONObject, error) {
	_, resp, err := httputils.JSONRequest(self.client, context.Background(), httputils.GET, self._url(res), nil, nil, self.debug)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (self *SRemoteFileClient) GetRegions() ([]SRegion, error) {
	if len(self.regions) > 0 {
		return self.regions, nil
	}
	self.regions = []SRegion{}
	resp, err := self.get("regions")
	if err != nil {
		return nil, err
	}
	return self.regions, resp.Unmarshal(&self.regions)
}

func (self *SRemoteFileClient) GetVpcs() ([]SVpc, error) {
	if len(self.vpcs) > 0 {
		return self.vpcs, nil
	}
	self.vpcs = []SVpc{}
	resp, err := self.get("vpcs")
	if err != nil {
		return nil, err
	}
	return self.vpcs, resp.Unmarshal(&self.vpcs)
}

func (self *SRemoteFileClient) GetMisc() ([]SMisc, error) {
	if len(self.misc) > 0 {
		return self.misc, nil
	}
	self.misc = []SMisc{}
	resp, err := self.get("misc")
	if err != nil {
		return nil, err
	}
	return self.misc, resp.Unmarshal(&self.misc)
}

func (self *SRemoteFileClient) GetSecgroups() ([]SSecurityGroup, error) {
	if len(self.secgroups) > 0 {
		return self.secgroups, nil
	}
	self.secgroups = []SSecurityGroup{}
	resp, err := self.get("secgroups")
	if err != nil {
		return nil, err
	}
	return self.secgroups, resp.Unmarshal(&self.secgroups)
}

func (self *SRemoteFileClient) GetInstances() ([]SInstance, error) {
	if len(self.vms) > 0 {
		return self.vms, nil
	}
	self.vms = []SInstance{}
	resp, err := self.get("instances")
	if err != nil {
		return nil, err
	}
	return self.vms, resp.Unmarshal(&self.vms)
}

func (self *SRemoteFileClient) GetBuckets() ([]SBucket, error) {
	if len(self.buckets) > 0 {
		return self.buckets, nil
	}
	self.buckets = []SBucket{}
	resp, err := self.get("buckets")
	if err != nil {
		return nil, err
	}
	return self.buckets, resp.Unmarshal(&self.buckets)
}

func (self *SRemoteFileClient) GetEips() ([]SEip, error) {
	if len(self.eips) > 0 {
		return self.eips, nil
	}
	self.eips = []SEip{}
	resp, err := self.get("eips")
	if err != nil {
		return nil, err
	}
	return self.eips, resp.Unmarshal(&self.eips)
}

func (self *SRemoteFileClient) GetDBInstances() ([]SDBInstance, error) {
	if len(self.rds) > 0 {
		return self.rds, nil
	}
	self.rds = []SDBInstance{}
	resp, err := self.get("dbinstances")
	if err != nil {
		return nil, err
	}
	return self.rds, resp.Unmarshal(&self.rds)
}

func (self *SRemoteFileClient) GetLoadbalancers() ([]SLoadbalancer, error) {
	if len(self.lbs) > 0 {
		return self.lbs, nil
	}
	self.lbs = []SLoadbalancer{}
	resp, err := self.get("loadbalancers")
	if err != nil {
		return nil, err
	}
	return self.lbs, resp.Unmarshal(&self.lbs)
}

func (self *SRemoteFileClient) GetIRegions() []cloudprovider.ICloudRegion {
	ret := []cloudprovider.ICloudRegion{}
	for i := range self.regions {
		self.regions[i].client = self
		ret = append(ret, &self.regions[i])
	}
	return ret
}

func (self *SRemoteFileClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	regions := self.GetIRegions()
	for i := range regions {
		if regions[i].GetGlobalId() == id {
			return regions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRemoteFileClient) GetZones() ([]SZone, error) {
	if len(self.zones) > 0 {
		return self.zones, nil
	}
	self.zones = []SZone{}
	resp, err := self.get("zones")
	if err != nil {
		return nil, err
	}
	return self.zones, resp.Unmarshal(&self.zones)
}

func (self *SRemoteFileClient) GetHosts() ([]SHost, error) {
	if len(self.hosts) > 0 {
		return self.hosts, nil
	}
	self.hosts = []SHost{}
	resp, err := self.get("hosts")
	if err != nil {
		return nil, err
	}
	return self.hosts, resp.Unmarshal(&self.hosts)
}

func (self *SRemoteFileClient) GetStorages() ([]SStorage, error) {
	if len(self.storages) > 0 {
		return self.storages, nil
	}
	self.storages = []SStorage{}
	resp, err := self.get("storages")
	if err != nil {
		return nil, err
	}
	return self.storages, resp.Unmarshal(&self.storages)
}

func (self *SRemoteFileClient) getMetrics(resourceType cloudprovider.TResourceType, metricType cloudprovider.TMetricType) ([]cloudprovider.MetricValues, error) {
	ret := []cloudprovider.MetricValues{}
	self.lock.Lock()
	defer self.lock.Unlock()

	if self.metrics == nil {
		self.metrics = []map[cloudprovider.TMetricType]map[string]interface{}{}
		resp, err := self.get("metrics")
		if err != nil {
			return nil, err
		}
		err = resp.Unmarshal(&self.metrics)
		if err != nil {
			return nil, err
		}
	}

	for _, metric := range self.metrics {
		values, ok := metric[metricType]
		if ok {
			mid := cloudprovider.MetricValues{}
			jsonutils.Update(&mid, values)
			ret = append(ret, mid)
		}
	}

	return ret, nil
}

func (self *SRemoteFileClient) GetWires() ([]SWire, error) {
	if len(self.wires) > 0 {
		return self.wires, nil
	}
	self.wires = []SWire{}
	resp, err := self.get("wires")
	if err != nil {
		return nil, err
	}
	return self.wires, resp.Unmarshal(&self.wires)
}

func (self *SRemoteFileClient) GetNetworks() ([]SNetwork, error) {
	if len(self.networks) > 0 {
		return self.networks, nil
	}
	self.networks = []SNetwork{}
	resp, err := self.get("networks")
	if err != nil {
		return nil, err
	}
	return self.networks, resp.Unmarshal(&self.networks)
}

func (self *SRemoteFileClient) GetDisks() ([]SDisk, error) {
	if len(self.disks) > 0 {
		return self.disks, nil
	}
	self.disks = []SDisk{}
	resp, err := self.get("disks")
	if err != nil {
		return nil, err
	}
	return self.disks, resp.Unmarshal(&self.disks)
}

func (self *SRemoteFileClient) GetProjects() ([]SProject, error) {
	if len(self.projects) > 0 {
		return self.projects, nil
	}
	resp, err := self.get("projects")
	if err != nil {
		return nil, err
	}
	self.projects = []SProject{}
	return self.projects, resp.Unmarshal(&self.projects)
}

func (self *SRemoteFileClient) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	projects, err := self.GetProjects()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudProject{}
	for i := range projects {
		ret = append(ret, &projects[i])
	}
	return ret, nil
}

func (self *SRemoteFileClient) GetCapabilities() []string {
	caps := []string{
		cloudprovider.CLOUD_CAPABILITY_PROJECT + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_COMPUTE + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_NETWORK + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_SECURITY_GROUP + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_EIP + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_LOADBALANCER + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_QUOTA + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_RDS + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_CACHE + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_MISC + cloudprovider.READ_ONLY_SUFFIX,
	}
	return caps
}
