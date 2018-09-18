package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type ParametersListOptions struct {
		NamespaceId string `help:"List parameter of specificated namespace id, ADMIN only"`
		User        string `help:"List parameter of specificated user id, ADMIN only"`
		Service     string `help:"List parameter of specificated service id, ADMIN only"`
		options.BaseListOptions
	}

	R(&ParametersListOptions{}, "parameter-list", "list parameters", func(s *mcclient.ClientSession, args *ParametersListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}

		var result *modules.ListResult
		if len(args.NamespaceId) > 0 {
			params.Add(jsonutils.NewString(args.NamespaceId), "namespace_id")
			result, err = modules.Parameters.List(s, params)
		} else if len(args.User) > 0 {
			result, err = modules.Parameters.ListInContext(s, params, &modules.UsersV3, args.User)
		} else if len(args.Service) > 0 {
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
		if len(args.NamespaceId) > 0 {
			params.Add(jsonutils.JSONTrue, "admin")
			params.Add(jsonutils.NewString(args.NamespaceId), "namespace_id")
		}

		var parameter jsonutils.JSONObject
		var err error
		if len(args.NamespaceId) > 0 {
			params.Add(jsonutils.NewString(args.NamespaceId), "namespace_id")
			parameter, err = modules.Parameters.Get(s, args.NAME, params)
		} else if len(args.User) > 0 {
			parameter, err = modules.Parameters.GetInContext(s, args.NAME, params, &modules.UsersV3, args.User)
		} else if len(args.Service) > 0 {
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
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString(args.VALUE), "value")

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
		params := jsonutils.NewDict()
		if len(args.VALUE) > 0 {
			params.Add(jsonutils.NewString(args.VALUE), "value")
		}

		var parameter jsonutils.JSONObject
		var err error
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
