package shell

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

type BaseEventListOptions struct {
	Since      string   `help:"Show logs since specific date" metavar:"DATETIME"`
	Until      string   `help:"Show logs until specific date" metavar:"DATETIME"`
	Limit      int64    `help:"Limit number of logs" default:"20"`
	Offset     int64    `help:"Offset"`
	Ascending  bool     `help:"Ascending order"`
	Descending bool     `help:"Descending order"`
	OrderBy    string   `help:"order by specific field"`
	Action     []string `help:"Log action"`
}

type EventListOptions struct {
	BaseEventListOptions
	Id   string   `help:"" metavar:"OBJ_ID"`
	Type []string `help:"Type of relevant object" metavar:"OBJ_TYPE"`
}

type TypeEventListOptions struct {
	BaseEventListOptions
	ID string `help:"" metavar:"OBJ_ID"`
}

func doK8sEventList(s *mcclient.ClientSession, args *EventListOptions) error {
	return doEventList(*k8s.Logs.ResourceManager, s, args)
}

func doComputeEventList(s *mcclient.ClientSession, args *EventListOptions) error {
	return doEventList(modules.Logs, s, args)
}

func doEventList(man modules.ResourceManager, s *mcclient.ClientSession, args *EventListOptions) error {
	params := jsonutils.NewDict()
	if len(args.Type) > 0 {
		params.Add(jsonutils.NewStringArray(args.Type), "obj_type")
	}
	if len(args.Id) > 0 {
		params.Add(jsonutils.NewString(args.Id), "obj_id")
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
	if len(args.OrderBy) > 0 {
		params.Add(jsonutils.NewString(args.OrderBy), "order_by")
	}
	if len(args.Action) > 0 {
		params.Add(jsonutils.NewStringArray(args.Action), "action")
	}
	logs, err := man.List(s, params)
	if err != nil {
		return err
	}
	printList(logs, man.GetColumns(s))
	return nil
}

func init() {
	R(&EventListOptions{}, "event-show", "Show operation event logs", doComputeEventList)

	R(&TypeEventListOptions{}, "server-event", "Show operation event logs of server", func(s *mcclient.ClientSession, args *TypeEventListOptions) error {
		nargs := EventListOptions{BaseEventListOptions: args.BaseEventListOptions, Id: args.ID, Type: []string{"server"}}
		return doComputeEventList(s, &nargs)
	})

	R(&TypeEventListOptions{}, "host-event", "Show operation event logs of host", func(s *mcclient.ClientSession, args *TypeEventListOptions) error {
		nargs := EventListOptions{BaseEventListOptions: args.BaseEventListOptions, Id: args.ID, Type: []string{"host"}}
		return doComputeEventList(s, &nargs)
	})

	R(&TypeEventListOptions{}, "vpc-event", "Show operation event logs of vpc", func(s *mcclient.ClientSession, args *TypeEventListOptions) error {
		nargs := EventListOptions{BaseEventListOptions: args.BaseEventListOptions, Id: args.ID, Type: []string{"vpc"}}
		return doComputeEventList(s, &nargs)
	})

	R(&TypeEventListOptions{}, "zone-event", "Show operation event logs of zone", func(s *mcclient.ClientSession, args *TypeEventListOptions) error {
		nargs := EventListOptions{BaseEventListOptions: args.BaseEventListOptions, Id: args.ID, Type: []string{"zone"}}
		return doComputeEventList(s, &nargs)
	})

	R(&TypeEventListOptions{}, "region-event", "Show operation event logs of region", func(s *mcclient.ClientSession, args *TypeEventListOptions) error {
		nargs := EventListOptions{BaseEventListOptions: args.BaseEventListOptions, Id: args.ID, Type: []string{"cloudregion"}}
		return doComputeEventList(s, &nargs)
	})

	R(&TypeEventListOptions{}, "wire-event", "Show operation event logs of wire", func(s *mcclient.ClientSession, args *TypeEventListOptions) error {
		nargs := EventListOptions{BaseEventListOptions: args.BaseEventListOptions, Id: args.ID, Type: []string{"wire"}}
		return doComputeEventList(s, &nargs)
	})

	R(&TypeEventListOptions{}, "network-event", "Show operation event logs of network", func(s *mcclient.ClientSession, args *TypeEventListOptions) error {
		nargs := EventListOptions{BaseEventListOptions: args.BaseEventListOptions, Id: args.ID, Type: []string{"network"}}
		return doComputeEventList(s, &nargs)
	})

	R(&TypeEventListOptions{}, "vcenter-event", "Show operation event logs of vcenter", func(s *mcclient.ClientSession, args *TypeEventListOptions) error {
		nargs := EventListOptions{BaseEventListOptions: args.BaseEventListOptions, Id: args.ID, Type: []string{"vcenter"}}
		return doComputeEventList(s, &nargs)
	})

	R(&TypeEventListOptions{}, "kube-cluster-event", "Show operation event logs of kubernetes cluster", func(s *mcclient.ClientSession, args *TypeEventListOptions) error {
		nargs := EventListOptions{BaseEventListOptions: args.BaseEventListOptions, Id: args.ID, Type: []string{"kube_cluster"}}
		return doK8sEventList(s, &nargs)
	})

	R(&TypeEventListOptions{}, "kubecluster-event", "Show operation event logs of kubernetes cluster", func(s *mcclient.ClientSession, args *TypeEventListOptions) error {
		nargs := EventListOptions{BaseEventListOptions: args.BaseEventListOptions, Id: args.ID, Type: []string{"kubecluster"}}
		return doK8sEventList(s, &nargs)
	})

	R(&TypeEventListOptions{}, "kubemachine-event", "Show operation event logs of kubernetes machine", func(s *mcclient.ClientSession, args *TypeEventListOptions) error {
		nargs := EventListOptions{BaseEventListOptions: args.BaseEventListOptions, Id: args.ID, Type: []string{"kubemachine"}}
		return doK8sEventList(s, &nargs)
	})
}
