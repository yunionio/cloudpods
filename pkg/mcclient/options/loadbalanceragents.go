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

package options

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"

	compute_apis "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/util/ansible"
)

type LoadbalancerAgentParamsOptions struct {
	KeepalivedConfTmpl string
	HaproxyConfTmpl    string
	TelegrafConfTmpl   string

	VrrpPriority          *int // required
	VrrpVirtualRouterId   *int // required
	VrrpGarpMasterRefresh *int
	VrrpPreempt           string `choices:"true|false"`
	VrrpInterface         string // required
	VrrpAdvertInt         *int
	VrrpPass              string

	HaproxyGlobalLog      string
	HaproxyGlobalNbthread *int   `help:"enable experimental multi-threading support available since haproxy 1.8"`
	HaproxyLogHttp        string `choices:"true|false"`
	HaproxyLogTcp         string `choices:"true|false"`
	HaproxyLogNormal      string `choices:"true|false"`

	TelegrafInfluxDbOutputUrl       string
	TelegrafInfluxDbOutputName      string
	TelegrafInfluxDbOutputUnsafeSsl *bool
	TelegrafHaproxyInputInterval    int
}

func (opts *LoadbalancerAgentParamsOptions) setPrefixedParams(params *jsonutils.JSONDict, pref string) {
	pref_ := pref + "_"
	pref_len := len(pref_)
	pp := jsonutils.NewDict()
	pMap, _ := params.GetMap()
	for k, v := range pMap {
		if strings.HasPrefix(k, pref_) && !strings.HasSuffix(k, "_conf_tmpl") {
			params.Remove(k)
			pp.Set(k[pref_len:], v)
		}
	}
	if pp.Length() > 0 {
		params.Set(pref, pp)
	}
}

func (opts *LoadbalancerAgentParamsOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := optionsStructToParams(opts)
	if err != nil {
		return nil, err
	}
	opts.setPrefixedParams(params, "vrrp")
	opts.setPrefixedParams(params, "haproxy")
	opts.setPrefixedParams(params, "telegraf")
	return params, nil
}

type LoadbalancerAgentCreateOptions struct {
	NAME      string
	HbTimeout *int
	Cluster   string `required:"true"`

	LoadbalancerAgentParamsOptions
}

func (opts *LoadbalancerAgentCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := StructToParams(opts)
	if err != nil {
		return nil, err
	}
	paramsSub, err := opts.LoadbalancerAgentParamsOptions.Params()
	if err != nil {
		return nil, err
	}
	params.Set("params", paramsSub)
	return params, nil
}

type LoadbalancerAgentListOptions struct {
	BaseListOptions

	Cluster string
}

type LoadbalancerAgentGetOptions struct {
	ID string `json:"-"`
}

type LoadbalancerAgentUpdateOptions struct {
	ID   string `json:"-"`
	Name string

	HbTimeout *int

	Loadbalancers             *time.Time
	LoadbalancerListeners     *time.Time
	LoadbalancerListenerRules *time.Time
	LoadbalancerBackendGroups *time.Time
	LoadbalancerBackends      *time.Time
	LoadbalancerAcls          *time.Time
	LoadbalancerCertificates  *time.Time
}

type LoadbalancerAgentDeleteOptions struct {
	ID string `json:"-"`
}

type LoadbalancerAgentActionHbOptions struct {
	ID string `json:"-"`

	Version string
	IP      string
	HaState string
}

type LoadbalancerAgentActionPatchParamsOptions struct {
	ID string `json:"-"`

	LoadbalancerAgentParamsOptions
}

func (opts *LoadbalancerAgentActionPatchParamsOptions) Params() (*jsonutils.JSONDict, error) {
	return opts.LoadbalancerAgentParamsOptions.Params()
}

type LoadbalancerAgentActionDeployOptions struct {
	ID string `json:"-"`

	Host         string `help:"name or id of the server in format '<[server:]id|host:id>|ipaddr var=val'" json:"-"`
	DeployMethod string `help:"use yum repo or use file copy" choices:"yum|copy" default:"copy"`
}

func (opts *LoadbalancerAgentActionDeployOptions) Params() (*jsonutils.JSONDict, error) {
	host, err := ansible.ParseHostLine(opts.Host)
	if err != nil {
		return nil, fmt.Errorf("parse host: %v", err)
	}
	input := &compute_apis.LoadbalancerAgentDeployInput{
		Host:         host,
		DeployMethod: opts.DeployMethod,
	}
	params := input.JSON(input)
	return params, nil
}

type LoadbalancerAgentActionUndeployOptions struct {
	ID string `json:"-"`
}
