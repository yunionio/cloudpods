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

package k8s

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type ClusterListOptions struct {
	options.BaseListOptions
	// federated resource keyword, e.g: federatednamespace
	FederatedKeyword    string `json:"federated_keyword"`
	FederatedResourceId string `json:"federated_resource_id"`
	FederatedUsed       *bool  `json:"-"`
	FederatedUnused     *bool  `json:"-"`
	ManagerId           string `json:"manager_id"`
	CloudRegionId       string `json:"cloudregion_id"`
}

func (o *ClusterListOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.ListStructToParams(o)
	if err != nil {
		return nil, err
	}
	if o.FederatedUnused != nil {
		params.Add(jsonutils.JSONFalse, "federated_used")
	}
	if o.FederatedUsed != nil {
		params.Add(jsonutils.JSONTrue, "federated_used")
	}
	return params, nil
}

type AddMachineOptions struct {
	Machine           []string `help:"Machine create desc, e.g. host01:baremetal:controlplane"`
	MachineNet        string   `help:"Machine net config"`
	MachineDisk       string   `help:"Machine root disk size, e.g. 100G"`
	MachineCpu        int      `help:"Machine cpu count"`
	MachineMemory     string   `help:"Machine memory size, e.g. 1G"`
	MachineSku        string   `help:"Machine sku, e.g. 'ecs.c6.large'"`
	MachineHypervisor string   `help:"Machine hypervisor, e.g. kvm, openstack"`
}

type K8SClusterCreateOptions struct {
	NAME string `help:"Name of cluster"`
	File string `help:"JSON file of create body"`
	// ClusterType  string `help:"Cluster cluster type" choices:"default|serverless"`
	ResourceType string `help:"Cluster cluster type" choices:"host|guest"`
	// CloudType         string `help:"Cluster cloud type" choices:"private|public|hybrid"`
	Mode              string `help:"Cluster mode type" choices:"customize|import"`
	Provider          string `help:"Cluster provider" choices:"onecloud|system"`
	ServiceCidr       string `help:"Cluster service CIDR, e.g. 10.43.0.0/16"`
	ServiceDomain     string `help:"Cluster service domain, e.g. cluster.local"`
	Vip               string `help:"Cluster api server static loadbalancer vip"`
	Version           string `help:"Cluster kubernetes version" choices:"v1.17.0|v1.19.0|v1.22.9"`
	ImageRepo         string `help:"Image repository, e.g. registry-1.docker.io/yunion"`
	ImageRepoInsecure bool   `help:"Image repostiory is insecure"`
	Vpc               string `help:"Cluster nodes network vpc"`

	// AddMachineOptions include create machine options
	AddMachineOptions
	// Addons options
	EnableNativeIPAlloc bool `help:"Calico CNI plugin enable native ip allocation"`
}

func parseMachineDesc(
	desc string,
	disk string,
	netConf string,
	ncpu int,
	memorySize string,
	sku string,
	hypervisor string,
) (*MachineCreateOptions, error) {
	matchType := func(p string) bool {
		switch p {
		case "baremetal", "vm":
			return true
		default:
			return false
		}
	}
	matchRole := func(p string) bool {
		switch p {
		case "controlplane", "node":
			return true
		default:
			return false
		}
	}
	mo := new(MachineCreateOptions)
	for _, part := range strings.Split(desc, ":") {
		switch {
		case matchType(part):
			mo.Type = part
		case matchRole(part):
			mo.ROLE = part
		default:
			mo.Instance = part
		}
	}
	if mo.ROLE == "" {
		return nil, fmt.Errorf("Machine role is empty")
	}
	if mo.Type == "" {
		return nil, fmt.Errorf("Machine type is empty")
	}
	mo.Disk = disk
	mo.Cpu = ncpu
	mo.Memory = memorySize
	mo.Sku = sku
	mo.Net = netConf
	mo.Hypervisor = hypervisor
	return mo, nil
}

type K8SClusterAddonNetworkConfig struct {
	EnableNativeIPAlloc bool `json:"enable_native_ip_alloc"`
}

type K8SClusterAddonConfig struct {
	Network K8SClusterAddonNetworkConfig `json:"network"`
}

func (o K8SClusterCreateOptions) getAddonsConfig() (jsonutils.JSONObject, error) {
	conf := &K8SClusterAddonConfig{
		Network: K8SClusterAddonNetworkConfig{
			EnableNativeIPAlloc: o.EnableNativeIPAlloc,
		},
	}
	return jsonutils.Marshal(conf), nil
}

func (o K8SClusterCreateOptions) Params() (jsonutils.JSONObject, error) {
	if o.File != "" {
		content, err := os.ReadFile(o.File)
		if err != nil {
			return nil, errors.Wrapf(err, "read file: %s", o.File)
		}
		obj, err := jsonutils.Parse(content)
		if err != nil {
			return nil, errors.Wrapf(err, "parse file %s content: %s", o.File, content)
		}
		return obj, nil
	}
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(o.NAME), "name")
	/*
	 * if o.ClusterType != "" {
	 *     params.Add(jsonutils.NewString(o.ClusterType), "cluster_type")
	 * }
	 */
	if o.ResourceType != "" {
		params.Add(jsonutils.NewString(o.ResourceType), "resource_type")
	}
	/*
	 * if o.CloudType != "" {
	 *     params.Add(jsonutils.NewString(o.CloudType), "cloud_type")
	 * }
	 */
	if o.Mode != "" {
		params.Add(jsonutils.NewString(o.Mode), "mode")
	}
	if o.Provider != "" {
		params.Add(jsonutils.NewString(o.Provider), "provider")
	}
	if o.ServiceCidr != "" {
		params.Add(jsonutils.NewString(o.ServiceCidr), "service_cidr")
	}
	if o.ServiceDomain != "" {
		params.Add(jsonutils.NewString(o.ServiceDomain), "service_domain")
	}
	if o.Vip != "" {
		params.Add(jsonutils.NewString(o.Vip), "vip")
	}
	if o.Vpc != "" {
		params.Add(jsonutils.NewString(o.Vpc), "vpc_id")
	}
	if o.Version != "" {
		params.Add(jsonutils.NewString(o.Version), "version")
	}
	imageRepo := jsonutils.NewDict()
	if o.ImageRepo != "" {
		imageRepo.Add(jsonutils.NewString(o.ImageRepo), "url")
	}
	if o.ImageRepoInsecure {
		imageRepo.Add(jsonutils.JSONTrue, "insecure")
	}
	machineObjs, err := o.AddMachineOptions.Params()
	if err != nil {
		return nil, err
	}
	params.Add(machineObjs, "machines")
	params.Add(imageRepo, "image_repository")

	addonsConf, err := o.getAddonsConfig()
	if err != nil {
		return nil, errors.Wrap(err, "get addons config")
	}
	params.Add(addonsConf, "addons_config")
	return params, nil
}

type KubeClusterImportOptions struct {
	NAME             string `help:"Name of cluster"`
	KUBECONFIG       string `help:"Cluster kubeconfig file path"`
	Distro           string `help:"Kubernetes distribution, e.g. openshift"`
	Provider         string `help:"Provider type" choices:"external|aliyun|qcloud|azure"`
	ResourceType     string `help:"Node resource type" choices:"unknown|guest"`
	CloudKubeCluster string `help:"Cloud kube cluster id or name"`
}

func (o KubeClusterImportOptions) Params() (jsonutils.JSONObject, error) {
	kubeconfig, err := ioutil.ReadFile(o.KUBECONFIG)
	if err != nil {
		return nil, fmt.Errorf("Read kube config %q error: %v", o.KUBECONFIG, err)
	}
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(o.NAME), "name")
	params.Add(jsonutils.NewString("import"), "mode")
	params.Add(jsonutils.NewString(o.Provider), "provider")
	if o.ResourceType == "" {
		o.ResourceType = "unknown"
	}
	params.Add(jsonutils.NewString(o.ResourceType), "resource_type")
	if o.CloudKubeCluster != "" {
		params.Add(jsonutils.NewString(o.CloudKubeCluster), "external_cluster_id")
	}

	importData := jsonutils.NewDict()
	importData.Add(jsonutils.NewString(string(kubeconfig)), "kubeconfig")
	if o.Distro != "" {
		importData.Add(jsonutils.NewString(o.Distro), "distribution")
	}
	params.Add(importData, "import_data")
	return params, nil
}

type ClusterGCOpts struct{}

func (o ClusterGCOpts) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type IdentOptions struct {
	ID string `help:"ID or name of the model"`
}

func (o IdentOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

func (o IdentOptions) GetId() string {
	return o.ID
}

type ClusterPurgeOptions struct {
	IdentOptions
	Force bool `help:"force purge"`
}

func (o ClusterPurgeOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	force := jsonutils.JSONFalse
	if o.Force {
		force = jsonutils.JSONTrue
	}
	params.Add(force, "force")
	return params, nil
}

type ClusterSyncOptions struct {
	IdentOptions
	Force bool `help:"force sync"`
}

func (o ClusterSyncOptions) Params() (jsonutils.JSONObject, error) {
	param := jsonutils.NewDict()
	if o.Force {
		param.Add(jsonutils.JSONTrue, "force")
	}
	return param, nil
}

type ClusterDeployOptions struct {
	IdentOptions
	Force  bool   `help:"force deploy"`
	Action string `help:"deploy action" choices:"run|upgrade-master-config"`
}

func (o ClusterDeployOptions) Params() (jsonutils.JSONObject, error) {
	param := jsonutils.NewDict()
	if o.Force {
		param.Add(jsonutils.JSONTrue, "force")
	}
	if o.Action != "" {
		param.Add(jsonutils.NewString(o.Action), "action")
	}
	return param, nil
}

type ClusterK8sVersions struct {
	PROVIDER string `help:"cluster provider" choices:"system|onecloud"`
}

type ClusterCheckOptions struct{}

type IdentsOptions struct {
	ID []string `help:"ID of models to operate"`
}

func (o IdentsOptions) GetIds() []string {
	return o.ID
}

func (o IdentsOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type ClusterDeleteOptions struct {
	IdentsOptions
}

type KubeClusterAddMachinesOptions struct {
	IdentOptions
	AddMachineOptions
}

func (o AddMachineOptions) Params() (jsonutils.JSONObject, error) {
	machineObjs := jsonutils.NewArray()
	if len(o.Machine) == 0 {
		return machineObjs, nil
	}
	for _, m := range o.Machine {
		machine, err := parseMachineDesc(m, o.MachineDisk, o.MachineNet, o.MachineCpu, o.MachineMemory, o.MachineSku, o.MachineHypervisor)
		if err != nil {
			return nil, err
		}
		params, err := machine.Params()
		if err != nil {
			return nil, err
		}
		machineObjs.Add(params)
	}
	return machineObjs, nil
}

func (o KubeClusterAddMachinesOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	machinesArray, err := o.AddMachineOptions.Params()
	if err != nil {
		return nil, err
	}
	params.Add(machinesArray, "machines")
	return params, nil
}

type KubeClusterDeleteMachinesOptions struct {
	IdentOptions
	Machines []string `help:"Machine id or name"`
}

func (o KubeClusterDeleteMachinesOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	machinesArray := jsonutils.NewArray()
	for _, m := range o.Machines {
		machinesArray.Add(jsonutils.NewString(m))
	}
	params.Add(machinesArray, "machines")
	return params, nil
}

type ClusterComponentOptions struct {
	IdentOptions
}

func (o ClusterComponentOptions) Params(typ string) *jsonutils.JSONDict {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(typ), "type")
	return params
}

type ClusterComponentType struct {
	ClusterComponentOptions
	TYPE string `help:"Component type"`
}

type ClusterComponentTypeOptions struct {
	ClusterComponentType
	AsHelmValues bool `help:"As helm values config" json:"as_helm_values"`
}

type ClusterEnableComponentCephCSIOpt struct {
	ClusterComponentOptions
	ClusterId string   `help:"Ceph cluster id"`
	Monitor   []string `help:"Ceph monitor, format is 'ip:port'"`
}

func (o ClusterEnableComponentCephCSIOpt) Params() (*jsonutils.JSONDict, error) {
	params := o.ClusterComponentOptions.Params("cephCSI")
	conf := jsonutils.NewDict()
	clusterConfs := jsonutils.NewArray()
	clusterConf := jsonutils.NewDict()
	clusterConf.Add(jsonutils.NewString(o.ClusterId), "clusterId")
	mons := jsonutils.NewArray()
	for _, m := range o.Monitor {
		mons.Add(jsonutils.NewString(m))
	}
	clusterConf.Add(mons, "monitors")
	clusterConfs.Add(clusterConf)
	conf.Add(clusterConfs, "config")
	params.Add(conf, "cephCSI")
	return params, nil
}

type ClusterComponentStorage struct {
	Enabled   bool   `help:"Enable this storage" json:"enabled"`
	SizeMB    int    `help:"Persistent storage size MB" json:"sizeMB"`
	ClassName string `help:"Storage class name" json:"storageClassName"`
}

type ClusterComponentMonitorGrafanaTlsOpt struct {
	CertificateFile string `help:"TLS certificate file" json:"-"`
	KeyFile         string `help:"TLS key file" json:"-"`
}

type ClusterComponentMonitorGrafanaOAuth struct {
	Enabled           bool   `help:"Enable oauth setting" json:"enabled"`
	ClientId          string `help:"Client id" json:"clientId"`
	ClientSecret      string `help:"Client secret" json:"clientSecret"`
	Scopes            string `help:"Client scopes" json:"scopes"`
	AuthURL           string `help:"Auth url" json:"authURL"`
	TokenURL          string `help:"Token url" json:"tokenURL"`
	ApiURL            string `help:"API URL" json:"apiURL"`
	AllowedDomains    string `help:"Allowed domains" json:"allowedDomains"`
	AllowSignUp       bool   `help:"Allow sign up" json:"allowSignUp"`
	RoleAttributePath string `help:"Role attribute path" json:"roleAttributePath"`
}

type ClusterComponentMonitorGrafanaDB struct {
	Host     string `help:"db host" json:"host"`
	Port     int    `help:"db port" json:"port"`
	Database string `help:"db name" json:"database"`
	Username string `help:"db username" json:"username"`
	Password string `help:"db password" json:"password"`
}

type ClusterComponentMonitorGrafana struct {
	Disable           bool                                 `help:"Disable grafana component" json:"disable"`
	AdminUser         string                               `help:"Grafana admin user" default:"admin" json:"adminUser"`
	AdminPassword     string                               `help:"Grafana admin user password" json:"adminPassword"`
	Storage           ClusterComponentStorage              `help:"Storage setting"`
	PublicAddress     string                               `help:"Grafana expose public IP address or domain hostname" json:"publicAddress"`
	Host              string                               `help:"Grafana ingress host domain name" json:"host"`
	EnforceDomain     bool                                 `help:"Enforce use domain" json:"enforceDomain"`
	Tls               ClusterComponentMonitorGrafanaTlsOpt `help:"TLS setting"`
	DisableSubpath    bool                                 `help:"Disable grafana subpath" json:"disableSubpath"`
	Subpath           string                               `help:"Grafana subpath" default:"grafana" json:"subpath"`
	EnableThanosQuery bool                                 `help:"Enable thanos query datasource" json:"enableThanosQueryDataSource"`
	Oauth             ClusterComponentMonitorGrafanaOAuth  `help:"OAuth config" json:"oauth"`
	Db                ClusterComponentMonitorGrafanaDB     `help:"db config" json:"db"`
}

type ObjectStoreConfig struct {
	Bucket    string `json:"bucket"`
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Insecure  bool   `json:"insecure"`
}

type ClusterComponentMonitorLoki struct {
	Disable           bool                    `help:"Disable loki component" json:"disable"`
	Storage           ClusterComponentStorage `help:"Storage setting" json:"storage"`
	ObjectStoreConfig ObjectStoreConfig       `json:"objectStoreConfig"`
}

type MonitorPrometheusThanosSidecar struct {
	ObjectStoreConfig ObjectStoreConfig `json:"objectStoreConfig"`
}

type ClusterComponentMonitorPrometheus struct {
	Disable bool                           `help:"Disable prometheus component" json:"disable"`
	Storage ClusterComponentStorage        `help:"Storage setting" json:"storage"`
	Thanos  MonitorPrometheusThanosSidecar `json:"thanosSidecar"`
}

type ClusterComponentMonitorPromtail struct {
	Disable bool `help:"Disable promtail component" json:"disable"`
}

type ClusterComponentMonitorSetting struct {
	Grafana    ClusterComponentMonitorGrafana    `help:"Grafana setting" json:"grafana"`
	Loki       ClusterComponentMonitorLoki       `help:"Loki setting" json:"loki"`
	Prometheus ClusterComponentMonitorPrometheus `help:"Prometheus setting" json:"prometheus"`
	Promtail   ClusterComponentMonitorPromtail   `help:"Promtail setting" json:"promtail"`
}

type ClusterEnableComponentMonitorOpt struct {
	ClusterComponentOptions
	ClusterComponentMonitorSetting
}

func (o ClusterEnableComponentMonitorOpt) Params() (jsonutils.JSONObject, error) {
	params := o.ClusterComponentOptions.Params("monitor")
	setting := jsonutils.Marshal(o.ClusterComponentMonitorSetting)
	params.Add(setting, "monitor")
	certFile := o.Grafana.Tls.CertificateFile
	keyFile := o.Grafana.Tls.KeyFile
	if certFile != "" {
		cert, err := ioutil.ReadFile(certFile)
		if err != nil {
			return nil, errors.Wrap(err, "read grafana tls certFile")
		}
		key, err := ioutil.ReadFile(keyFile)
		if err != nil {
			return nil, errors.Wrap(err, "read grafana tls keyFile")
		}
		params.Add(jsonutils.NewString(string(cert)), "monitor", "grafana", "tlsKeyPair", "certificate")
		params.Add(jsonutils.NewString(string(key)), "monitor", "grafana", "tlsKeyPair", "key")
	}
	return params, nil
}

type ClusterComponentFluentBitBackendCommon struct {
	Enabled bool `help:"Enable this component"`
}

func (o ClusterComponentFluentBitBackendCommon) Params() (*jsonutils.JSONDict, error) {
	ret := jsonutils.NewDict()
	if o.Enabled {
		ret.Add(jsonutils.JSONTrue, "enabled")
	}
	return ret, nil
}

type ClusterComponentFluentBitBackendTLS struct {
	Tls       bool `help:"Enable TLS support"`
	TlsVerify bool `help:"Enable TLS validation"`
}

func (o ClusterComponentFluentBitBackendTLS) Params() (*jsonutils.JSONDict, error) {
	ret := jsonutils.NewDict()
	if o.Tls {
		ret.Add(jsonutils.JSONTrue, "tls")
	}
	if o.TlsVerify {
		ret.Add(jsonutils.JSONTrue, "tlsVerify")
	}
	return ret, nil
}

type ClusterComponentFluentBitBackendES struct {
	ClusterComponentFluentBitBackendCommon
	ClusterComponentFluentBitBackendTLS
	Host       string `help:"IP address or hostname of the target Elasticsearch instance"`
	Port       int    `help:"TCP port of the target Elasticsearch instance" default:"9200"`
	Index      string `help:"Elastic idnex name" default:"fluentbit"`
	Type       string `help:"Elastic type name" default:"flb_type"`
	HTTPUser   string `help:"Optional username credential for Elastic X-Pack access"`
	HTTPPasswd string `help:"Password for user defined in HTTPUser"`
}

func (o ClusterComponentFluentBitBackendES) Params() (*jsonutils.JSONDict, error) {
	ret, err := o.ClusterComponentFluentBitBackendCommon.Params()
	if err != nil {
		return nil, err
	}
	if o.Host == "" {
		return nil, fmt.Errorf("host must specified")
	}
	ret.Add(jsonutils.NewString(o.Host), "host")
	ret.Add(jsonutils.NewInt(int64(o.Port)), "port")
	ret.Add(jsonutils.NewString(o.Index), "index")
	ret.Add(jsonutils.NewString(o.Type), "type")
	if o.HTTPUser != "" {
		ret.Add(jsonutils.NewString(o.HTTPUser), "httpUser")
		ret.Add(jsonutils.NewString(o.HTTPPasswd), "httpPassword")
	}
	tlsParams, err := o.ClusterComponentFluentBitBackendTLS.Params()
	if err != nil {
		return nil, err
	}
	ret.Update(tlsParams)
	return ret, nil
}

type ClusterComponentFluentBitBackendKafka struct {
	ClusterComponentFluentBitBackendCommon
	Brokers  string `help:"Single of multiple list of Kafka Brokers, e.g: 192.168.1.3:9092, 192.168.1.4:9092"`
	Topics   string `help:"Single entry or list of topics separated by comma (,) that Fluent Bit will use to send messages to Kafka" default:"fluent-bit"`
	TopicKey string `help:"If multiple Topics exists, the value of TopicKey in the record will indicate the topic to use."`
}

func (o ClusterComponentFluentBitBackendKafka) Params() (*jsonutils.JSONDict, error) {
	if o.Brokers == "" {
		return nil, fmt.Errorf("brokers is empty")
	}
	brokers := strings.Split(o.Brokers, ",")
	if len(brokers) == 0 {
		return nil, fmt.Errorf("invaild brokers %s", o.Brokers)
	}
	ret, err := o.ClusterComponentFluentBitBackendCommon.Params()
	if err != nil {
		return nil, err
	}
	ret.Add(jsonutils.NewStringArray(brokers), "brokers")
	topics := strings.Split(o.Topics, ",")
	if len(topics) == 0 {
		return nil, fmt.Errorf("invalid topics %s", o.Topics)
	}
	ret.Add(jsonutils.NewStringArray(topics), "topics")
	if len(o.TopicKey) != 0 {
		ret.Add(jsonutils.NewString(o.TopicKey), "topicKey")
	}
	return ret, nil
}

type ClusterComponentFluentBitBackend struct {
	Es    ClusterComponentFluentBitBackendES    `help:"Elasticsearch setting"`
	Kafka ClusterComponentFluentBitBackendKafka `help:"Kafka setting"`
}

func (o ClusterComponentFluentBitBackend) Params() (*jsonutils.JSONDict, error) {
	ret := jsonutils.NewDict()
	if o.Es.Enabled {
		es, err := o.Es.Params()
		if err != nil {
			return nil, errors.Wrap(err, "es config")
		}
		ret.Add(es, "es")
	}
	if o.Kafka.Enabled {
		kafka, err := o.Kafka.Params()
		if err != nil {
			return nil, errors.Wrap(err, "kafka config")
		}
		ret.Add(kafka, "kafka")
	}
	return ret, nil
}

type ClusterComponentFluentBitSetting struct {
	Backend ClusterComponentFluentBitBackend
}

func (o ClusterComponentFluentBitSetting) Params() (*jsonutils.JSONDict, error) {
	backend, err := o.Backend.Params()
	if err != nil {
		return nil, errors.Wrap(err, "backend config")
	}
	ret := jsonutils.NewDict()
	ret.Add(backend, "backend")
	return ret, nil
}

type ClusterEnableComponentFluentBitOpt struct {
	ClusterComponentOptions
	ClusterComponentFluentBitSetting
}

func (o ClusterEnableComponentFluentBitOpt) Params() (*jsonutils.JSONDict, error) {
	params := o.ClusterComponentOptions.Params("fluentbit")
	setting, err := o.ClusterComponentFluentBitSetting.Params()
	if err != nil {
		return nil, err
	}
	params.Add(setting, "fluentbit")
	return params, nil
}

type ClusterDisableComponent struct {
	ClusterComponentOptions
	TYPE string `help:"component type"`
}

func (o ClusterDisableComponent) Params() *jsonutils.JSONDict {
	p := o.ClusterComponentOptions.Params(o.TYPE)
	return p
}

type ClusterUpdateComponentCephCSIOpt struct {
	ClusterComponentOptions
	ClusterId string   `help:"Ceph cluster id"`
	Monitor   []string `help:"Ceph monitor, format is 'ip:port'"`
}

func (o ClusterUpdateComponentCephCSIOpt) Params() (*jsonutils.JSONDict, error) {
	params := o.ClusterComponentOptions.Params("cephCSI")
	conf := jsonutils.NewDict()
	clusterConfs := jsonutils.NewArray()
	clusterConf := jsonutils.NewDict()
	if o.ClusterId != "" {
		clusterConf.Add(jsonutils.NewString(o.ClusterId), "clusterId")
	}
	mons := jsonutils.NewArray()
	for _, m := range o.Monitor {
		mons.Add(jsonutils.NewString(m))
	}
	if mons.Length() != 0 {
		clusterConf.Add(mons, "monitors")
	}
	clusterConfs.Add(clusterConf)
	conf.Add(clusterConfs, "config")
	params.Add(conf, "cephCSI")
	return params, nil
}

type ClusterComponentMinioSetting struct {
	Mode          string                  `help:"MinIO mode" choices:"standalone|distributed" json:"mode"`
	Replicas      int                     `help:"MinIO replicas" default:"1" json:"replicas"`
	DrivesPerNode int                     `help:"MinIO drives per node" default:"1" json:"drivesPerNode"`
	AccessKey     string                  `help:"MinIO admin access key" json:"accessKey"`
	SecretKey     string                  `help:"MinIO admin secret key" json:"secretKey"`
	MountPath     string                  `help:"MinIO export mount path" json:"mountPath"`
	Storage       ClusterComponentStorage `help:"Storage setting" json:"storage"`
}

type ClusterEnableComponentMinioBaseOpt struct {
	ClusterComponentOptions
	ClusterComponentMinioSetting
}

func (o ClusterEnableComponentMinioBaseOpt) Params(typ string) (jsonutils.JSONObject, error) {
	params := o.ClusterComponentOptions.Params(typ)
	setting := jsonutils.Marshal(o.ClusterComponentMinioSetting)
	params.Add(setting, typ)
	return params, nil
}

type ClusterEnableComponentMinioOpt struct {
	ClusterEnableComponentMinioBaseOpt
}

func (o ClusterEnableComponentMinioOpt) Params() (jsonutils.JSONObject, error) {
	return o.ClusterEnableComponentMinioBaseOpt.Params("minio")
}

type ClusterEnableComponentMonitorMinioOpt struct {
	ClusterEnableComponentMinioBaseOpt
}

func (o ClusterEnableComponentMonitorMinioOpt) Params() (jsonutils.JSONObject, error) {
	return o.ClusterEnableComponentMinioBaseOpt.Params("monitorMinio")
}

type ComponentThanosDnsDiscovery struct {
	SidecarsService   string `help:"Sidecars service name to discover them using DNS discovery" default:"prometheus-operated" json:"sidecarsService"`
	SidecarsNamespace string `help:"Sidecars namespace to discover them using DNS discovery" default:"onecloud-monitoring" json:"sidecarsNamespace"`
}

type ComponentThanosQuery struct {
	DnsDiscovery ComponentThanosDnsDiscovery `json:"dnsDiscovery"`
	Stores       []string                    `help:"Statically configure store APIs to connect with Thanos" json:"stores"`
}

type ComponentThanosCompactor struct {
	Storage ClusterComponentStorage `json:"storage"`
}

type ComponentThanosStoregateway struct {
	Storage ClusterComponentStorage `json:"storage"`
}

type ClusterComponentThanosSetting struct {
	ClusterDomain     string                      `json:"clusterDomain"`
	ObjectStoreConfig ObjectStoreConfig           `json:"objectStoreConfig"`
	Query             ComponentThanosQuery        `json:"query"`
	Store             ComponentThanosStoregateway `json:"store"`
	Compactor         ComponentThanosCompactor    `json:"compactor"`
}

type ClusterEnableComponentThanosOpt struct {
	ClusterComponentOptions
	ClusterComponentThanosSetting
}

func (o ClusterEnableComponentThanosOpt) Params() (jsonutils.JSONObject, error) {
	params := o.ClusterComponentOptions.Params("thanos")
	setting := jsonutils.Marshal(o.ClusterComponentThanosSetting)
	params.Add(setting, "thanos")
	return params, nil
}

type ClusterGetAddonsOpt struct {
	IdentOptions
	EnableNativeIPAlloc bool `json:"enable_native_ip_alloc"`
}

func (o ClusterGetAddonsOpt) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewBool(o.EnableNativeIPAlloc), "enable_native_ip_alloc")

	return params, nil
}

type ClusterSetExtraConfigOpt struct {
	IdentOptions
	DockerRegistryMirrors    []string `json:"docker_registry_mirrors"`
	DockerInsecureRegistries []string `json:"docker_insecure_registries"`
}

func (o ClusterSetExtraConfigOpt) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}
