package shell

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type BaseActionListOptions struct {
	Since      string   `help:"Show logs since specific date" metavar:"DATETIME"`
	Until      string   `help:"Show logs until specific date" metavar:"DATETIME"`
	Limit      int64    `help:"Limit number of logs" default:"20"`
	Offset     int64    `help:"Offset"`
	Ascending  bool     `help:"Ascending order"`
	Descending bool     `help:"Descending order"`
	Action     []string `help:"Log action"`
	Search     string   `help:"Filter action logs by obj_name, using 'like' syntax."`
}

type ActionListOptions struct {
	BaseActionListOptions
	Id   string   `help:"" metavar:"OBJ_ID"`
	Type []string `help:"Type of relevant object" metavar:"OBJ_TYPE"`
}

type TypeActionListOptions struct {
	BaseActionListOptions
	ID string `help:"" metavar:"OBJ_ID"`
}

func doActionList(s *mcclient.ClientSession, args *ActionListOptions) error {
	params := jsonutils.NewDict()
	if len(args.Type) > 0 {
		params.Add(jsonutils.NewStringArray(args.Type), "obj_type")
	}
	if len(args.Id) > 0 {
		params.Add(jsonutils.NewString(args.Id), "obj_id")
	}
	if len(args.Search) > 0 {
		params.Add(jsonutils.NewString(args.Search), "search")
	}
	if len(args.Since) > 0 {
		params.Add(jsonutils.NewString(args.Since), "since")
	}
	if len(args.Until) > 0 {
		params.Add(jsonutils.NewString(args.Until), "until")
	}
	if args.Limit > 0 {
		params.Add(jsonutils.NewInt(args.Limit), "limit")
	}
	if args.Offset > 0 {
		params.Add(jsonutils.NewInt(args.Offset), "offset")
	}
	if args.Ascending && !args.Descending {
		params.Add(jsonutils.NewString("asc"), "order")
	} else if !args.Ascending && args.Descending {
		params.Add(jsonutils.NewString("desc"), "order")
	}
	if len(args.Action) > 0 {
		params.Add(jsonutils.NewStringArray(args.Action), "action")
	}
	logs, err := modules.Actions.List(s, params)
	if err != nil {
		return err
	}
	printList(logs, modules.Actions.GetColumns(s))
	return nil
}

func init() {
	R(&ActionListOptions{}, "action-show", "Show operation action logs", doActionList)

	R(&TypeActionListOptions{}, "server-action", "Show operation action logs of server", func(s *mcclient.ClientSession, args *TypeActionListOptions) error {
		nargs := ActionListOptions{BaseActionListOptions: args.BaseActionListOptions, Id: args.ID, Type: []string{"server"}}
		return doActionList(s, &nargs)
	})

	R(&TypeActionListOptions{}, "host-action", "Show operation action logs of host", func(s *mcclient.ClientSession, args *TypeActionListOptions) error {
		nargs := ActionListOptions{BaseActionListOptions: args.BaseActionListOptions, Id: args.ID, Type: []string{"host"}}
		return doActionList(s, &nargs)
	})

	R(&TypeActionListOptions{}, "vcenter-action", "Show operation action logs of vcenter", func(s *mcclient.ClientSession, args *TypeActionListOptions) error {
		nargs := ActionListOptions{BaseActionListOptions: args.BaseActionListOptions, Id: args.ID, Type: []string{"vcenter"}}
		return doActionList(s, &nargs)
	})
}
