package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/zstack"
)

func init() {
	type OfferingListOptions struct {
		OfferId string
	}
	shellutils.R(&OfferingListOptions{}, "offering-list", "List instance offerings", func(cli *zstack.SRegion, args *OfferingListOptions) error {
		offerings, err := cli.GetInstanceOfferings(args.OfferId)
		if err != nil {
			return err
		}
		printList(offerings, len(offerings), 0, 0, []string{})
		return nil
	})
}
