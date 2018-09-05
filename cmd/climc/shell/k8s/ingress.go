package k8s

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func initIngress() {
	cmdN := func(suffix string) string {
		return resourceCmdN("statefulset", suffix)
	}

	R(&NamespaceResourceListOptions{}, cmdN("list"), "List k8s ingress", func(s *mcclient.ClientSession, args *NamespaceResourceListOptions) error {
		ret, err := k8s.Ingresses.List(s, args.Params())
		if err != nil {
			return err
		}
		PrintListResultTable(ret, k8s.Ingresses, s)
		return nil
	})

	type getOpt struct {
		resourceGetOptions
	}
	R(&getOpt{}, cmdN("show"), "Get ingress details", func(s *mcclient.ClientSession, args *getOpt) error {
		id := args.NAME
		params := args.ClusterParams()
		if args.Namespace != "" {
			params.Add(jsonutils.NewString(args.Namespace), "namespace")
		}
		ret, err := k8s.Ingresses.Get(s, id, params)
		if err != nil {
			return err
		}
		printObjectYAML(ret)
		return nil
	})
}
