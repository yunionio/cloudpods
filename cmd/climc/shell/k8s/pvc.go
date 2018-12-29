package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
)

func initPVC() {
	cmdN := NewCmdNameFactory("pvc")
	pvcCmd := NewShellCommands(cmdN.Do).AddR(
		NewK8sNsResourceGetCmd(cmdN, k8s.PersistentVolumeClaims),
		NewK8sNsResourceDeleteCmd(cmdN, k8s.PersistentVolumeClaims),
	)
	listCmd := NewCommand(
		&o.PVCListOptions{},
		cmdN.Do("list"),
		"List PersistentVolumeClaims resource",
		func(s *mcclient.ClientSession, args *o.PVCListOptions) error {
			ret, err := k8s.PersistentVolumeClaims.List(s, args.Params())
			if err != nil {
				return err
			}
			PrintListResultTable(ret, k8s.PersistentVolumeClaims, s)
			return nil
		},
	)
	createCmd := NewCommand(
		&o.PVCCreateOptions{},
		cmdN.Do("create"),
		"Create PersistentVolumeClaims resource",
		func(s *mcclient.ClientSession, args *o.PVCCreateOptions) error {
			ret, err := k8s.PersistentVolumeClaims.Create(s, args.Params())
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		},
	)
	pvcCmd.AddR(listCmd, createCmd)
}
