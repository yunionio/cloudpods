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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type ClusterListOptions struct {
	options.BaseListOptions
}

func (o ClusterListOptions) Params() *jsonutils.JSONDict {
	o.Details = options.Bool(true)
	params, err := o.BaseListOptions.Params()
	if err != nil {
		panic(err)
	}
	return params
}

type AddMachineOptions struct {
	Machine           []string `help:"Machine create desc, e.g. host01:baremetal:controlplane"`
	MachineNet        string   `help:"Machine net config"`
	MachineDisk       string   `help:"Machine root disk size, e.g. 100G"`
	MachineCpu        int      `help:"Machine cpu count"`
	MachineMemory     string   `help:"Machine memory size, e.g. 1G"`
	MachineHypervisor string   `help:"Machine hypervisor, e.g. kvm, openstack"`
}

type KubeClusterCreateOptions struct {
	NAME              string `help:"Name of cluster"`
	ClusterType       string `help:"Cluster cluster type" choices:"default|serverless"`
	ResourceType      string `help:"Cluster cluster type" choices:"host|guest"`
	CloudType         string `help:"Cluster cloud type" choices:"private|public|hybrid"`
	Mode              string `help:"Cluster mode type" choices:"customize|managed|import"`
	Provider          string `help:"Cluster provider" choices:"onecloud|aws|aliyun|azure|qcloud|system"`
	ServiceCidr       string `help:"Cluster service CIDR, e.g. 10.43.0.0/16"`
	ServiceDomain     string `help:"Cluster service domain, e.g. cluster.local"`
	Vip               string `help:"Cluster api server static loadbalancer vip"`
	Version           string `help:"Cluster kubernetes version"`
	ImageRepo         string `help:"Image repository, e.g. registry-1.docker.io/yunion"`
	ImageRepoInsecure bool   `help:"Image repostiory is insecure"`

	AddMachineOptions
}

type KubeClusterImportOptions struct {
	NAME       string `help:"Name of cluster"`
	KUBECONFIG string `help:"Cluster kubeconfig file path"`
}

func parseMachineDesc(desc string, disk string, netConf string, ncpu int, memorySize string, hypervisor string) (*MachineCreateOptions, error) {
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
	mo.Net = netConf
	mo.Hypervisor = hypervisor
	return mo, nil
}

func (o KubeClusterCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(o.NAME), "name")
	if o.ClusterType != "" {
		params.Add(jsonutils.NewString(o.ClusterType), "cluster_type")
	}
	if o.ResourceType != "" {
		params.Add(jsonutils.NewString(o.ResourceType), "resource_type")
	}
	if o.CloudType != "" {
		params.Add(jsonutils.NewString(o.CloudType), "cloud_type")
	}
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
	return params, nil
}

func (o KubeClusterImportOptions) Params() (*jsonutils.JSONDict, error) {
	kubeconfig, err := ioutil.ReadFile(o.KUBECONFIG)
	if err != nil {
		return nil, fmt.Errorf("Read kube config %q error: %v", o.KUBECONFIG, err)
	}
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(o.NAME), "name")
	params.Add(jsonutils.NewString("import"), "mode")
	params.Add(jsonutils.NewString(string(kubeconfig)), "kubeconfig")
	params.Add(jsonutils.NewString("external"), "provider")
	params.Add(jsonutils.NewString("unknown"), "resource_type")
	return params, nil
}

type IdentOptions struct {
	ID string `help:"ID or name of the model"`
}

type ClusterK8sVersions struct {
	PROVIDER string `help:"cluster provider" choices:"system|onecloud"`
}

type ClusterCheckOptions struct{}

type IdentsOptions struct {
	ID []string `help:"ID of models to operate"`
}

type ClusterDeleteOptions struct {
	IdentsOptions
}

type KubeClusterAddMachinesOptions struct {
	IdentOptions
	AddMachineOptions
}

func (o AddMachineOptions) Params() (*jsonutils.JSONArray, error) {
	machineObjs := jsonutils.NewArray()
	if len(o.Machine) == 0 {
		return machineObjs, nil
	}
	for _, m := range o.Machine {
		machine, err := parseMachineDesc(m, o.MachineDisk, o.MachineNet, o.MachineCpu, o.MachineMemory, o.MachineHypervisor)
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

func (o KubeClusterAddMachinesOptions) Params() (*jsonutils.JSONDict, error) {
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

func (o KubeClusterDeleteMachinesOptions) Params() (*jsonutils.JSONDict, error) {
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

type ClusterComponentTypeOptions struct {
	IdentOptions
	TYPE string `help:"component type"`
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

type ClusterComponentMonitorStorage struct {
	Enabled   bool   `help:"Enable this storage" json:"enabled"`
	SizeMB    int    `help:"Persistent storage size MB" json:"sizeMB"`
	ClassName string `help:"Storage class name" json:"storageClassName"`
}

type ClusterComponentMonitorGrafana struct {
	AdminUser     string                         `help:"Grafana admin user" default:"admin" json:"adminUser"`
	AdminPassword string                         `help:"Grafana admin user password" json:"adminPassword"`
	Storage       ClusterComponentMonitorStorage `help:"Storage setting"`
}

type ClusterComponentMonitorLoki struct {
	Storage ClusterComponentMonitorStorage `help:"Storage setting"`
}

type ClusterComponentMonitorPrometheus struct {
	Storage ClusterComponentMonitorStorage `help:"Storage setting"`
}

type ClusterComponentMonitorPromtail struct {
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

func (o ClusterEnableComponentMonitorOpt) Params() (*jsonutils.JSONDict, error) {
	params := o.ClusterComponentOptions.Params("monitor")
	setting := jsonutils.Marshal(o.ClusterComponentMonitorSetting)
	params.Add(setting, "monitor")
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
