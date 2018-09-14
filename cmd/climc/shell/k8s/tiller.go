package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
)

func initTiller() {
	cmdN := func(suffix string) string {
		return resourceCmdN("tiller", suffix)
	}
	R(&o.TillerCreateOptions{}, cmdN("create"), "Install helm tiller server to Kubernetes cluster", func(s *mcclient.ClientSession, args *o.TillerCreateOptions) error {
		params := args.Params()
		ret, err := k8s.Tiller.Create(s, params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
}
