package shell

import (
	"fmt"
	"github.com/yunionio/onecloud/pkg/util/aliyun"
)

func init() {
	type InstanceListOptions struct {
		Id     []string `help:"IDs of instances to show"`
		Zone   string   `help:"Zone ID"`
		Limit  int      `help:"page size"`
		Offset int      `help:"page offset"`
	}
	R(&InstanceListOptions{}, "instance-list", "List intances", func(cli *aliyun.SRegion, args *InstanceListOptions) error {
		instances, total, e := cli.GetInstances(args.Zone, args.Id, args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(instances, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type InstanceCrateOptions struct {
		NAME      string `help:"name of instance"`
		IMAGE     string `help:"image ID"`
		CPU       int    `help:"CPU count"`
		MEMORYGB  int    `help:"MemoryGB"`
		Disk      []int  `help:"Data disk sizes int GB"`
		STORAGE   string `help:"Storage type"`
		VSWITCH   string `help:"Vswitch ID"`
		PASSWD    string `help:"password"`
		PublicKey string `help:"PublicKey"`
	}
	R(&InstanceCrateOptions{}, "instance-create", "Create a instance", func(cli *aliyun.SRegion, args *InstanceCrateOptions) error {
		instance, e := cli.CreateInstanceSimple(args.NAME, args.IMAGE, args.CPU, args.MEMORYGB, args.STORAGE, args.Disk, args.VSWITCH, args.PASSWD, args.PublicKey)
		if e != nil {
			return e
		}
		printObject(instance)
		return nil
	})

	type InstanceOperationOptions struct {
		ID string `help:"instance ID"`
	}
	R(&InstanceOperationOptions{}, "instance-start", "Start a instance", func(cli *aliyun.SRegion, args *InstanceOperationOptions) error {
		err := cli.StartVM(args.ID)
		if err != nil {
			return err
		}
		return nil
	})

	R(&InstanceOperationOptions{}, "instance-vnc", "Get a instance VNC url", func(cli *aliyun.SRegion, args *InstanceOperationOptions) error {
		url, err := cli.GetInstanceVNCUrl(args.ID)
		if err != nil {
			return err
		}
		fmt.Println(url)
		return nil
	})

	type InstanceStopOptions struct {
		ID    string `help:"instance ID"`
		Force bool   `help:"Force stop instance"`
	}
	R(&InstanceStopOptions{}, "instance-stop", "Stop a instance", func(cli *aliyun.SRegion, args *InstanceStopOptions) error {
		err := cli.StopVM(args.ID, args.Force)
		if err != nil {
			return err
		}
		return nil
	})
	R(&InstanceOperationOptions{}, "instance-delete", "Delete a instance", func(cli *aliyun.SRegion, args *InstanceOperationOptions) error {
		err := cli.DeleteVM(args.ID)
		if err != nil {
			return err
		}
		return nil
	})
}
