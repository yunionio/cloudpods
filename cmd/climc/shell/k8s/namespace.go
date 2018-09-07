package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func initNamespace() {
	cmdN := func(suffix string) string {
		return resourceCmdN("namespace", suffix)
	}

	type listOpt struct {
		clusterBaseOptions
		baseListOptions
	}
	R(&listOpt{}, cmdN("list"), "List k8s namespace", func(s *mcclient.ClientSession, args *listOpt) error {
		params := fetchPagingParams(args.baseListOptions)
		params.Update(args.ClusterParams())
		ret, err := k8s.Namespaces.List(s, params)
		if err != nil {
			return err
		}
		PrintListResultTable(ret, k8s.Namespaces, s)
		return nil
	})

	type getOpt struct {
		clusterBaseOptions
		NAME string `help:"Namespace name"`
	}
	R(&getOpt{}, cmdN("show"), "Show k8s namespace", func(s *mcclient.ClientSession, args *getOpt) error {
		params := args.ClusterParams()
		ret, err := k8s.Namespaces.Get(s, args.NAME, params)
		if err != nil {
			return err
		}
		printObjectYAML(ret)
		return nil
	})
}
