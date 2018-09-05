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
	}

	R(&ParametersListOptions{}, "parameter-list", "list parameters", func(s *mcclient.ClientSession, args *ParametersListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}

		result, err := modules.Parameters.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Parameters.GetColumns(s))
		return nil
	})

	type ParametersShowOptions struct {
		Admin  bool   `help:"Show parameter of all users, ADMIN only"`
		UserId string `help:"Show parameter of a user, ADMIN only"`
		NAME   string `help:"The name of parameter"`
	}

	R(&ParametersShowOptions{}, "parameter-show", "show a parameter", func(s *mcclient.ClientSession, args *ParametersShowOptions) error {
		params := jsonutils.NewDict()
		if args.Admin {
			params.Add(jsonutils.JSONTrue, "admin")
		}

		if len(args.UserId) > 0 {
			params.Add(jsonutils.NewString(args.UserId), "user_id")
		}
		parameter, err := modules.Parameters.Get(s, args.NAME, params)
		if err != nil {
			return err
		}
		printObject(parameter)
		return nil
	})

	type ParametersCreateOptions struct {
		NAME  string `help:"The name of parameter"`
		VALUE string `help:"The content of parameter"`
	}

	R(&ParametersCreateOptions{}, "parameter-create", "create a parameter", func(s *mcclient.ClientSession, args *ParametersCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString(args.VALUE), "value")

		parameter, err := modules.Parameters.Create(s, params)
		if err != nil {
			return err
		}
		printObject(parameter)
		return nil
	})

	type ParametersUpdateOptions struct {
		Admin  bool   `help:"Update parameter of all users, ADMIN only"`
		UserId string `help:"Update parameter of a user, ADMIN only"`
		NAME   string `help:"The name of parameter"`
		VALUE  string `help:"The content of parameter"`
	}

	R(&ParametersUpdateOptions{}, "parameter-update", "update parameter", func(s *mcclient.ClientSession, args *ParametersUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.VALUE) > 0 {
			params.Add(jsonutils.NewString(args.VALUE), "value")
		}

		if args.Admin {
			params.Add(jsonutils.JSONTrue, "admin")
		}

		if len(args.UserId) > 0 {
			params.Add(jsonutils.NewString(args.UserId), "user_id")
		}

		parameter, err := modules.Parameters.Update(s, args.NAME, params)
		if err != nil {
			return err
		}
		printObject(parameter)
		return nil
	})

	type ParametersDeleteOptions struct {
		Admin  bool   `help:"delete parameter of a user, ADMIN only"`
		UserId string `help:"delete parameter of a user, ADMIN only"`
		NAME   string `help:"The name of parameter"`
	}

	R(&ParametersDeleteOptions{}, "parameter-delete", "delete notice", func(s *mcclient.ClientSession, args *ParametersDeleteOptions) error {
		params := jsonutils.NewDict()
		if args.Admin {
			params.Add(jsonutils.JSONTrue, "admin")
		}

		if len(args.UserId) > 0 {
			params.Add(jsonutils.NewString(args.UserId), "user_id")
		}

		parameter, err := modules.Notice.Delete(s, args.NAME, params)
		if err != nil {
			return err
		}
		printObject(parameter)
		return nil
	})
}
