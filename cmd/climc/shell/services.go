package shell

import (
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/mcclient"
	"github.com/yunionio/onecloud/pkg/mcclient/modules"
)

func init() {
	type ServiceListOptions struct {
		Limit  int64  `help:"Limit, default 0, i.e. no limit" default:"20"`
		Offset int64  `help:"Offset, default 0, i.e. no offset"`
		Name   string `help:"Search by name"`
		Type   string `help:"Search by type"`
	}
	R(&ServiceListOptions{}, "service-list", "List services", func(s *mcclient.ClientSession, args *ServiceListOptions) error {
		mod, err := modules.GetModule(s, "services")
		if err != nil {
			return err
		}
		query := jsonutils.NewDict()
		if args.Limit > 0 {
			query.Add(jsonutils.NewInt(args.Limit), "limit")
		}
		if args.Offset > 0 {
			query.Add(jsonutils.NewInt(args.Offset), "offset")
		}
		if len(args.Name) > 0 {
			query.Add(jsonutils.NewString(args.Name), "name__icontains")
		}
		if len(args.Type) > 0 {
			query.Add(jsonutils.NewString(args.Type), "type__icontains")
		}
		result, err := mod.List(s, query)
		if err != nil {
			return err
		}
		printList(result, mod.GetColumns(s))
		return nil
	})

	type ServiceShowOptions struct {
		ID string `help:"ID of service"`
	}
	R(&ServiceShowOptions{}, "service-show", "Show details of a service", func(s *mcclient.ClientSession, args *ServiceShowOptions) error {
		mod, err := modules.GetModule(s, "services")
		if err != nil {
			return err
		}
		srvId, err := mod.GetId(s, args.ID, nil)
		if err != nil {
			return err
		}
		result, err := mod.Get(s, srvId, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
	R(&ServiceShowOptions{}, "service-delete", "Delete a service", func(s *mcclient.ClientSession, args *ServiceShowOptions) error {
		mod, err := modules.GetModule(s, "services")
		if err != nil {
			return err
		}
		srvId, err := mod.GetId(s, args.ID, nil)
		if err != nil {
			return err
		}
		result, err := mod.Delete(s, srvId, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServiceCreateOptions struct {
		TYPE     string `help:"Service type"`
		NAME     string `help:"Service name"`
		Desc     string `help:"Description"`
		Enabled  bool   `help:"Enabeld"`
		Disabled bool   `help:"Disabled"`
	}
	R(&ServiceCreateOptions{}, "service-create", "Create a service", func(s *mcclient.ClientSession, args *ServiceCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.TYPE), "type")
		params.Add(jsonutils.NewString(args.NAME), "name")
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if args.Enabled && !args.Disabled {
			params.Add(jsonutils.JSONTrue, "enabled")
		} else if !args.Enabled && args.Disabled {
			params.Add(jsonutils.JSONFalse, "enabled")
		}
		mod, err := modules.GetModule(s, "services")
		if err != nil {
			return err
		}
		srv, err := mod.Create(s, params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

	type ServiceUpdateOptions struct {
		ID       string `help:"ID or name of the service"`
		Type     string `help:"Service type"`
		Name     string `help:"Service name"`
		Desc     string `help:"Description"`
		Enabled  bool   `help:"Enabeld"`
		Disabled bool   `help:"Disabled"`
	}
	R(&ServiceUpdateOptions{}, "service-update", "Update a service", func(s *mcclient.ClientSession, args *ServiceUpdateOptions) error {
		mod, err := modules.GetModule(s, "services")
		if err != nil {
			return err
		}
		srvId, err := mod.GetId(s, args.ID, nil)
		if err != nil {
			return err
		}
		params := jsonutils.NewDict()
		if len(args.Type) > 0 {
			params.Add(jsonutils.NewString(args.Type), "type")
		}
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if args.Enabled && !args.Disabled {
			params.Add(jsonutils.JSONTrue, "enabled")
		} else if !args.Enabled && args.Disabled {
			params.Add(jsonutils.JSONFalse, "enabled")
		}
		srv, err := mod.Patch(s, srvId, params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})
}
