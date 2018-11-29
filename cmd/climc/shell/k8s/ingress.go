package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
)

func initIngress() {
	cmdN := initK8sNamespaceResource("ingress", k8s.Ingresses)
	createCmd := NewCommand(
		&o.IngressCreateOptions{},
		cmdN.CommandNameFactory("create"),
		"Create ingress rules to service",
		func(s *mcclient.ClientSession, args *o.IngressCreateOptions) error {
			spec := args.Params()
			ret, err := k8s.Ingresses.Create(s, spec)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})
	cmdN.AddR(createCmd)
}
