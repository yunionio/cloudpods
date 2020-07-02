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

package monitor

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type SuggestRuleConfigListOptions struct {
	options.BaseListOptions
	options.ScopedResourceListOptions
	Type string `help:"Suggest system rule type, e.g 'EIP_UNUSED,DISK_UNUSED'"`
}

func (o *SuggestRuleConfigListOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := o.BaseListOptions.Params()
	if err != nil {
		return nil, err
	}
	scopeParams, err := o.ScopedResourceListOptions.Params()
	if err != nil {
		return nil, err
	}
	params.Update(scopeParams)
	if o.Type != "" {
		params.Add(jsonutils.NewString(o.Type), "type")
	}
	return params, nil
}

type SuggestRuleConfigShowOptions struct {
	ID string `help:"ID or name of the suggest config" json:"-"`
}

type SuggestRuleConfigSupportTypesOptions struct {
}

type SuggestRuleConfigCreateOptions struct {
	NAME         string `help:"Name of the suggest config"`
	Scope        string `help:"Resource scope" choices:"system|domain|project" default:"project"`
	Domain       string `help:"'Owner domain ID or Name" json:"project_domain"`
	Project      string `help:"'Owner project ID or Name" json:"project"`
	Type         string `help:"Suggest system rule type, e.g 'EIP_UNUSED,DISK_UNUSED'"`
	IgnoreAlert  bool   `help:"Ignore suggest system alert result"`
	ResourceId   string `help:"Resource id"`
	ResourceType string `help:"Resource Type"`
}

func (o *SuggestRuleConfigCreateOptions) Params() (*jsonutils.JSONDict, error) {
	input := &monitor.SuggestSysRuleConfigCreateInput{}
	input.Name = o.NAME
	if o.Type != "" {
		t := monitor.SuggestDriverType(o.Type)
		input.Type = &t
	}
	if o.Domain != "" {
		input.ProjectDomain = o.Domain
	}
	if o.Project != "" {
		input.Project = o.Project
	}
	if o.ResourceId != "" {
		input.ResourceId = &o.ResourceId
		if o.ResourceType == "" {
			return nil, errors.Errorf("resource type must specifid")
		}
		resType := monitor.MonitorResourceType(o.ResourceType)
		input.ResourceType = &resType
	}
	input.Scope = o.Scope
	if o.IgnoreAlert {
		input.IgnoreAlert = true
	}
	return input.JSON(input), nil
}

type SuggestRuleConfigIdOptions struct {
	ID string `help:"Id or name of the suggest config"`
}

type SuggestRuleConfigIdsOptions struct {
	ID []string `help:"Id or name of the suggest config"`
}
