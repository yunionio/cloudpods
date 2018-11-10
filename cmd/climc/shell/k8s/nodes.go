package k8s

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
)

func initNode() {
	cmdN := func(suffix string) string {
		return kubeResourceCmdN("node", suffix)
	}
	R(&o.NodeListOptions{}, cmdN("list"), "List k8s infra nodes", func(s *mcclient.ClientSession, args *o.NodeListOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		result, err := k8s.Nodes.List(s, params)
		if err != nil {
			return err
		}
		printList(result, k8s.Nodes.GetColumns(s))
		return nil
	})

	R(&o.NodeCreateOptions{}, cmdN("create"), "Create k8s cluster node", func(s *mcclient.ClientSession, args *o.NodeCreateOptions) error {
		params := args.Params()
		node, err := k8s.Nodes.Create(s, params)
		if err != nil {
			return err
		}
		printObject(node)
		return nil
	})

	R(&o.IdentOptions{}, cmdN("delete"), "Delete node", func(s *mcclient.ClientSession, args *o.IdentOptions) error {
		ret, err := k8s.Nodes.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	R(&o.IdentOptions{}, cmdN("show"), "Show node details", func(s *mcclient.ClientSession, args *o.IdentOptions) error {
		obj, err := k8s.Nodes.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(obj)
		return nil
	})

	R(&o.IdentsOptions{}, cmdN("purge"), "Purge a node record in database, not actually do deletion", func(s *mcclient.ClientSession, args *o.IdentsOptions) error {
		ret := k8s.Nodes.BatchPerformAction(s, args.ID, "purge", nil)
		printBatchResults(ret, k8s.Nodes.GetColumns(s))
		return nil
	})

	R(&o.IdentOptions{}, cmdN("dockerconfig"), "Get docker daemon config", func(s *mcclient.ClientSession, args *o.IdentOptions) error {
		ret, err := k8s.Nodes.GetSpecific(s, args.ID, "docker-config", nil)
		if err != nil {
			return err
		}
		fmt.Println(ret)
		return nil
	})

	R(&o.NodeConfigDockerRegistryOptions{}, cmdN("config-docker-registry"), "Config node docker daemon registry",
		func(s *mcclient.ClientSession, args *o.NodeConfigDockerRegistryOptions) error {
			params := args.Params()
			ret := k8s.Nodes.BatchPerformAction(s, args.ID, "config-docker-registry", params)
			printBatchResults(ret, k8s.Nodes.GetColumns(s))
			return nil
		})
}
