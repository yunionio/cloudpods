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
	"io/ioutil"
	"strings"

	"yunion.io/x/jsonutils"

	apis "yunion.io/x/onecloud/pkg/apis/ansible"
	"yunion.io/x/onecloud/pkg/util/ansible"
)

type DevtoolTemplateIdOptions struct {
	ID string `help:"name/id of the playbook"`
}

type DevtoolTemplateBindingOptions struct {
	DevtoolTemplateIdOptions
	ServerID string `help:"host/vm name/id to apply"`
}

type DevtoolTemplateListOptions struct {
	BaseListOptions
}

type DevtoolTemplateCronjobOptions struct {
	Day      int   `help:"Cronjob runs at given day" default:"0"`
	Hour     int   `help:"Cronjob runs at given hour" default:"0"`
	Min      int   `help:"Cronjob runs at given min" default:"0"`
	Sec      int   `help:"Cronjob runs at given sec" default:"0"`
	Interval int64 `help:"Cronjob runs at given interval" default:"0"`
	Start    bool  `help:"start job when created" default:"false"`
	Enabled  bool  `help:"Set job status enabled" default:"false"`
}

type DevtoolTemplateCommonOptions struct {
	DevtoolTemplateCronjobOptions
	Host []string `help:"name or id of server or host in format '<[server:]id|host:id>|ipaddr var=val'"`
	Mod  []string `help:"ansible modules and their arguments in format 'name k1=v1 k2=v2'"`
	File []string `help:"files for use by modules, e.g. name=content, name=@file"`
}

func (opts *DevtoolTemplateCommonOptions) ToPlaybook() (*ansible.Playbook, error) {
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
	files := map[string][]byte{}
	for _, s := range opts.File {
		i := strings.IndexByte(s, '=')
		if i < 0 {
			return nil, fmt.Errorf("missing '=' in argument for --file.  Read command help")
		}
		name := strings.TrimSpace(s[:i])
		if name == "" {
			return nil, fmt.Errorf("empty file name: %s", s)
		}
		v := s[i+1:]
		if len(v) > 0 && v[0] == '@' {
			path := v[1:]
			d, err := ioutil.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("read file %s: %v", path, err)
			}
			files[name] = d
		} else {
			files[name] = []byte(v)
		}
	}
	pb.Files = files
	return pb, nil
}

type DevtoolTemplateCreateOptions struct {
	NAME string `help:"name of the playbook"`
	DevtoolTemplateCommonOptions
}

func (opts *DevtoolTemplateCreateOptions) Params() (*jsonutils.JSONDict, error) {
	pb, err := opts.DevtoolTemplateCommonOptions.ToPlaybook()
	if err != nil {
		return nil, err
	}
	input := &apis.AnsiblePlaybookCreateInput{
		Name:     opts.NAME,
		Playbook: *pb,
	}
	params := input.JSON(input)
	params.Add(jsonutils.NewInt(int64(opts.Day)), "day")
	params.Add(jsonutils.NewInt(int64(opts.Hour)), "hour")
	params.Add(jsonutils.NewInt(int64(opts.Min)), "min")
	params.Add(jsonutils.NewInt(int64(opts.Sec)), "sec")
	params.Add(jsonutils.NewInt(opts.Interval), "interval")
	params.Add(jsonutils.NewBool(opts.Start), "start")
	params.Add(jsonutils.NewBool(opts.Enabled), "enabled")
	return params, nil
}

type DevtoolTemplateUpdateOptions struct {
	ID   string `json:"-" help:"name/id of the playbook"`
	Name string
	DevtoolTemplateCommonOptions
	Rebind bool `help:"Unbind and Bind all related servers" default:"false"`
}

func (opts *DevtoolTemplateUpdateOptions) Params() (*jsonutils.JSONDict, error) {
	pb, err := opts.DevtoolTemplateCommonOptions.ToPlaybook()
	if err != nil {
		return nil, err
	}
	input := &apis.AnsiblePlaybookUpdateInput{
		Name:     opts.Name,
		Playbook: *pb,
	}
	params := input.JSON(input)
	params.Add(jsonutils.NewBool(opts.Rebind), "rebind")
	params.Add(jsonutils.NewInt(int64(opts.Day)), "day")
	params.Add(jsonutils.NewInt(int64(opts.Hour)), "hour")
	params.Add(jsonutils.NewInt(int64(opts.Min)), "min")
	params.Add(jsonutils.NewInt(int64(opts.Sec)), "sec")
	params.Add(jsonutils.NewInt(opts.Interval), "interval")
	params.Add(jsonutils.NewBool(opts.Start), "start")
	params.Add(jsonutils.NewBool(opts.Enabled), "enabled")
	return params, nil
}
