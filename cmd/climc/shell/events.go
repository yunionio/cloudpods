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
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

type BaseEventListOptions struct {
	Scope      string   `help:"scope" choices:"project|domain|system"`
	Since      string   `help:"Show logs since specific date" metavar:"DATETIME"`
	Until      string   `help:"Show logs until specific date" metavar:"DATETIME"`
	Limit      int64    `help:"Limit number of logs" default:"20"`
	Offset     int64    `help:"Offset"`
	Ascending  bool     `help:"Ascending order"`
	Descending bool     `help:"Descending order"`
	OrderBy    string   `help:"order by specific field"`
	Action     []string `help:"Log action"`

	User    []string `help:"filter by operator user"`
	Project []string `help:"filter by owner project"`

	PagingMarker string `help:"marker for pagination"`
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

func doImageEventList(s *mcclient.ClientSession, args *EventListOptions) error {
	return doEventList(modules.ImageLogs, s, args)
}

func doIdentityEventList(s *mcclient.ClientSession, args *EventListOptions) error {
	return doEventList(modules.IdentityLogs, s, args)
}

func doEventList(man modulebase.ResourceManager, s *mcclient.ClientSession, args *EventListOptions) error {
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
	if len(args.User) > 0 {
		params.Add(jsonutils.NewStringArray(args.User), "user")
	}
	if len(args.Project) > 0 {
		params.Add(jsonutils.NewStringArray(args.Project), "project")
	}
	if len(args.Scope) > 0 {
		params.Add(jsonutils.NewString(args.Scope), "scope")
	}
	if len(args.PagingMarker) > 0 {
		params.Add(jsonutils.NewString(args.PagingMarker), "paging_marker")
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

	R(&TypeEventListOptions{}, "disk-event", "Show operation event logs of disk", func(s *mcclient.ClientSession, args *TypeEventListOptions) error {
		nargs := EventListOptions{BaseEventListOptions: args.BaseEventListOptions, Id: args.ID, Type: []string{"disk"}}
		return doComputeEventList(s, &nargs)
	})

	R(&TypeEventListOptions{}, "eip-event", "Show operation event logs of elastic IP", func(s *mcclient.ClientSession, args *TypeEventListOptions) error {
		nargs := EventListOptions{BaseEventListOptions: args.BaseEventListOptions, Id: args.ID, Type: []string{"eip"}}
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

	R(&TypeEventListOptions{}, "cloud-provider-event", "Show operation event logs of cloudprovider", func(s *mcclient.ClientSession, args *TypeEventListOptions) error {
		nargs := EventListOptions{BaseEventListOptions: args.BaseEventListOptions, Id: args.ID, Type: []string{"cloudprovider"}}
		return doComputeEventList(s, &nargs)
	})

	R(&TypeEventListOptions{}, "cloud-account-event", "Show operation event logs of cloudaccount", func(s *mcclient.ClientSession, args *TypeEventListOptions) error {
		nargs := EventListOptions{BaseEventListOptions: args.BaseEventListOptions, Id: args.ID, Type: []string{"cloudaccount"}}
		return doComputeEventList(s, &nargs)
	})

	R(&TypeEventListOptions{}, "bucket-event", "Show operation event logs of bucket", func(s *mcclient.ClientSession, args *TypeEventListOptions) error {
		nargs := EventListOptions{BaseEventListOptions: args.BaseEventListOptions, Id: args.ID, Type: []string{"bucket"}}
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

	R(&TypeEventListOptions{}, "image-event", "Show operation event logs of glance images", func(s *mcclient.ClientSession, args *TypeEventListOptions) error {
		nargs := EventListOptions{BaseEventListOptions: args.BaseEventListOptions, Id: args.ID, Type: []string{"image"}}
		return doImageEventList(s, &nargs)
	})

	R(&TypeEventListOptions{}, "user-event", "Show operation event logs of keystone users", func(s *mcclient.ClientSession, args *TypeEventListOptions) error {
		nargs := EventListOptions{BaseEventListOptions: args.BaseEventListOptions, Id: args.ID, Type: []string{"user"}}
		return doIdentityEventList(s, &nargs)
	})

	R(&TypeEventListOptions{}, "group-event", "Show operation event logs of keystone groups", func(s *mcclient.ClientSession, args *TypeEventListOptions) error {
		nargs := EventListOptions{BaseEventListOptions: args.BaseEventListOptions, Id: args.ID, Type: []string{"group"}}
		return doIdentityEventList(s, &nargs)
	})

	R(&TypeEventListOptions{}, "domain-event", "Show operation event logs of keystone domains", func(s *mcclient.ClientSession, args *TypeEventListOptions) error {
		nargs := EventListOptions{BaseEventListOptions: args.BaseEventListOptions, Id: args.ID, Type: []string{"domain"}}
		return doIdentityEventList(s, &nargs)
	})

	R(&TypeEventListOptions{}, "project-event", "Show operation event logs of keystone projects", func(s *mcclient.ClientSession, args *TypeEventListOptions) error {
		nargs := EventListOptions{BaseEventListOptions: args.BaseEventListOptions, Id: args.ID, Type: []string{"project"}}
		return doIdentityEventList(s, &nargs)
	})

	R(&TypeEventListOptions{}, "role-event", "Show operation event logs of keystone roles", func(s *mcclient.ClientSession, args *TypeEventListOptions) error {
		nargs := EventListOptions{BaseEventListOptions: args.BaseEventListOptions, Id: args.ID, Type: []string{"role"}}
		return doIdentityEventList(s, &nargs)
	})

	R(&TypeEventListOptions{}, "policy-event", "Show operation event logs of keystone policies", func(s *mcclient.ClientSession, args *TypeEventListOptions) error {
		nargs := EventListOptions{BaseEventListOptions: args.BaseEventListOptions, Id: args.ID, Type: []string{"policy"}}
		return doIdentityEventList(s, &nargs)
	})

	R(&TypeEventListOptions{}, "endpoint-event", "Show operation event logs of keystone endpoints", func(s *mcclient.ClientSession, args *TypeEventListOptions) error {
		nargs := EventListOptions{BaseEventListOptions: args.BaseEventListOptions, Id: args.ID, Type: []string{"endpoint"}}
		return doIdentityEventList(s, &nargs)
	})

	R(&TypeEventListOptions{}, "service-event", "Show operation event logs of keystone services", func(s *mcclient.ClientSession, args *TypeEventListOptions) error {
		nargs := EventListOptions{BaseEventListOptions: args.BaseEventListOptions, Id: args.ID, Type: []string{"service"}}
		return doIdentityEventList(s, &nargs)
	})

	R(&TypeEventListOptions{}, "credential-event", "Show operation event logs of keystone credentials", func(s *mcclient.ClientSession, args *TypeEventListOptions) error {
		nargs := EventListOptions{BaseEventListOptions: args.BaseEventListOptions, Id: args.ID, Type: []string{"credential"}}
		return doIdentityEventList(s, &nargs)
	})
}
