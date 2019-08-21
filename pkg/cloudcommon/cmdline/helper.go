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

package cmdline

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/fileutils"
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/apis/scheduler"
)

type resourceConfigOutput struct {
	keyword       string
	keywordPlural string
}

func newResouceConfigOutput(kw, kws string) *resourceConfigOutput {
	return &resourceConfigOutput{keyword: kw, keywordPlural: kws}
}

func (o resourceConfigOutput) Keyword() string {
	return o.keyword
}

func (o resourceConfigOutput) KeywordPlural() string {
	return o.keywordPlural
}

type iResourcesOutput interface {
	Keyword() string
	KeywordPlural() string
	NewConfig() interface{}
	ParseDesc(desc string, idx int) (interface{}, error)
	Add(config interface{})
	Resources() interface{}
}

func fetchResourceConfigsByJSON(
	obj jsonutils.JSONObject,
	output iResourcesOutput,
) error {
	if obj == nil {
		obj = jsonutils.NewDict()
	}
	keywordPlural := output.KeywordPlural()
	keyword := output.Keyword()
	if obj.Contains(keywordPlural) {
		err := obj.Unmarshal(output.Resources(), keywordPlural)
		return err
	}

	configs := jsonutils.GetArrayOfPrefix(obj, keyword)
	for idx, config := range configs {
		configJson, ok := config.(*jsonutils.JSONDict)
		resConfig := output.NewConfig()
		if ok {
			if err := configJson.Unmarshal(resConfig); err != nil {
				return err
			}
		} else {
			configStr, err := config.GetString()
			if err != nil {
				return err
			}
			resConfig, err = output.ParseDesc(configStr, idx)
			if err != nil {
				return err
			}
		}
		output.Add(resConfig)
	}
	return nil
}

type diskConfigOutput struct {
	*resourceConfigOutput
	disks []*compute.DiskConfig
}

func newDiskConfigOutput() *diskConfigOutput {
	return &diskConfigOutput{
		resourceConfigOutput: newResouceConfigOutput("disk", "disks"),
	}
}

func (output *diskConfigOutput) Disks() []*compute.DiskConfig {
	return output.disks
}

func (output *diskConfigOutput) Resources() interface{} {
	return &output.disks
}

func (output *diskConfigOutput) NewConfig() interface{} {
	return new(compute.DiskConfig)
}

func (output *diskConfigOutput) Add(config interface{}) {
	output.disks = append(output.disks, config.(*compute.DiskConfig))
}

func (output *diskConfigOutput) ParseDesc(desc string, idx int) (interface{}, error) {
	return ParseDiskConfig(desc, idx)
}

func FetchDiskConfigsByJSON(obj jsonutils.JSONObject) ([]*compute.DiskConfig, error) {
	output := newDiskConfigOutput()
	err := fetchResourceConfigsByJSON(obj, output)
	return output.Disks(), err
}

type netConfigOutput struct {
	*resourceConfigOutput
	nets []*compute.NetworkConfig
}

func newNetworkConfigOutput() *netConfigOutput {
	return &netConfigOutput{
		resourceConfigOutput: newResouceConfigOutput("net", "nets"),
	}
}

func (output *netConfigOutput) Networks() []*compute.NetworkConfig {
	return output.nets
}

func (output *netConfigOutput) Resources() interface{} {
	return &output.nets
}

func (output *netConfigOutput) NewConfig() interface{} {
	return new(compute.NetworkConfig)
}

func (output *netConfigOutput) Add(config interface{}) {
	output.nets = append(output.nets, config.(*compute.NetworkConfig))
}

func (output *netConfigOutput) ParseDesc(desc string, idx int) (interface{}, error) {
	return ParseNetworkConfig(desc, idx)
}

func FetchNetworkConfigsByJSON(obj jsonutils.JSONObject) ([]*compute.NetworkConfig, error) {
	output := newNetworkConfigOutput()
	err := fetchResourceConfigsByJSON(obj, output)
	return output.Networks(), err
}

type schedtagConfigOutput struct {
	*resourceConfigOutput
	tags []*compute.SchedtagConfig
}

func newSchedtagConfigOutput() *schedtagConfigOutput {
	return &schedtagConfigOutput{
		resourceConfigOutput: newResouceConfigOutput("schedtag", "schedtags"),
	}
}

func (output *schedtagConfigOutput) Schedtags() []*compute.SchedtagConfig {
	return output.tags
}

func (output *schedtagConfigOutput) Resources() interface{} {
	return &output.tags
}

func (output *schedtagConfigOutput) NewConfig() interface{} {
	return new(compute.SchedtagConfig)
}

func (output *schedtagConfigOutput) Add(config interface{}) {
	output.tags = append(output.tags, config.(*compute.SchedtagConfig))
}

func (output *schedtagConfigOutput) ParseDesc(desc string, _ int) (interface{}, error) {
	return ParseSchedtagConfig(desc)
}

func FetchSchedtagConfigsByJSON(obj jsonutils.JSONObject) ([]*compute.SchedtagConfig, error) {
	output := newSchedtagConfigOutput()
	if err := fetchResourceConfigsByJSON(obj, output); err != nil {
		return nil, err
	}
	if len(output.tags) == 0 {
		// compatible with old api
		schedtags, _ := obj.GetMap("aggregate_strategy")
		if schedtags != nil {
			for id, strategyObj := range schedtags {
				strategy, _ := strategyObj.GetString()
				output.tags = append(output.tags, &compute.SchedtagConfig{
					Id:       id,
					Strategy: strategy,
				})
			}
		}
	}
	return output.tags, nil
}

type isoDevConfigOutput struct {
	*resourceConfigOutput
	devs []*compute.IsolatedDeviceConfig
}

func newIsoDevConfigOutput() *isoDevConfigOutput {
	return &isoDevConfigOutput{
		resourceConfigOutput: newResouceConfigOutput("isolated_device", "isolated_devices"),
	}
}

func (output *isoDevConfigOutput) Devs() []*compute.IsolatedDeviceConfig {
	return output.devs
}

func (output *isoDevConfigOutput) Resources() interface{} {
	return &output.devs
}

func (output *isoDevConfigOutput) NewConfig() interface{} {
	return new(compute.IsolatedDeviceConfig)
}

func (output *isoDevConfigOutput) Add(config interface{}) {
	output.devs = append(output.devs, config.(*compute.IsolatedDeviceConfig))
}

func (output *isoDevConfigOutput) ParseDesc(desc string, idx int) (interface{}, error) {
	return ParseIsolatedDevice(desc, idx)
}

func FetchIsolatedDeviceConfigsByJSON(obj jsonutils.JSONObject) ([]*compute.IsolatedDeviceConfig, error) {
	output := newIsoDevConfigOutput()
	err := fetchResourceConfigsByJSON(obj, output)
	return output.devs, err
}

type bmDiskConfigOutput struct {
	*resourceConfigOutput
	configs []*compute.BaremetalDiskConfig
}

func newbmDiskConfigOutput() *bmDiskConfigOutput {
	return &bmDiskConfigOutput{
		resourceConfigOutput: newResouceConfigOutput("baremetal_disk_config", "baremetal_disk_configs"),
	}
}

func (output *bmDiskConfigOutput) Devs() []*compute.BaremetalDiskConfig {
	return output.configs
}

func (output *bmDiskConfigOutput) Resources() interface{} {
	return &output.configs
}

func (output *bmDiskConfigOutput) NewConfig() interface{} {
	return new(compute.BaremetalDiskConfig)
}

func (output *bmDiskConfigOutput) Add(config interface{}) {
	output.configs = append(output.configs, config.(*compute.BaremetalDiskConfig))
}

func (output *bmDiskConfigOutput) ParseDesc(desc string, idx int) (interface{}, error) {
	return ParseBaremetalDiskConfig(desc)
}

func FetchBaremetalDiskConfigsByJSON(obj jsonutils.JSONObject) ([]*compute.BaremetalDiskConfig, error) {
	output := newbmDiskConfigOutput()
	err := fetchResourceConfigsByJSON(obj, output)
	return output.configs, err
}

func FetchServerConfigsByJSON(obj jsonutils.JSONObject) (*compute.ServerConfigs, error) {
	conf := new(compute.ServerConfigs)
	if err := obj.Unmarshal(conf); err != nil {
		return nil, err
	}

	if instanceType, _ := obj.GetString("sku"); instanceType != "" {
		conf.InstanceType = instanceType
	}

	var err error
	conf.Disks, err = FetchDiskConfigsByJSON(obj)
	if err != nil {
		return nil, err
	}
	conf.Networks, err = FetchNetworkConfigsByJSON(obj)
	if err != nil {
		return nil, err
	}
	conf.Schedtags, err = FetchSchedtagConfigsByJSON(obj)
	if err != nil {
		return nil, err
	}
	conf.IsolatedDevices, err = FetchIsolatedDeviceConfigsByJSON(obj)
	if err != nil {
		return nil, err
	}
	conf.BaremetalDiskConfigs, err = FetchBaremetalDiskConfigsByJSON(obj)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

func FetchScheduleInputByJSON(obj jsonutils.JSONObject) (*scheduler.ScheduleInput, error) {
	input := new(scheduler.ScheduleInput)
	err := obj.Unmarshal(input)
	if err != nil {
		return nil, err
	}
	conf := &input.ServerConfig
	if obj.Contains("scheduler") {
		obj, _ = obj.Get("scheduler")
		obj.Unmarshal(conf)
	}
	conf.ServerConfigs, err = FetchServerConfigsByJSON(obj)
	if err != nil {
		return nil, err
	}
	return input, nil
}

func FetchDeployConfigsByJSON(obj jsonutils.JSONObject) ([]*compute.DeployConfig, error) {
	deploys := make([]*compute.DeployConfig, 0)
	if obj.Contains("deploy_configs") {
		err := obj.Unmarshal(&deploys, "deploy_configs")
		return deploys, err
	}
	for idx := 0; obj.Contains(fmt.Sprintf("deploy.%d.path", idx)); idx += 1 {
		path, _ := obj.GetString(fmt.Sprintf("deploy.%d.path", idx))
		action, _ := obj.GetString(fmt.Sprintf("deploy.%d.action", idx))
		content, _ := obj.GetString(fmt.Sprintf("deploy.%d.content", idx))
		deploys = append(deploys, &compute.DeployConfig{Path: path, Action: action, Content: content})
	}
	return deploys, nil
}

func FetchServerCreateInputByJSON(obj jsonutils.JSONObject) (*compute.ServerCreateInput, error) {
	input := new(compute.ServerCreateInput)
	// compatible with vmem_size string
	memSizeObj, _ := obj.Get("vmem_size")
	if memSizeObj != nil {
		if _, ok := memSizeObj.(*jsonutils.JSONString); ok {
			memStr, err := memSizeObj.GetString()
			if err != nil {
				return nil, err
			}
			if !regutils.MatchSize(memStr) {
				return nil, fmt.Errorf("Invalid memory size string: %s, Memory size must be number[+unit], like 256M, 1G or 256", memStr)
			}
			vmemSize, err := fileutils.GetSizeMb(memStr, 'M', 1024)
			if err != nil {
				return nil, err
			}
			obj.(*jsonutils.JSONDict).Set("vmem_size", jsonutils.NewInt(int64(vmemSize)))
		}
	}

	err := obj.Unmarshal(input)
	if err != nil {
		return nil, err
	}
	config, err := FetchServerConfigsByJSON(obj)
	if err != nil {
		return nil, err
	}
	input.ServerConfigs = config

	deployConfigs, err := FetchDeployConfigsByJSON(obj)
	if err != nil {
		return nil, err
	}
	input.DeployConfigs = deployConfigs

	if skuName := jsonutils.GetAnyString(obj, []string{"sku", "flavor", "instance_type"}); skuName != "" {
		input.InstanceType = skuName
	}
	if keypair := jsonutils.GetAnyString(obj, []string{"keypair", "keypair_id"}); keypair != "" {
		input.KeypairId = keypair
	}
	if secgroup := jsonutils.GetAnyString(obj, []string{"secgroup", "secgroup_id", "secgrp_id"}); len(secgroup) != 0 {
		input.SecgroupId = secgroup
	}

	return input, nil
}

func FetchDiskCreateInputByJSON(data jsonutils.JSONObject) (*compute.DiskCreateInput, error) {
	config := new(compute.DiskConfig)
	input := &compute.DiskCreateInput{
		DiskConfig: config,
	}
	if err := data.Unmarshal(input); err != nil {
		return nil, err
	}
	if data.Contains("disk") {
		desc, err := data.GetString("disk")
		config, err = ParseDiskConfig(desc, -1)
		if err != nil {
			return nil, err
		}
		input.DiskConfig = config
	}
	if storageId := jsonutils.GetAnyString(data, []string{"storage_id", "storage"}); storageId != "" {
		input.Storage = storageId
	}
	return input, nil
}
