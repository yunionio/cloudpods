package shell

import (
	"yunion.io/x/onecloud/pkg/util/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type InstanceListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&InstanceListOptions{}, "instance-list", "List intances", func(cli *azure.SRegion, args *InstanceListOptions) error {
		if instances, err := cli.GetInstances(); err != nil {
			return err
		} else {
			printList(instances, len(instances), args.Offset, args.Limit, []string{})
			return nil
		}
	})

	type InstanceCrateOptions struct {
		NAME      string `help:"name of instance"`
		IMAGE     string `help:"image ID"`
		CPU       int    `help:"CPU count"`
		MEMORYGB  int    `help:"MemoryGB"`
		Disk      []int  `help:"Data disk sizes int GB"`
		STORAGE   string `help:"Storage type"`
		NETWORK   string `help:"Network ID"`
		PASSWD    string `help:"password"`
		PublicKey string `help:"PublicKey"`
	}
	shellutils.R(&InstanceCrateOptions{}, "instance-create", "Create a instance", func(cli *azure.SRegion, args *InstanceCrateOptions) error {
		instance, e := cli.CreateInstanceSimple(args.NAME, args.IMAGE, args.CPU, args.MEMORYGB, args.STORAGE, args.Disk, args.NETWORK, args.PASSWD, args.PublicKey)
		if e != nil {
			return e
		}
		printObject(instance)
		return nil
	})

	type InstanceShowOptions struct {
		ID string `help:"Instance ID"`
	}
	shellutils.R(&InstanceShowOptions{}, "instance-show", "Show intance detail", func(cli *azure.SRegion, args *InstanceShowOptions) error {
		resourceGroup, instanceName := azure.PareResourceGroupWithName(args.ID, azure.INSTANCE_RESOURCE)
		if instance, err := cli.GetInstance(resourceGroup, instanceName); err != nil {
			return err
		} else {
			printObject(instance)
			return nil
		}
	})

}
