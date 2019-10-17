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

package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	printAnsiblePlaybookObject := func(obj jsonutils.JSONObject) {
		dict := obj.(*jsonutils.JSONDict)
		pbJson, err := dict.Get("playbook")
		if err != nil {
			printObject(obj)
			return
		}
		pbStr := pbJson.YAMLString()
		dict.Set("playbook", jsonutils.NewString(pbStr))
		printObject(obj)
	}

	type TemplateListOptions struct {
		options.BaseListOptions
		Name string `help:"cloud region ID or Name" json:"-"`
	}

	R(&TemplateListOptions{}, "devtooltemplate-list", "List Devtool Templates", func(s *mcclient.ClientSession, args *TemplateListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		var result *modulebase.ListResult
		result, err = modules.DevToolTemplates.List(s, params)
		printList(result, modules.DevToolTemplates.GetColumns(s))
		return nil
	})

	R(
		&options.DevtoolTemplateCreateOptions{},
		"devtooltemplate-create",
		"Create a template repo component",
		func(s *mcclient.ClientSession, opts *options.DevtoolTemplateCreateOptions) error {

			params, err := opts.Params()
			if err != nil {
				return err
			}
			log.Infof("devtool playbook create opts: %+v", params)
			apb, err := modules.DevToolTemplates.Create(s, params)
			if err != nil {
				return err
			}
			printAnsiblePlaybookObject(apb)
			return nil
		},
	)

	R(
		&options.DevtoolTemplateIdOptions{},
		"devtooltemplate-show",
		"Show devtool template",
		func(s *mcclient.ClientSession, opts *options.DevtoolTemplateIdOptions) error {
			apb, err := modules.DevToolTemplates.Get(s, opts.ID, nil)
			if err != nil {
				return err
			}
			printAnsiblePlaybookObject(apb)
			return nil
		},
	)

	R(
		&options.DevtoolTemplateBindingOptions{},
		"devtooltemplate-bind",
		"Binding devtool template to a host/vm",
		func(s *mcclient.ClientSession, opts *options.DevtoolTemplateBindingOptions) error {
			params := jsonutils.NewDict()
			params.Set("server_id", jsonutils.NewString(opts.ServerID))
			apb, err := modules.DevToolTemplates.PerformAction(s, opts.ID, "bind", params)
			if err != nil {
				return err
			}
			printAnsiblePlaybookObject(apb)
			return nil
		},
	)

	R(
		&options.DevtoolTemplateBindingOptions{},
		"devtooltemplate-unbind",
		"Binding devtool template to a host/vm",
		func(s *mcclient.ClientSession, opts *options.DevtoolTemplateBindingOptions) error {
			params := jsonutils.NewDict()
			params.Set("server_id", jsonutils.NewString(opts.ServerID))
			apb, err := modules.DevToolTemplates.PerformAction(s, opts.ID, "unbind", params)
			if err != nil {
				return err
			}
			printAnsiblePlaybookObject(apb)
			return nil
		},
	)

	R(
		&options.DevtoolTemplateIdOptions{},
		"devtooltemplate-delete",
		"Delete devtool template",
		func(s *mcclient.ClientSession, opts *options.DevtoolTemplateIdOptions) error {
			apb, err := modules.DevToolTemplates.Delete(s, opts.ID, nil)
			if err != nil {
				return err
			}
			printAnsiblePlaybookObject(apb)
			return nil
		},
	)

	R(
		&options.DevtoolTemplateUpdateOptions{},
		"devtooltemplate-update",
		"Update ansible playbook",
		func(s *mcclient.ClientSession, opts *options.DevtoolTemplateUpdateOptions) error {
			params, err := opts.Params()
			if err != nil {
				return err
			}
			apb, err := modules.DevToolTemplates.Update(s, opts.ID, params)
			if err != nil {
				return err
			}
			printAnsiblePlaybookObject(apb)
			return nil
		},
	)
}
