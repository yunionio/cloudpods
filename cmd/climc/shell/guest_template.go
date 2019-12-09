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

	type GuestTemplateListOptions struct {
		options.BaseListOptions
	}

	R(&GuestTemplateListOptions{}, "server-template-list", "List server template", func(s *mcclient.ClientSession,
		opts *GuestTemplateListOptions) error {

		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.GuestTemplate.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.GuestTemplate.GetColumns(s))
		return nil
	})

	type GuestTemplateCreateOptions struct {
		options.ServerCreateOptionalOptions
		NAME string `help:"Name of server template" json:"-"`
	}

	R(&GuestTemplateCreateOptions{}, "server-template-create", "Create a server template",
		func(s *mcclient.ClientSession,
			opts *GuestTemplateCreateOptions) error {

			params, err := opts.OptionalParams()
			if err != nil {
				return err
			}
			if options.BoolV(opts.DryRun) {
				fmt.Println("no support operator")
				return nil
			}

			dict := jsonutils.NewDict()
			if opts.GenerateName {
				dict.Add(jsonutils.NewString(opts.NAME), "generate_name")
			} else {
				dict.Add(jsonutils.NewString(opts.NAME), "name")
			}
			dict.Add(params.JSON(params), "content")
			tem, err := modules.GuestTemplate.Create(s, dict)
			if err != nil {
				return err
			}
			printObject(tem)
			return nil
		})

	type GuestTemplateUpdateOptions struct {
		options.ServerCreateOptionalOptions
		ID   string `help:"ID of server template"`
		name string `help:"name of server template"`
	}

	R(&GuestTemplateUpdateOptions{}, "server-template-update", "Update a server template",
		func(s *mcclient.ClientSession, opts *GuestTemplateUpdateOptions) error {

			params, err := opts.OptionalParams()
			if err != nil {
				return err
			}
			if options.BoolV(opts.DryRun) {
				fmt.Println("no support operator")
				return nil
			}
			dict := jsonutils.NewDict()
			if len(opts.name) != 0 {
				dict.Add(jsonutils.NewString(opts.name), "name")
			}
			dict.Add(params.JSON(params), "content")
			tem, err := modules.GuestTemplate.Update(s, opts.ID, dict)
			if err != nil {
				return err
			}
			printObject(tem)
			return nil
		})

	type GuestTemplateOptions struct {
		ID string `help:"ID or Name of server template"`
	}

	R(&GuestTemplateOptions{}, "server-template-show", "Show a server template",
		func(s *mcclient.ClientSession, opts *GuestTemplateOptions) error {
			tem, err := modules.GuestTemplate.Get(s, opts.ID, jsonutils.JSONNull)
			if err != nil {
				return err
			}
			printObject(tem)
			return nil
		})

	R(&GuestTemplateOptions{}, "server-tempalte-delete", "Delete a server template",
		func(s *mcclient.ClientSession, opts *GuestTemplateOptions) error {

			tem, err := modules.GuestTemplate.Delete(s, opts.ID, jsonutils.JSONNull)
			if err != nil {
				return err
			}
			printObject(tem)
			return nil
		},
	)

	R(&GuestTemplateOptions{}, "server-template-private", "Private server template",
		func(s *mcclient.ClientSession, opts *GuestTemplateOptions) error {
			tem, err := modules.GuestTemplate.PerformAction(s, opts.ID, "private", jsonutils.JSONNull)
			if err != nil {
				return err
			}
			printObject(tem)
			return nil
		},
	)

	type GuestTemplatePublicOptions struct {
		ID          string `help:"ID or Name of server template"`
		PublicScope string `help:"public scope"`
	}

	R(&GuestTemplatePublicOptions{}, "server-template-public", "Public server template",
		func(s *mcclient.ClientSession, opts *GuestTemplatePublicOptions) error {

			dict := jsonutils.NewDict()
			if len(opts.PublicScope) != 0 {
				dict.Add(jsonutils.NewString(opts.PublicScope), "public_scope")
			}
			tem, err := modules.GuestTemplate.PerformAction(s, opts.ID, "public", dict)
			if err != nil {
				return err
			}
			printObject(tem)
			return nil
		},
	)
}
