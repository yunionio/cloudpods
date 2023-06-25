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

package aliyun

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/apis"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SKubeCluster struct {
	multicloud.SResourceBase

	region *SRegion

	Name             string    `json:"name"`
	ClusterId        string    `json:"cluster_id"`
	Size             int       `json:"size"`
	RegionId         string    `json:"region_id"`
	State            string    `json:"state"`
	ClusterType      string    `json:"cluster_type"`
	Created          time.Time `json:"created"`
	Updated          time.Time `json:"updated"`
	InitVersion      string    `json:"init_version"`
	CurrentVersion   string    `json:"current_version"`
	MetaData         string    `json:"meta_data"`
	ResourceGroupId  string    `json:"resource_group_id"`
	InstanceType     string    `json:"instance_type"`
	VpcId            string    `json:"vpc_id"`
	VswitchId        string    `json:"vswitch_id"`
	VswitchCidr      string    `json:"vswitch_cidr"`
	DataDiskSize     int       `json:"data_disk_size"`
	DataDiskCategory string    `json:"data_disk_category"`
	SecurityGroupId  string    `json:"security_group_id"`
	Tags             []struct {
		Key   string
		Value string
	}
	ZoneId                 string `json:"zone_id"`
	NetworkMode            string `json:"network_mode"`
	SubnetCidr             string `json:"subnet_cidr"`
	MasterURL              string `json:"master_url"`
	ExternalLoadbalancerId string `json:"external_loadbalancer_id"`
	Port                   int    `json:"port"`
	NodeStatus             string `json:"node_status"`
	ClusterHealthy         string `json:"cluster_healthy"`
	DockerVersion          string `json:"docker_version"`
	SwarmMode              bool   `json:"swarm_mode"`
	GwBridge               string `json:"gw_bridge"`
	UpgradeComponents      string `json:"upgrade_components"`
	NextVersion            string `json:"next_version"`
	PrivateZone            bool   `json:"private_zone"`
	ServiceDiscoveryTypes  string `json:"service_discovery_types"`
	PrivateLink            bool   `json:"private_link"`
	Profile                string `json:"profile"`
	DeletionProtection     bool   `json:"deletion_protection"`
	ClusterSpec            string `json:"cluster_spec"`
	MaintenanceWindow      struct {
		Enable          bool   `json:"enable"`
		MaintenanceTime string `json:"maintenance_time"`
		Duration        string `json:"duration"`
		WeeklyPeriod    string `json:"weekly_period"`
	} `json:"maintenance_window"`
	Capabilities      string `json:"capabilities"`
	EnabledMigration  bool   `json:"enabled_migration"`
	NeedUpdateAgent   bool   `json:"need_update_agent"`
	Outputs           string `json:"outputs"`
	Parameters        string `json:"parameters"`
	WorkerRAMRoleName string `json:"worker_ram_role_name"`
	MaintenanceInfo   string `json:"maintenance_info"`
}

func (self *SKubeCluster) GetName() string {
	return self.Name
}

func (self *SKubeCluster) GetStatus() string {
	switch self.State {
	case "initial":
		return apis.STATUS_CREATING
	case "failed":
		return apis.STATUS_CREATE_FAILED
	case "deleting", "deleted":
		return apis.STATUS_DELETING
	case "delete_failed":
		return apis.STATUS_DELETE_FAILED
	default:
		return self.State
	}
}

func (self *SKubeCluster) GetId() string {
	return self.ClusterId
}

func (self *SKubeCluster) GetGlobalId() string {
	return self.ClusterId
}

func (self *SKubeCluster) GetEnabled() bool {
	return true
}

func (self *SKubeCluster) Refresh() error {
	cluster, err := self.region.GetKubeCluster(self.ClusterId)
	if err != nil {
		return errors.Wrapf(err, "GetKubeCluster(%s)", self.ClusterId)
	}
	return jsonutils.Update(self, cluster)
}

func (self *SKubeCluster) GetKubeConfig(private bool, expireMinutes int) (*cloudprovider.SKubeconfig, error) {
	return self.region.GetKubeConfig(self.ClusterId, private, expireMinutes)
}

func (self *SKubeCluster) GetVersion() string {
	return self.CurrentVersion
}

func (self *SKubeCluster) GetVpcId() string {
	return self.VpcId
}

func (self *SKubeCluster) GetNetworkIds() []string {
	if len(self.VswitchId) > 0 {
		return []string{self.VswitchId}
	}
	return []string{}
}

func (self *SKubeCluster) Delete(isRetain bool) error {
	return self.region.DeleteKubeCluster(self.ClusterId, isRetain)
}

func (self *SKubeCluster) GetTags() (map[string]string, error) {
	ret := map[string]string{}
	for _, tag := range self.Tags {
		ret[tag.Key] = tag.Value
	}
	return ret, nil
}

func (self *SKubeCluster) GetSysTags() map[string]string {
	return nil
}

func (self *SKubeCluster) SetTags(tags map[string]string, replace bool) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetKubeConfig(clusterId string, private bool, minutes int) (*cloudprovider.SKubeconfig, error) {
	params := map[string]string{
		"PathPattern":      fmt.Sprintf("/k8s/%s/user_config", clusterId),
		"PrivateIpAddress": fmt.Sprintf("%v", private),
	}
	if minutes >= 15 && minutes <= 4320 {
		params["TemporaryDurationMinutes"] = fmt.Sprintf("%d", minutes)
	}
	resp, err := self.k8sRequest("DescribeClusterUserKubeconfig", params, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeClusterUserKubeconfig")
	}
	result := &cloudprovider.SKubeconfig{}
	err = resp.Unmarshal(result)
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return result, nil
}

func (self *SRegion) GetKubeClusters(pageSize, pageNumber int) ([]SKubeCluster, int, error) {
	if pageSize < 1 || pageSize > 100 {
		pageSize = 100
	}
	if pageNumber < 1 {
		pageNumber = 1
	}
	params := map[string]string{
		"page_size":   fmt.Sprintf("%d", pageSize),
		"page_number": fmt.Sprintf("%d", pageNumber),
		"PathPattern": "/api/v1/clusters",
	}
	resp, err := self.k8sRequest("DescribeClustersV1", params, nil)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeClustersV1")
	}
	clusters := []SKubeCluster{}
	err = resp.Unmarshal(&clusters, "clusters")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	totalCnt, _ := resp.Int("page_info", "total_count")
	return clusters, int(totalCnt), nil
}

func (self *SRegion) GetICloudKubeClusters() ([]cloudprovider.ICloudKubeCluster, error) {
	clusters := []SKubeCluster{}
	for {
		part, total, err := self.GetKubeClusters(100, len(clusters)/100+1)
		if err != nil {
			return nil, errors.Wrapf(err, "GetKubeClusters")
		}
		clusters = append(clusters, part...)
		if len(clusters) >= total || len(part) == 0 {
			break
		}
	}
	ret := []cloudprovider.ICloudKubeCluster{}
	for i := range clusters {
		if clusters[i].RegionId == self.RegionId {
			clusters[i].region = self
			ret = append(ret, &clusters[i])
		}
	}
	return ret, nil
}

func (self *SRegion) GetICloudKubeClusterById(id string) (cloudprovider.ICloudKubeCluster, error) {
	cluster, err := self.GetKubeCluster(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetKubeCluster(%s)", id)
	}
	if cluster.RegionId != self.RegionId {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "%s at region %s", id, cluster.RegionId)
	}
	return cluster, nil
}

func (self *SRegion) GetKubeCluster(id string) (*SKubeCluster, error) {
	params := map[string]string{
		"PathPattern": fmt.Sprintf("/clusters/%s", id),
	}
	resp, err := self.k8sRequest("DescribeClusterDetail", params, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeClusterDetail")
	}
	cluster := &SKubeCluster{region: self}
	err = resp.Unmarshal(&cluster)
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return cluster, nil
}

func (self *SRegion) DeleteKubeCluster(id string, isRetain bool) error {
	params := map[string]string{
		"PathPattern": fmt.Sprintf("/clusters/%s", id),
	}
	if isRetain {
		params["retain_all_resources"] = "true"
		params["keep_slb"] = "true"
	}
	_, err := self.k8sRequest("DeleteCluster", params, nil)
	return errors.Wrapf(err, "DeleteCluster")
}

func (self *SKubeCluster) CreateIKubeNodePool(opts *cloudprovider.KubeNodePoolCreateOptions) (cloudprovider.ICloudKubeNodePool, error) {
	pool, err := self.region.CreateKubeNodePool(self.ClusterId, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateKubeNodePool")
	}
	pool.cluster = self
	return pool, nil
}

func (self *SRegion) CreateKubeNodePool(clusterId string, opts *cloudprovider.KubeNodePoolCreateOptions) (*SKubeNodePool, error) {
	keyName, err := self.syncKeypair(opts.PublicKey)
	if err != nil {
		return nil, errors.Wrapf(err, "syncKeypair")
	}
	params := map[string]interface{}{
		"kubernetes_config": map[string]interface{}{
			"cms_enabled": true,
		},
		"nodepool_info": map[string]interface{}{
			"name": opts.NAME,
		},
		"auto_scaling": map[string]interface{}{
			"enable": false,
		},
		"scaling_group": map[string]interface{}{
			"instance_types":             opts.InstanceTypes,
			"key_pair":                   keyName,
			"system_disk_category":       "cloud_efficiency",
			"internet_max_bandwidth_out": 20,
			"internet_charge_type":       "PayByTraffic",
			"system_disk_size":           opts.RootDiskSizeGb,
			"vswitch_ids":                opts.NetworkIds,
			"desired_size":               opts.DesiredInstanceCount,
		},
	}

	path := fmt.Sprintf("/clusters/%s/nodepools", clusterId)
	resp, err := self.k8sRequest("CreateClusterNodePool", map[string]string{"PathPattern": path}, params)
	if err != nil {
		return nil, err
	}
	poolId, err := resp.GetString("nodepool_id")
	if err != nil {
		return nil, errors.Wrapf(err, "get nodepool_id")
	}
	return self.GetKubeNodePool(clusterId, poolId)
}

func (self *SRegion) CreateIKubeCluster(opts *cloudprovider.KubeClusterCreateOptions) (cloudprovider.ICloudKubeCluster, error) {
	cluster, err := self.CreateKubeCluster(opts)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func (self *SRegion) CreateKubeCluster(opts *cloudprovider.KubeClusterCreateOptions) (*SKubeCluster, error) {
	tags := []struct {
		Key   string
		Value string
	}{}
	for k, v := range opts.Tags {
		tags = append(tags, struct {
			Key   string
			Value string
		}{
			Key:   k,
			Value: v,
		})
	}
	if len(opts.ServiceCIDR) == 0 {
		opts.ServiceCIDR = "192.168.0.0/16"
	}

	addons := []struct {
		Name     string
		Config   string
		Disabled bool
	}{
		{
			Name:   "terway-eniip",
			Config: `{"IPVlan":"false","NetworkPolicy":"false","ENITrunking":"false"}`,
		},
		{
			Name: "csi-plugin",
		},
		{
			Name: "csi-provisioner",
		},
		{
			Name:   "storage-operator",
			Config: `{"CnfsOssEnable":"false","CnfsNasEnable":"false"}`,
		},
		{
			Name:     "nginx-ingress-controller",
			Disabled: true,
		},
	}

	if len(opts.PublicKey) == 0 {
		return nil, fmt.Errorf("missing public_key")
	}

	keyName, err := self.syncKeypair(opts.PublicKey)
	if err != nil {
		return nil, errors.Wrapf(err, "syncKeypair")
	}

	params := map[string]interface{}{
		"name":                         opts.NAME,
		"region_id":                    self.RegionId,
		"disable_rollback":             true,
		"cluster_type":                 "ManagedKubernetes",
		"cluster_spec":                 "ack.pro.small",
		"deletion_protection":          false,
		"proxy_mode":                   "ipvs",
		"cis_enable_risk_check":        false,
		"os_type":                      "Linux",
		"platform":                     "AliyunLinux",
		"image_type":                   "AliyunLinux",
		"timezone":                     "Asia/Shanghai",
		"pod_vswitch_ids":              opts.NetworkIds,
		"vswitch_ids":                  opts.NetworkIds,
		"charge_type":                  "PostPaid",
		"vpcid":                        opts.VpcId,
		"service_cidr":                 opts.ServiceCIDR,
		"api_audiences":                "https://kubernetes.default.svc",
		"service_account_issuer":       "https://kubernetes.default.svc",
		"key_pair":                     keyName,
		"snat_entry":                   false,
		"ssh_flags":                    true,
		"tags":                         tags,
		"cloud_monitor_flags":          true,
		"is_enterprise_security_group": true,
		"num_of_nodes":                 0,
		"nodepools":                    []interface{}{},
		"addons":                       addons,
	}
	if opts.PublicAccess {
		params["endpoint_public_access"] = true
	}
	if len(opts.Version) > 0 {
		params["kubernetes_version"] = opts.Version
	}

	resp, err := self.k8sRequest("CreateCluster", map[string]string{"PathPattern": "/clusters"}, params)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateCluster")
	}
	ret := &SKubeCluster{region: self}
	return ret, resp.Unmarshal(ret)
}
