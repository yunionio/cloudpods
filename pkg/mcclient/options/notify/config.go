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

package notify

import (
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type ConfigListOptions struct {
	options.BaseListOptions
	Type        string `json:"type"`
	Attribution string `json:"attribution"`
}

func (cl *ConfigListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(cl)
}

type ConfigCreateOptions struct {
	NAME        string   `help:"Name of Config"`
	Domain      string   `help:"which domain create for, required if attribution is 'domain'"`
	Type        string   `help:"The type of config"`
	Configs     []string `help:"Config content, format: 'key:value'"`
	Attribution string   `help:"Attribution" choices:"system|domain"`
}

func (cc *ConfigCreateOptions) Params() (jsonutils.JSONObject, error) {
	jo := jsonutils.Marshal(cc)
	d := jo.(*jsonutils.JSONDict)
	d.Remove("configs")
	configs := jsonutils.NewDict()
	for _, kv := range cc.Configs {
		index := strings.IndexByte(kv, ':')
		configs.Set(kv[:index], jsonutils.NewString(kv[index+1:]))
	}
	d.Set("content", configs)
	return d, nil
}

type ConfigOptions struct {
	ID string
}

func (c *ConfigOptions) GetId() string {
	return c.ID
}

func (c *ConfigOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type ConfigUpdateOptions struct {
	ConfigOptions
	Configs []string
}

func (cu *ConfigUpdateOptions) Params() (jsonutils.JSONObject, error) {
	configs := jsonutils.NewDict()
	for _, kv := range cu.Configs {
		index := strings.IndexByte(kv, ':')
		configs.Set(kv[:index], jsonutils.NewString(kv[index+1:]))
	}
	d := jsonutils.NewDict()
	d.Set("content", configs)
	return d, nil
}

type ConfigCapabilityOptions struct {
}

func (c *ConfigCapabilityOptions) Property() string {
	return "capability"
}

func (c *ConfigCapabilityOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}
