package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type ParametersListOptions struct {
		options.BaseListOptions
		NamespaceId string `help:"Show parameter of specificated namespace id, ADMIN only"`
	}

	R(&ParametersListOptions{}, "parameter-list", "list parameters", func(s *mcclient.ClientSession, args *ParametersListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}

		if len(args.NamespaceId) > 0 {
			params.Add(jsonutils.JSONTrue, "admin")
			params.Add(jsonutils.NewString(args.NamespaceId), "namespace_id")
		}

		result, err := modules.Parameters.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Parameters.GetColumns(s))
		return nil
	})

	type ParametersShowOptions struct {
		Namespace   string `help:"Show parameter of specificated namespace, ADMIN only"`
		NamespaceId string `help:"Show parameter of specificated namespace id, ADMIN only"`
		UserId      string `help:"Show parameter created by specificated user, ADMIN only"`
		NAME        string `help:"The name of parameter"`
	}

	R(&ParametersShowOptions{}, "parameter-show", "show a parameter", func(s *mcclient.ClientSession, args *ParametersShowOptions) error {
		params := jsonutils.NewDict()
		if len(args.Namespace) > 0 {
			params.Add(jsonutils.JSONTrue, "admin")
			params.Add(jsonutils.NewString(args.Namespace), "namespace")
		}

		if len(args.NamespaceId) > 0 {
			params.Add(jsonutils.JSONTrue, "admin")
			params.Add(jsonutils.NewString(args.NamespaceId), "namespace_id")
		}

		if len(args.UserId) > 0 {
			params.Add(jsonutils.JSONTrue, "admin")
			params.Add(jsonutils.NewString(args.UserId), "created_by")
		}
		parameter, err := modules.Parameters.Get(s, args.NAME, params)
		if err != nil {
			return err
		}
		printObject(parameter)
		return nil
	})

	type ParametersCreateOptions struct {
		User    string `help:"Create parameter for specificated user, ADMIN only"`
		Service string `help:"Create parameter for specificated service, ADMIN only"`
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
		User    string `help:"Update parameter for specificated user, ADMIN only"`
		Service string `help:"Update parameter for specificated service, ADMIN only"`
		NAME    string `help:"The name of parameter"`
		VALUE   string `help:"The content of parameter"`
	}

	R(&ParametersUpdateOptions{}, "parameter-update", "update parameter", func(s *mcclient.ClientSession, args *ParametersUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.VALUE) > 0 {
			params.Add(jsonutils.NewString(args.VALUE), "value")
		}

		var ctx modules.Manager
		var ctxid string
		if len(args.User) > 0 {
			ctxid = args.User
			ctx = &modules.UsersV3
			params.Add(jsonutils.JSONTrue, "admin")
			params.Add(jsonutils.NewString(args.User), "user_id")

		} else if len(args.Service) > 0 {
			ctxid = args.Service
			ctx = &modules.ServicesV3
			params.Add(jsonutils.JSONTrue, "admin")
			params.Add(jsonutils.NewString(args.Service), "service_id")
		} else {
			ctxid = s.GetUserId()
			ctx = &modules.UsersV3
		}

		parameter, err := modules.Parameters.PutInContext(s, args.NAME, params, ctx, ctxid)
		if err != nil {
			return err
		}
		printObject(parameter)
		return nil
	})

	type ParametersDeleteOptions struct {
		User    string `help:"Delete parameter for specificated user, ADMIN only"`
		Service string `help:"Delete parameter for specificated service, ADMIN only"`
		NAME    string `help:"The name of parameter"`
	}

	R(&ParametersDeleteOptions{}, "parameter-delete", "delete notice", func(s *mcclient.ClientSession, args *ParametersDeleteOptions) error {
		params := jsonutils.NewDict()

		var ctx modules.Manager
		var ctxid string
		if len(args.User) > 0 {
			ctxid = args.User
			ctx = &modules.UsersV3
			params.Add(jsonutils.JSONTrue, "admin")
			params.Add(jsonutils.NewString(args.User), "user_id")
		} else if len(args.Service) > 0 {
			ctxid = args.Service
			ctx = &modules.ServicesV3
			params.Add(jsonutils.JSONTrue, "admin")
			params.Add(jsonutils.NewString(args.Service), "service_id")
		} else {
			ctxid = s.GetUserId()
			ctx = &modules.UsersV3
		}

		parameter, err := modules.Parameters.DeleteInContext(s, args.NAME, params, ctx, ctxid)
		if err != nil {
			return err
		}
		printObject(parameter)
		return nil
	})
}
