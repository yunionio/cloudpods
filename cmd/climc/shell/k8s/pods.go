package k8s

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func initPod() {
	cmdN := func(suffix string) string {
		return resourceCmdN("pod", suffix)
	}

	type listOpt struct {
		namespaceListOptions
		baseListOptions
	}
	R(&listOpt{}, cmdN("list"), "List k8s pod", func(s *mcclient.ClientSession, args *listOpt) error {
		params := fetchNamespaceParams(args.namespaceListOptions)
		params.Update(fetchPagingParams(args.baseListOptions))
		ret, err := k8s.Pods.ListInContexts(s, params, args.ClusterContext())
		if err != nil {
			return err
		}
		PrintListResultTable(ret, k8s.Pods, s)
		return nil
	})

	type deleteOpt struct {
		resourceGetOptions
	}
	R(&deleteOpt{}, cmdN("delete"), "Delete pod", func(s *mcclient.ClientSession, args *deleteOpt) error {
		id := args.NAME
		ret, err := k8s.Pods.DeleteInContexts(s, id, args.ToJSON(), args.ClusterContext())
		if err != nil {
			return err
		}
		fmt.Println(ret)
		return nil
	})
}
