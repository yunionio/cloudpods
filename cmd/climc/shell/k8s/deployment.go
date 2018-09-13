package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
)

func initDeployment() {
	cmd := initK8sNamespaceResource("deployment", k8s.Deployments)
	cmdN := cmd.CommandNameFactory

	createCmd := NewCommand(
		&o.DeploymentCreateOptions{},
		cmdN("create"),
		"Create deployment resource",
		func(s *mcclient.ClientSession, args *o.DeploymentCreateOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := k8s.Deployments.Create(s, params)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})

	createFromFileCmd := NewCommand(
		&o.DeploymentCreateFromFileOptions{},
		"k8s-create",
		"Create resource by file",
		func(s *mcclient.ClientSession, args *o.DeploymentCreateFromFileOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := k8s.DeployFromFile.Create(s, params)
			if err != nil {
				return err
			}
			printObjectYAML(ret)
			return nil
		})

	cmd.AddR(createCmd, createFromFileCmd)
}
