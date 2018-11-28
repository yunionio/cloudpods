package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
)

func initStatefulset() {
	cmd := initK8sNamespaceResource("statefulset", k8s.StatefulSets)
	cmdN := cmd.CommandNameFactory

	createCmd := NewCommand(
		&o.StatefulSetCreateOptions{},
		cmdN("create"),
		"Create statefulset resource",
		func(s *mcclient.ClientSession, args *o.StatefulSetCreateOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := k8s.StatefulSets.Create(s, params)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})

	cmd.AddR(createCmd)
}
