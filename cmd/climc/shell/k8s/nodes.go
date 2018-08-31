package k8s

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func initNode() {
	cmdN := func(suffix string) string {
		return resourceCmdN("node", suffix)
	}
	type listOpt struct {
		options.BaseListOptions
		Cluster string `help:"Filter by cluster"`
	}
	R(&listOpt{}, cmdN("list"), "List k8s node", func(s *mcclient.ClientSession, args *listOpt) error {
		args.Details = options.Bool(true)
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = options.ListStructToParams(args)
			if err != nil {
				return err

			}
		}
		result, err := k8s.Nodes.List(s, params)
		if err != nil {
			return err
		}
		printList(result, k8s.Nodes.GetColumns(s))
		return nil
	})

	type createOpt struct {
		CLUSTER          string   `help:"Cluster id"`
		Etcd             bool     `help:"Etcd role"`
		Controlplane     bool     `help:"Controlplane role"`
		Worker           bool     `help:"Worker role"`
		AllRole          bool     `help:"All roles"`
		HostnameOverride string   `help:"Worker node overrided hostname"`
		Host             string   `help:"Yunion host server name or id"`
		Name             string   `help:"Name of node"`
		RegistryMirror   []string `help:"Docker registry mirrors, e.g. 'https://registry.docker-cn.com'"`
		InsecureRegistry []string `help:"Docker insecure registry"`
	}
	R(&createOpt{}, cmdN("create"), "Create k8s cluster node", func(s *mcclient.ClientSession, args *createOpt) error {
		params := jsonutils.NewDict()
		if args.Name != "" {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		params.Add(jsonutils.NewString(args.CLUSTER), "cluster")

		dockerConf := dockerConfig{}
		for _, rm := range args.RegistryMirror {
			dockerConf.RegistryMirrors = append(dockerConf.RegistryMirrors, rm)
		}
		for _, im := range args.InsecureRegistry {
			dockerConf.InsecureRegistries = append(dockerConf.InsecureRegistries, im)
		}
		confObj := jsonutils.Marshal(dockerConf)
		params.Add(confObj, "dockerd_config")

		roles := jsonutils.NewArray()
		if args.AllRole {
			roles.Add(jsonutils.NewString("etcd"), jsonutils.NewString("controlplane"), jsonutils.NewString("worker"))
		} else {
			if args.Etcd {
				roles.Add(jsonutils.NewString("etcd"))
			}
			if args.Controlplane {
				roles.Add(jsonutils.NewString("controlplane"))
			}
			if args.Worker {
				roles.Add(jsonutils.NewString("worker"))
			}
		}
		params.Add(roles, "roles")
		if args.HostnameOverride != "" {
			params.Add(jsonutils.NewString(args.HostnameOverride), "hostname_override")
		}
		if args.Host != "" {
			params.Add(jsonutils.NewString(args.Host), "host")
		}
		node, err := k8s.Nodes.Create(s, params)
		if err != nil {
			return err
		}
		printObject(node)
		return nil
	})

	type identOpt struct {
		ID string `help:"ID or name of the node"`
	}

	type identsOpt struct {
		ID []string `help:"ID or name of the nodes"`
	}
	type deleteOpt struct {
		identsOpt
	}
	R(&deleteOpt{}, cmdN("delete"), "Delete node", func(s *mcclient.ClientSession, args *deleteOpt) error {
		ret := k8s.Nodes.BatchDeleteWithParam(s, args.ID, nil, nil)
		printBatchResults(ret, k8s.Nodes.GetColumns(s))
		return nil
	})

	R(&identOpt{}, cmdN("show"), "Show node details", func(s *mcclient.ClientSession, args *identOpt) error {
		obj, err := k8s.Nodes.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(obj)
		return nil
	})

	R(&identsOpt{}, cmdN("purge"), "Purge a node record in database, not actually do deletion", func(s *mcclient.ClientSession, args *identsOpt) error {
		ret := k8s.Nodes.BatchPerformAction(s, args.ID, "purge", nil)
		printBatchResults(ret, k8s.Nodes.GetColumns(s))
		return nil
	})

	R(&identOpt{}, cmdN("dockerconfig"), "Get docker daemon config", func(s *mcclient.ClientSession, args *identOpt) error {
		ret, err := k8s.Nodes.GetSpecific(s, args.ID, "docker-config", nil)
		if err != nil {
			return err
		}
		fmt.Println(ret)
		return nil
	})
}
