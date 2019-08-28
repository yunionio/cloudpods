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
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type SchedtagListOptions struct {
		options.BaseListOptions
		Type string `help:"Filter by resource type"`
	}
	R(&SchedtagListOptions{}, "schedtag-list", "List schedule tags", func(s *mcclient.ClientSession, args *SchedtagListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		if len(args.Type) > 0 {
			params.Add(jsonutils.NewString(args.Type), "resource_type")
		}
		result, err := modules.Schedtags.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Schedtags.GetColumns(s))
		return nil
	})

	type SchedtagShowOptions struct {
		ID string `help:"ID or Name of the scheduler tag to show"`
	}
	R(&SchedtagShowOptions{}, "schedtag-show", "Show scheduler tag details", func(s *mcclient.ClientSession, args *SchedtagShowOptions) error {
		result, err := modules.Schedtags.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&SchedtagShowOptions{}, "schedtag-delete", "Delete a scheduler tag", func(s *mcclient.ClientSession, args *SchedtagShowOptions) error {
		result, err := modules.Schedtags.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type SchedtagCreateOptions struct {
		NAME     string `help:"Name of new schedtag"`
		Strategy string `help:"Policy" choices:"require|exclude|prefer|avoid"`
		Desc     string `help:"Description"`
		Scope    string `help:"Resource scope" choices:"system|domain|project"`
		Type     string `help:"Resource type" choices:"hosts|storages|networks"`
	}
	R(&SchedtagCreateOptions{}, "schedtag-create", "Create a schedule tag", func(s *mcclient.ClientSession, args *SchedtagCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		if len(args.Strategy) > 0 {
			params.Add(jsonutils.NewString(args.Strategy), "default_strategy")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if len(args.Type) > 0 {
			params.Add(jsonutils.NewString(args.Type), "resource_type")
		}
		if len(args.Scope) > 0 {
			params.Add(jsonutils.NewString(args.Scope), "scope")
		}
		schedtag, err := modules.Schedtags.Create(s, params)
		if err != nil {
			return err
		}
		printObject(schedtag)
		return nil
	})

	type SchedtagUpdateOptions struct {
		ID            string `help:"ID or Name of schetag"`
		Name          string `help:"New name of schedtag"`
		Strategy      string `help:"Policy" choices:"require|exclude|prefer|avoid"`
		Desc          string `help:"Description"`
		ClearStrategy bool   `help:"Clear default schedule policy"`
	}
	R(&SchedtagUpdateOptions{}, "schedtag-update", "Update a schedule tag", func(s *mcclient.ClientSession, args *SchedtagUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Strategy) > 0 {
			params.Add(jsonutils.NewString(args.Strategy), "default_strategy")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if args.ClearStrategy {
			params.Add(jsonutils.NewString(""), "default_strategy")
		}
		if params.Size() == 0 {
			return fmt.Errorf("No valid data to update")
		}
		schedtag, err := modules.Schedtags.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(schedtag)
		return nil
	})

	type SetScopeOptions struct {
		ID      []string `help:"ID or Name of schetag"`
		Project string   `help:"ID or Name of project"`
		Domain  string   `help:"ID or Name of domain"`
		System  bool     `help:"Set to system scope"`
	}
	R(&SetScopeOptions{}, "schedtag-set-scope", "Set schedtag scope", func(s *mcclient.ClientSession, args *SetScopeOptions) error {
		params := jsonutils.NewDict()
		domainId := args.Domain
		projectId := args.Project
		if args.System {
			domainId = ""
			projectId = ""
		}
		params.Add(jsonutils.NewString(domainId), "domain")
		params.Add(jsonutils.NewString(projectId), "project")
		ret := modules.Schedtags.BatchPerformAction(s, args.ID, "set-scope", params)
		printBatchResults(ret, modules.Schedtags.GetColumns(s))
		return nil
	})
}
