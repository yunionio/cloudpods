package shell

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/zstack"
)

func init() {
	type InstanceListOptions struct {
		HostId     string
		InstanceId string
		NicId      string
	}
	shellutils.R(&InstanceListOptions{}, "instance-list", "List instances", func(cli *zstack.SRegion, args *InstanceListOptions) error {
		instances, err := cli.GetInstances(args.HostId, args.InstanceId, args.NicId)
		if err != nil {
			return err
		}
		printList(instances, len(instances), 0, 0, []string{})
		return nil
	})

	type InstanceOperation struct {
		ID string
	}

	shellutils.R(&InstanceOperation{}, "instance-console-password", "Show instance console password", func(cli *zstack.SRegion, args *InstanceOperation) error {
		password, err := cli.GetInstanceConsolePassword(args.ID)
		if err != nil {
			return err
		}
		fmt.Println(password)
		return nil
	})

	shellutils.R(&InstanceOperation{}, "instance-console-info", "Show instance console info", func(cli *zstack.SRegion, args *InstanceOperation) error {
		info, err := cli.GetInstanceConsoleInfo(args.ID)
		if err != nil {
			return err
		}
		printObject(info)
		return nil
	})

}
