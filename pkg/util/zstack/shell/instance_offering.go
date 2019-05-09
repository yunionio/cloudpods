package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/zstack"
)

func init() {
	type InstanceOfferingListOptions struct {
		OfferId  string
		Name     string
		Cpu      int
		MemoryMb int
	}
	shellutils.R(&InstanceOfferingListOptions{}, "instance-offering-list", "List instance offerings", func(cli *zstack.SRegion, args *InstanceOfferingListOptions) error {
		offerings, err := cli.GetInstanceOfferings(args.OfferId, args.Name, args.Cpu, args.MemoryMb)
		if err != nil {
			return err
		}
		printList(offerings, len(offerings), 0, 0, []string{})
		return nil
	})
}
