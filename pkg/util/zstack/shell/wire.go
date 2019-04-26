package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/zstack"
)

func init() {
	type WireListOptions struct {
		ZoneId    string
		WireId    string
		ClusterId string
	}
	shellutils.R(&WireListOptions{}, "wire-list", "List wires", func(cli *zstack.SRegion, args *WireListOptions) error {
		wires, err := cli.GetWires(args.ZoneId, args.WireId, args.ClusterId)
		if err != nil {
			return err
		}
		printList(wires, len(wires), 0, 0, []string{})
		return nil
	})
}
