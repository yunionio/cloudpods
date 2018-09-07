package k8s

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func initPod() {
	cmdN := func(suffix string) string {
		return resourceCmdN("pod", suffix)
	}

	R(&NamespaceResourceListOptions{}, cmdN("list"), "List k8s pod", func(s *mcclient.ClientSession, args *NamespaceResourceListOptions) error {
		ret, err := k8s.Pods.List(s, args.Params())
		if err != nil {
			return err
		}
		PrintListResultTable(ret, k8s.Pods, s)
		return nil
	})

	type getOpt struct {
		resourceGetOptions
	}
	R(&getOpt{}, cmdN("show"), "Get pod details", func(s *mcclient.ClientSession, args *getOpt) error {
		id := args.NAME
		params := args.ClusterParams()
		if args.Namespace != "" {
			params.Add(jsonutils.NewString(args.Namespace), "namespace")
		}
		ret, err := k8s.Pods.Get(s, id, params)
		if err != nil {
			return err
		}
		printObjectYAML(ret)
		return nil
	})

	type deleteOpt struct {
		resourceGetOptions
	}
	R(&deleteOpt{}, cmdN("delete"), "Delete pod", func(s *mcclient.ClientSession, args *deleteOpt) error {
		id := args.NAME
		ret, err := k8s.Pods.Delete(s, id, args.ToJSON())
		if err != nil {
			return err
		}
		fmt.Println(ret)
		return nil
	})
}
