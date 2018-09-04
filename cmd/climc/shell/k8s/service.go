package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func initService() {
	cmdN := func(suffix string) string {
		return resourceCmdN("service", suffix)
	}

	type listOpt struct {
		namespaceListOptions
		baseListOptions
	}
	R(&listOpt{}, cmdN("list"), "List k8s service", func(s *mcclient.ClientSession, args *listOpt) error {
		params := fetchNamespaceParams(args.namespaceListOptions)
		params.Update(fetchPagingParams(args.baseListOptions))
		params.Update(args.ClusterParams())
		ret, err := k8s.Services.List(s, params)
		if err != nil {
			return err
		}
		PrintListResultTable(ret, k8s.Services, s)
		return nil
	})
}
