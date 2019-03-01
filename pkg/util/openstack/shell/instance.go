package shell

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/util/openstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type InstanceListOptions struct {
		Host string `help:"Host name for filter instance list"`
	}
	shellutils.R(&InstanceListOptions{}, "instance-list", "List instances", func(cli *openstack.SRegion, args *InstanceListOptions) error {
		instances, err := cli.GetInstances(args.Host)
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

}
