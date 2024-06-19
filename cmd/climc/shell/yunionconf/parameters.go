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

package yunionconf

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/printutils"
	"yunion.io/x/pkg/util/shellutils"

	api "yunion.io/x/onecloud/pkg/apis/yunionconf"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/mcclient/modules/yunionconf"
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

		var result *printutils.ListResult
		if len(args.NamespaceId) > 0 {
			params.Add(jsonutils.NewString(args.NamespaceId), "namespace_id")
			params.Add(jsonutils.NewString("system"), "scope")
			result, err = yunionconf.Parameters.List(s, params)
		} else if len(args.User) > 0 {
			params.Add(jsonutils.NewString("system"), "scope")
			result, err = yunionconf.Parameters.ListInContext(s, params, &identity.UsersV3, args.User)
		} else if len(args.Service) > 0 {
			params.Add(jsonutils.NewString("system"), "scope")
			result, err = yunionconf.Parameters.ListInContext(s, params, &identity.ServicesV3, args.Service)
		} else {
			result, err = yunionconf.Parameters.List(s, params)
		}

		if err != nil {
			return err
		}
		printList(result, yunionconf.Parameters.GetColumns(s))
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
			parameter, err = yunionconf.Parameters.Get(s, args.NAME, params)
		} else if len(args.User) > 0 {
			params.Add(jsonutils.NewString("system"), "scope")
			parameter, err = yunionconf.Parameters.GetInContext(s, args.NAME, params, &identity.UsersV3, args.User)
		} else if len(args.Service) > 0 {
			params.Add(jsonutils.NewString("system"), "scope")
			parameter, err = yunionconf.Parameters.GetInContext(s, args.NAME, params, &identity.ServicesV3, args.Service)
		} else {
			parameter, err = yunionconf.Parameters.Get(s, args.NAME, params)
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

		parameter, err := yunionconf.Parameters.Create(s, params)
		if err != nil {
			return err
		}
		printObject(parameter)
		return nil
	})

	type ParametersEditOptions struct {
		NamespaceId string `help:"List parameter of specificated namespace id, ADMIN only"`
		User        string `help:"Update parameter of specificated user id, ADMIN only"`
		Service     string `help:"Update parameter of specificated service id, ADMIN only"`
		NAME        string `help:"The name of parameter"`
	}

	type ParametersUpdateOptions struct {
		ParametersEditOptions
		VALUE string `help:"The content of parameter"`
	}

	updateFunc := func(s *mcclient.ClientSession, args *ParametersUpdateOptions) error {
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
			parameter, err = yunionconf.Parameters.PutInContext(s, args.NAME, params, &identity.UsersV3, args.User)
		} else if len(args.Service) > 0 {
			parameter, err = yunionconf.Parameters.PutInContext(s, args.NAME, params, &identity.ServicesV3, args.Service)
		} else {
			parameter, err = yunionconf.Parameters.Put(s, args.NAME, params)
		}

		if err != nil {
			return err
		}
		printObject(parameter)
		return nil
	}
	R(&ParametersUpdateOptions{}, "parameter-update", "update parameter", updateFunc)
	R(&ParametersEditOptions{}, "parameter-edit", "edit parameter", func(s *mcclient.ClientSession, args *ParametersEditOptions) error {
		params := jsonutils.NewDict()
		var parameter jsonutils.JSONObject
		var err error
		if len(args.User) > 0 {
			params.Add(jsonutils.NewString("system"), "scope")
			parameter, err = yunionconf.Parameters.GetInContext(s, args.NAME, params, &identity.UsersV3, args.User)
		} else if len(args.Service) > 0 {
			params.Add(jsonutils.NewString("system"), "scope")
			parameter, err = yunionconf.Parameters.GetInContext(s, args.NAME, params, &identity.ServicesV3, args.Service)
		} else {
			parameter, err = yunionconf.Parameters.Get(s, args.NAME, params)
		}
		if err != nil {
			return err
		}
		valueStr, err := parameter.GetString("value")
		if err != nil {
			return err
		}
		valueJson, err := jsonutils.ParseString(valueStr)
		if err != nil {
			return err
		}
		valueYaml := valueJson.YAMLString()
		valueYaml, err = shellutils.Edit(valueYaml)
		if err != nil {
			return err
		}
		valueJson, err = jsonutils.ParseYAML(valueYaml)
		if err != nil {
			return err
		}
		valueStr = valueJson.String()

		updateArgs := &ParametersUpdateOptions{
			ParametersEditOptions: *args,
			VALUE:                 valueStr,
		}
		return updateFunc(s, updateArgs)
	})

	type ParametersDeleteOptions struct {
		User    string `help:"Delete parameter of specificated user id, ADMIN only"`
		Service string `help:"Delete parameter of specificated service id, ADMIN only"`
		NAME    string `help:"The name of parameter"`
	}

	R(&ParametersDeleteOptions{}, "parameter-delete", "delete parameter", func(s *mcclient.ClientSession, args *ParametersDeleteOptions) error {
		params := jsonutils.NewDict()

		var parameter jsonutils.JSONObject
		var err error
		if len(args.User) > 0 {
			parameter, err = yunionconf.Parameters.DeleteInContext(s, args.NAME, params, &identity.UsersV3, args.User)
		} else if len(args.Service) > 0 {
			parameter, err = yunionconf.Parameters.DeleteInContext(s, args.NAME, params, &identity.ServicesV3, args.Service)
		} else {
			parameter, err = yunionconf.Parameters.Delete(s, args.NAME, nil)
		}

		if err != nil {
			return err
		}
		printObject(parameter)
		return nil
	})

	type ParameterCloneOptions struct {
		User        string `help:"Clone parameter of specificated user id"`
		Service     string `help:"Clone parameter of specificated service id"`
		NAME        string `help:"The name of parameter"`
		DestUser    string `help:"destination user id of clone action"`
		DestService string `help:"destination service id of clone action"`
		DestName    string `help:"destination parameter name of clone action, may be empty"`
	}
	R(&ParameterCloneOptions{}, "parameter-clone", "clone parameter", func(s *mcclient.ClientSession, args *ParameterCloneOptions) error {
		input := api.ParameterCloneInput{}

		input.DestName = args.DestName

		if len(args.DestUser) > 0 {
			input.DestNs = api.NAMESPACE_USER
			input.DestNsId = args.DestUser
		} else if len(args.DestService) > 0 {
			input.DestNs = api.NAMESPACE_SERVICE
			input.DestNsId = args.DestService
		} else {
			return errors.Wrap(httperrors.ErrMissingParameter, "either dest_user or dest_service must be specified")
		}
		params := jsonutils.Marshal(input).(*jsonutils.JSONDict)

		var parameter jsonutils.JSONObject
		var err error
		if len(args.User) > 0 {
			parameter, err = yunionconf.Parameters.PerformActionInContext(s, args.NAME, "clone", params, &identity.UsersV3, args.User)
		} else if len(args.Service) > 0 {
			parameter, err = yunionconf.Parameters.PerformActionInContext(s, args.NAME, "clone", params, &identity.ServicesV3, args.Service)
		} else {
			parameter, err = yunionconf.Parameters.PerformAction(s, args.NAME, "clone", params)
		}

		if err != nil {
			return err
		}
		printObject(parameter)
		return nil
	})

	type ParameterCloneDashboardOptions struct {
		SCOPE string `help:"dashboard scope" choices:"system|domain|project"`
		SRC   string `help:"source user id"`
		DST   string `help:"destination user id"`
	}
	R(&ParameterCloneDashboardOptions{}, "parameter-clone-dashboard", "clone dashboard parameter", func(s *mcclient.ClientSession, args *ParameterCloneDashboardOptions) error {
		cloneUserParams := func(srcUid, destUid, name string) (jsonutils.JSONObject, error) {
			paramId, err := yunionconf.Parameters.GetIdInContext(s, name, nil, &identity.UsersV3, srcUid)
			if err != nil {
				return nil, errors.Wrapf(err, "GetByName %s", name)
			}
			input := api.ParameterCloneInput{
				DestNs:   api.NAMESPACE_USER,
				DestNsId: destUid,
			}
			params := jsonutils.Marshal(input).(*jsonutils.JSONDict)
			return yunionconf.Parameters.PerformActionInContext(s, paramId, "clone", params, &identity.UsersV3, srcUid)
		}

		rootName := fmt.Sprintf("dashboard_%s", args.SCOPE)

		confs := []struct {
			Id   string
			Name string
		}{}

		paramObj, err := yunionconf.Parameters.GetInContext(s, rootName, nil, &identity.UsersV3, args.SRC)
		if err != nil {
			return errors.Wrap(err, "GetParameter")
		}

		err = paramObj.Unmarshal(&confs, "value")
		if err != nil {
			return errors.Wrap(err, "unmarshal value")
		}

		for _, conf := range confs {
			_, err := cloneUserParams(args.SRC, args.DST, conf.Id)
			if err != nil {
				return errors.Wrapf(err, "clone %s", conf.Id)
			}
			fmt.Println("cloned", conf.Id)
		}

		// finally copy the root
		_, err = cloneUserParams(args.SRC, args.DST, rootName)
		if err != nil {
			return errors.Wrapf(err, "clone %s", rootName)
		}
		fmt.Println("cloned", rootName)

		return nil
	})

}
