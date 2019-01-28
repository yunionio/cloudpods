package shell

import (
	"yunion.io/x/onecloud/pkg/baremetal/utils/ipmitool"
	"yunion.io/x/onecloud/pkg/util/printutils"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type LanOptions struct {
		CHANNEL int `help:"lan channel"`
	}
	shellutils.R(&LanOptions{}, "set-lan-dhcp", "Set lan channel DHCP", func(client ipmitool.IPMIExecutor, args *LanOptions) error {
		return ipmitool.SetLanDHCP(client, args.CHANNEL)
	})

	shellutils.R(&LanOptions{}, "get-lan-config", "Get lan channel config", func(cli ipmitool.IPMIExecutor, args *LanOptions) error {
		config, err := ipmitool.GetLanConfig(cli, args.CHANNEL)
		if err != nil {
			return err
		}
		printutils.PrintInterfaceObject(config)
		return nil
	})

	type SetLanStaticIpOptions struct {
		LanOptions
		IP      string
		MASK    string
		GATEWAY string
	}
	shellutils.R(&SetLanStaticIpOptions{}, "set-lan-static", "Set lan static network", func(cli ipmitool.IPMIExecutor, args *SetLanStaticIpOptions) error {
		return ipmitool.SetLanStatic(cli, args.CHANNEL, args.IP, args.MASK, args.GATEWAY)
	})
}
