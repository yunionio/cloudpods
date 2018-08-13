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
		k8sBaseListOptions
	}
	R(&listOpt{}, cmdN("list"), "List k8s pod", func(s *mcclient.ClientSession, args *listOpt) error {
		ret, err := k8s.Pods.ListInContexts(s, nil, args.ClusterContext())
		if err != nil {
			return err
		}
		fmt.Println(ret)
		return nil
	})
}
