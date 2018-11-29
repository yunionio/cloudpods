package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
)

func initSecret() {
	cmdN := NewCmdNameFactory("secret")
	secretCmd := NewShellCommands(cmdN.Do).AddR(
		NewK8sNsResourceGetCmd(cmdN, k8s.Secrets),
		NewK8sNsResourceDeleteCmd(cmdN, k8s.Secrets),
	)
	listCmd := NewCommand(
		&o.SecretListOptions{},
		cmdN.Do("list"),
		"List secret resource",
		func(s *mcclient.ClientSession, args *o.SecretListOptions) error {
			ret, err := k8s.Secrets.List(s, args.Params())
			if err != nil {
				return err
			}
			PrintListResultTable(ret, k8s.Secrets, s)
			return nil
		})
	secretCmd.AddR(listCmd)

	registryCmd := initK8sNamespaceResource("registrysecret", k8s.RegistrySecrets)
	createCmd := NewCommand(
		&o.RegistrySecretCreateOptions{},
		resourceCmdN("registrysecret", "create"),
		"Create docker registry secret resource",
		func(s *mcclient.ClientSession, args *o.RegistrySecretCreateOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := k8s.RegistrySecrets.Create(s, params)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})
	registryCmd.AddR(createCmd)
}
