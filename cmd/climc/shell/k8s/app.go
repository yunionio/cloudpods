package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
)

func initApp() {
	R(
		&o.K8sAppCreateFromFileOptions{},
		"k8s-create",
		"Create resource by file",
		func(s *mcclient.ClientSession, args *o.K8sAppCreateFromFileOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := k8s.AppFromFile.Create(s, params)
			if err != nil {
				return err
			}
			printObjectYAML(ret)
			return nil
		})
}
