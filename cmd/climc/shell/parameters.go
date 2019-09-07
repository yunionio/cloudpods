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
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type ParametersListOptions struct {
		Name        string `help:"List parameter of specificated name"`
		NamespaceId string `help:"List parameter of specificated namespace id, ADMIN only"`
		User        string `help:"List parameter of specificated user id, ADMIN only" token:"user-id"`
		Service     string `help:"List parameter of specificated service id, ADMIN only"`
		options.BaseListOptions
	}

	R(&ParametersListOptions{}, "parameter-list", "list parameters", func(s *mcclient.ClientSession, args *ParametersListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}

		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}

		var result *modulebase.ListResult
		if len(args.NamespaceId) > 0 {
			params.Add(jsonutils.NewString(args.NamespaceId), "namespace_id")
			params.Add(jsonutils.NewString("system"), "scope")
			result, err = modules.Parameters.List(s, params)
		} else if len(args.User) > 0 {
			params.Add(jsonutils.NewString("system"), "scope")
			result, err = modules.Parameters.ListInContext(s, params, &modules.UsersV3, args.User)
		} else if len(args.Service) > 0 {
			params.Add(jsonutils.NewString("system"), "scope")
			result, err = modules.Parameters.ListInContext(s, params, &modules.ServicesV3, args.Service)
		} else {
			result, err = modules.Parameters.List(s, params)
		}

		if err != nil {
			return err
		}
		printList(result, modules.Parameters.GetColumns(s))
		return nil
	})

	type ParametersShowOptions struct {
		NamespaceId string `help:"Show parameter of specificated namespace id, ADMIN only"`
		User        string `help:"Show parameter of specificated user id, ADMIN only"`
		Service     string `help:"Show parameter of specificated service id, ADMIN only"`
		NAME        string `help:"The name of parameter"`
	}

	R(&ParametersShowOptions{}, "parameter-show", "show a parameter", func(s *mcclient.ClientSession, args *ParametersShowOptions) error {
		params := jsonutils.NewDict()
		/*if len(args.NamespaceId) > 0 {
			params.Add(jsonutils.JSONTrue, "admin")
			params.Add(jsonutils.NewString(args.NamespaceId), "namespace_id")
		}*/

		var parameter jsonutils.JSONObject
		var err error
		if len(args.NamespaceId) > 0 {
			params.Add(jsonutils.NewString("system"), "scope")
			params.Add(jsonutils.NewString(args.NamespaceId), "namespace_id")
			parameter, err = modules.Parameters.Get(s, args.NAME, params)
		} else if len(args.User) > 0 {
			params.Add(jsonutils.NewString("system"), "scope")
			parameter, err = modules.Parameters.GetInContext(s, args.NAME, params, &modules.UsersV3, args.User)
		} else if len(args.Service) > 0 {
			params.Add(jsonutils.NewString("system"), "scope")
			parameter, err = modules.Parameters.GetInContext(s, args.NAME, params, &modules.ServicesV3, args.Service)
		} else {
			parameter, err = modules.Parameters.Get(s, args.NAME, params)
		}

		if err != nil {
			return err
		}
		printObject(parameter)
		return nil
	})

	type ParametersCreateOptions struct {
		User    string `help:"Create parameter for specificated user id, ADMIN only"`
		Service string `help:"Create parameter for specificated service id, ADMIN only"`
		NAME    string `help:"The name of parameter"`
		VALUE   string `help:"The content of parameter"`
	}

	R(&ParametersCreateOptions{}, "parameter-create", "create a parameter", func(s *mcclient.ClientSession, args *ParametersCreateOptions) error {
		value, err := jsonutils.ParseString(args.VALUE)
		if err != nil {
			return err
		}

		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(value, "value")

		if len(args.User) > 0 {
			params.Add(jsonutils.NewString(args.User), "user_id")
		} else if len(args.Service) > 0 {
			params.Add(jsonutils.NewString(args.Service), "service_id")
		}

		parameter, err := modules.Parameters.Create(s, params)
		if err != nil {
			return err
		}
		printObject(parameter)
		return nil
	})

	type ParametersUpdateOptions struct {
		User    string `help:"Update parameter of specificated user id, ADMIN only"`
		Service string `help:"Update parameter of specificated service id, ADMIN only"`
		NAME    string `help:"The name of parameter"`
		VALUE   string `help:"The content of parameter"`
	}

	R(&ParametersUpdateOptions{}, "parameter-update", "update parameter", func(s *mcclient.ClientSession, args *ParametersUpdateOptions) error {
		var parameter jsonutils.JSONObject
		var err error
		value, err := jsonutils.ParseString(args.VALUE)
		if err != nil {
			return err
		}

		params := jsonutils.NewDict()
		if len(args.VALUE) > 0 {
			params.Add(value, "value")
		}

		if len(args.User) > 0 {
			parameter, err = modules.Parameters.PutInContext(s, args.NAME, params, &modules.UsersV3, args.User)
		} else if len(args.Service) > 0 {
			parameter, err = modules.Parameters.PutInContext(s, args.NAME, params, &modules.ServicesV3, args.Service)
		} else {
			parameter, err = modules.Parameters.Put(s, args.NAME, params)
		}

		if err != nil {
			return err
		}
		printObject(parameter)
		return nil
	})

	type ParametersDeleteOptions struct {
		User    string `help:"Delete parameter of specificated user id, ADMIN only"`
		Service string `help:"Delete parameter of specificated service id, ADMIN only"`
		NAME    string `help:"The name of parameter"`
	}

	R(&ParametersDeleteOptions{}, "parameter-delete", "delete notice", func(s *mcclient.ClientSession, args *ParametersDeleteOptions) error {
		params := jsonutils.NewDict()

		var parameter jsonutils.JSONObject
		var err error
		if len(args.User) > 0 {
			parameter, err = modules.Parameters.DeleteInContext(s, args.NAME, params, &modules.UsersV3, args.User)
		} else if len(args.Service) > 0 {
			parameter, err = modules.Parameters.DeleteInContext(s, args.NAME, params, &modules.ServicesV3, args.Service)
		} else {
			parameter, err = modules.Parameters.Delete(s, args.NAME, nil)
		}

		if err != nil {
			return err
		}
		printObject(parameter)
		return nil
	})
}
