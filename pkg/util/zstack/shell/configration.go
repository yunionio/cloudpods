package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/zstack"
)

func init() {
	type ConfigrationListOptions struct {
	}
	shellutils.R(&ConfigrationListOptions{}, "configration-list", "List configration", func(cli *zstack.SRegion, args *ConfigrationListOptions) error {
		configrations, err := cli.GetConfigrations()
		if err != nil {
			return err
		}
		printList(configrations, len(configrations), 0, 0, []string{})
		return nil
	})
}
