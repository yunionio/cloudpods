package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
)

func initService() {
	cmdN := NewCmdNameFactory("service")
	svcCmd := NewShellCommands(cmdN.Do).AddR(
		NewK8sNsResourceGetCmd(cmdN, k8s.Services),
		NewK8sNsResourceDeleteCmd(cmdN, k8s.Services),
	)

	listCmd := NewCommand(
		&o.ServiceListOptions{},
		cmdN.Do("list"),
		"List Services resource",
		func(s *mcclient.ClientSession, args *o.ServiceListOptions) error {
			ret, err := k8s.Services.List(s, args.Params())
			if err != nil {
				return err
			}
			PrintListResultTable(ret, k8s.Services, s)
			return nil
		},
	)

	createCmd := NewCommand(
		&o.ServiceCreateOptions{},
		cmdN.Do("create"),
		"Create service resource",
		func(s *mcclient.ClientSession, args *o.ServiceCreateOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := k8s.Services.Create(s, params)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})
	svcCmd.AddR(listCmd, createCmd)
}
