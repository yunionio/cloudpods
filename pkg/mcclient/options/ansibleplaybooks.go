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

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/util/ansible"
)

type AnsiblePlaybookIdOptions struct {
	ID string `help:"name/id of the playbook"`
}

type AnsiblePlaybookListOptions struct {
	BaseListOptions
}

type AnsiblePlaybookCommonOptions struct {
	Host []string `help:"name or id of server or host in format '<[server:]id|host:id>|ipaddr var=val'"`
	Mod  []string `help:"ansible modules and their arguments in format 'name k1=v1 k2=v2'"`
}

func (opts *AnsiblePlaybookCommonOptions) params() (jsonutils.JSONObject, error) {
	if len(opts.Mod) == 0 {
		return nil, fmt.Errorf("Requires at least one --mod argument")
	}
	if len(opts.Host) == 0 {
		return nil, fmt.Errorf("Requires at least one server/host to operate on")
	}
	pb := ansible.NewPlaybook()
	hosts := []ansible.Host{}
	for _, s := range opts.Host {
		host, err := ansible.ParseHostLine(s)
		if err != nil {
			return nil, err
		}
		hosts = append(hosts, host)
	}
	pb.Inventory = ansible.Inventory{Hosts: hosts}
	for _, s := range opts.Mod {
		module, err := ansible.ParseModuleLine(s)
		if err != nil {
			return nil, err
		}
		pb.Modules = append(pb.Modules, module)
	}
	pbJson := jsonutils.Marshal(pb)
	return pbJson, nil
}

type AnsiblePlaybookCreateOptions struct {
	NAME string `help:"name of the playbook"`
	AnsiblePlaybookCommonOptions
}

func (opts *AnsiblePlaybookCreateOptions) Params() (*jsonutils.JSONDict, error) {
	pbJson, err := opts.AnsiblePlaybookCommonOptions.params()
	if err != nil {
		return nil, err
	}
	params := jsonutils.NewDict()
	params.Set("playbook", pbJson)
	params.Set("name", jsonutils.NewString(opts.NAME))
	return params, nil
}

type AnsiblePlaybookUpdateOptions struct {
	ID string `json:"-" help:"name/id of the playbook"`
	AnsiblePlaybookCommonOptions
}

func (opts *AnsiblePlaybookUpdateOptions) Params() (*jsonutils.JSONDict, error) {
	pbJson, err := opts.AnsiblePlaybookCommonOptions.params()
	if err != nil {
		return nil, err
	}
	params := jsonutils.NewDict()
	params.Set("playbook", pbJson)
	return params, nil
}
