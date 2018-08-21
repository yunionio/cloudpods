package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func initConfigMap() {
	cmdN := func(suffix string) string {
		return resourceCmdN("configmap", suffix)
	}

	type listOpt struct {
		namespaceListOptions
		baseListOptions
	}
	R(&listOpt{}, cmdN("list"), "List k8s configmap", func(s *mcclient.ClientSession, args *listOpt) error {
		params := fetchNamespaceParams(args.namespaceListOptions)
		params.Update(fetchPagingParams(args.baseListOptions))
		ret, err := k8s.ConfigMaps.ListInContexts(s, params, args.ClusterContext())
		if err != nil {
			return err
		}
		printList(ret, k8s.ConfigMaps.GetColumns(s))
		return nil
	})
}
