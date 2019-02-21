package shell

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/util/openstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type InstanceListOptions struct {
		ZoneID string `help:"Zone ID for filter instance list"`
		Host   string `help:"Host name for filter instance list"`
	}
	shellutils.R(&InstanceListOptions{}, "instance-list", "List instances", func(cli *openstack.SRegion, args *InstanceListOptions) error {
		instances, err := cli.GetInstances(args.ZoneID, args.Host)
		if err != nil {
			return err
		}
		printList(instances, 0, 0, 0, nil)
		return nil
	})

	type InstanceOptions struct {
		ID string `help:"Instance ID"`
	}
	shellutils.R(&InstanceOptions{}, "instance-show", "Show instance", func(cli *openstack.SRegion, args *InstanceOptions) error {
		instance, err := cli.GetInstance(args.ID)
		if err != nil {
			return err
		}
		printObject(instance)
		return nil
	})

	shellutils.R(&InstanceOptions{}, "instance-vnc", "Show instance vnc url", func(cli *openstack.SRegion, args *InstanceOptions) error {
		url, err := cli.GetInstanceVNCUrl(args.ID)
		if err != nil {
			return err
		}
		fmt.Println(url)
		return nil
	})

	type InstanceDeployOptions struct {
		ID       string `help:"Instance ID"`
		Password string `help:"Instance password"`
		Name     string `help:"Instance name"`
	}

	shellutils.R(&InstanceDeployOptions{}, "instance-deploy", "Deploy instance", func(cli *openstack.SRegion, args *InstanceDeployOptions) error {
		return cli.DeployVM(args.ID, args.Name, args.Password, "", false, "")
	})

	type InstanceChangeConfigOptions struct {
		ID        string `help:"Instance ID"`
		FLAVOR_ID string `help:"Flavor ID"`
	}

	shellutils.R(&InstanceChangeConfigOptions{}, "instance-change-config", "Change instance config", func(cli *openstack.SRegion, args *InstanceChangeConfigOptions) error {
		instance, err := cli.GetInstance(args.ID)
		if err != nil {
			return err
		}
		return cli.ChangeConfig(instance, args.FLAVOR_ID)
	})

}
