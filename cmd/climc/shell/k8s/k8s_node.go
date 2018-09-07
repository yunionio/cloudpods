package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func initK8sNode() {
	cmdN := func(suffix string) string {
		return resourceCmdN("node", suffix)
	}

	type listOpt struct {
		clusterBaseOptions
		baseListOptions
	}
	R(&listOpt{}, cmdN("list"), "List k8s nodes resource", func(s *mcclient.ClientSession, args *listOpt) error {
		params := fetchPagingParams(args.baseListOptions)
		params.Update(args.ClusterParams())
		ret, err := k8s.K8sNodes.List(s, params)
		if err != nil {
			return err
		}
		PrintListResultTable(ret, k8s.K8sNodes, s)
		return nil
	})

	type getOpt struct {
		clusterBaseOptions
		NAME string `help:"Node name"`
	}
	R(&getOpt{}, cmdN("show"), "Show k8s node", func(s *mcclient.ClientSession, args *getOpt) error {
		params := args.ClusterParams()
		ret, err := k8s.K8sNodes.Get(s, args.NAME, params)
		if err != nil {
			return err
		}
		printObjectYAML(ret)
		return nil
	})
}
