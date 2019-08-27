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

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type ExternalProjectListOptions struct {
		options.BaseListOptions
	}
	R(&ExternalProjectListOptions{}, "external-project-list", "List public cloud projects", func(s *mcclient.ClientSession, opts *ExternalProjectListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}

		result, err := modules.ExternalProjects.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.ExternalProjects.GetColumns(s))
		return nil
	})

	type ExternalProjectShowOptions struct {
		ID string `help:"ID"`
	}
	R(&ExternalProjectShowOptions{}, "external-project-show", "Show details of project mapping", func(s *mcclient.ClientSession, args *ExternalProjectShowOptions) error {
		info, err := modules.ExternalProjects.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(info)
		return nil
	})

	type ExternalProjectUpdateOptions struct {
		ID      string `help:"ExternalProject ID or Name"`
		PROJECT string `help:"Local project ID or Name"`
	}

	R(&ExternalProjectUpdateOptions{}, "external-project-change-project", "Change external project point to local project", func(s *mcclient.ClientSession, args *ExternalProjectUpdateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.PROJECT), "project")
		result, err := modules.ExternalProjects.PerformAction(s, args.ID, "change-project", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
