package shell

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/printutils"
	"yunion.io/x/onecloud/pkg/util/redfish"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {

	type VirtualMediaGetOptions struct {
	}
	shellutils.R(&VirtualMediaGetOptions{}, "cdrom-get", "Get details of manager virtual media", func(cli redfish.IRedfishDriver, args *VirtualMediaGetOptions) error {
		path, image, err := cli.GetVirtualCdromInfo(context.Background())
		if err != nil {
			return err
		}
		fmt.Println("Path:", path)
		fmt.Println("Image:", image.Image)
		fmt.Println("ApiSupport:", image.SupportAction)
		return nil
	})

	type VirtualMediaMountOptions struct {
		URL string `help:"cdrom http URL"`
	}
	shellutils.R(&VirtualMediaMountOptions{}, "cdrom-insert", "Insert iso into virtual CD-ROM", func(cli redfish.IRedfishDriver, args *VirtualMediaMountOptions) error {
		ctx := context.Background()
		path, cdInfo, err := cli.GetVirtualCdromInfo(ctx)
		if err != nil {
			return err
		}
		if len(cdInfo.Image) > 0 {
			return fmt.Errorf("image %s in cd-rom", cdInfo.Image)
		}
		if !cdInfo.SupportAction {
			return fmt.Errorf("action not supported")
		}
		err = cli.MountVirtualCdrom(context.Background(), path, args.URL)
		if err != nil {
			return err
		}
		fmt.Println("Success!")
		return nil
	})

	type VirtualMediaUmountOptions struct {
	}
	shellutils.R(&VirtualMediaUmountOptions{}, "cdrom-eject", "Eject iso from virtual CD-ROM", func(cli redfish.IRedfishDriver, args *VirtualMediaUmountOptions) error {
		ctx := context.Background()
		path, cdInfo, err := cli.GetVirtualCdromInfo(ctx)
		if err != nil {
			return err
		}
		if len(cdInfo.Image) == 0 {
			return fmt.Errorf("no image in cd-rom")
		}
		if !cdInfo.SupportAction {
			return fmt.Errorf("action not supported")
		}
		err = cli.UmountVirtualCdrom(context.Background(), path)
		if err != nil {
			return err
		}
		fmt.Println("Success!")
		return nil
	})

	type ReadLogsOptions struct {
	}
	shellutils.R(&ReadLogsOptions{}, "system-logs", "Read system logs", func(cli redfish.IRedfishDriver, args *ReadLogsOptions) error {
		events, err := cli.ReadSystemLogs(context.Background())
		if err != nil {
			return err
		}
		printutils.PrintInterfaceList(events, 0, 0, 0, nil)
		return nil
	})
	shellutils.R(&ReadLogsOptions{}, "manager-logs", "Read manager logs", func(cli redfish.IRedfishDriver, args *ReadLogsOptions) error {
		events, err := cli.ReadManagerLogs(context.Background())
		if err != nil {
			return err
		}
		printutils.PrintInterfaceList(events, 0, 0, 0, nil)
		return nil
	})
	shellutils.R(&ReadLogsOptions{}, "system-logs-clear", "Clear system logs", func(cli redfish.IRedfishDriver, args *ReadLogsOptions) error {
		err := cli.ClearSystemLogs(context.Background())
		if err != nil {
			return err
		}
		return nil
	})
	shellutils.R(&ReadLogsOptions{}, "manager-logs-clear", "Clear manager logs", func(cli redfish.IRedfishDriver, args *ReadLogsOptions) error {
		err := cli.ClearManagerLogs(context.Background())
		if err != nil {
			return err
		}
		return nil
	})

	shellutils.R(&ReadLogsOptions{}, "bmc-reset", "reset bmc", func(cli redfish.IRedfishDriver, args *ReadLogsOptions) error {
		err := cli.BmcReset(context.Background())
		if err != nil {
			return err
		}
		fmt.Println("Success!")
		return nil
	})

	type NtpConfGetOptions struct {
	}
	shellutils.R(&NtpConfGetOptions{}, "ntp-get", "Get ntp configuration", func(cli redfish.IRedfishDriver, args *NtpConfGetOptions) error {
		conf, err := cli.GetNTPConf(context.Background())
		if err != nil {
			return err
		}
		fmt.Println(jsonutils.Marshal(conf).PrettyString())
		return nil
	})

	type NtpConfSetOptions struct {
		SERVER   []string `help:"ntp servers"`
		TimeZone string   `help:"time zone, e.g. Asia/Shanghai"`
	}
	shellutils.R(&NtpConfSetOptions{}, "ntp-set", "Set ntp configuration", func(cli redfish.IRedfishDriver, args *NtpConfSetOptions) error {
		ntpConf := redfish.SNTPConf{}
		ntpConf.ProtocolEnabled = true
		ntpConf.NTPServers = args.SERVER
		ntpConf.TimeZone = args.TimeZone
		err := cli.SetNTPConf(context.Background(), ntpConf)
		if err != nil {
			return err
		}
		fmt.Println("Success!")
		return nil
	})

	type JNLPGetOptions struct {
		Save string `help:"save to file"`
	}
	shellutils.R(&JNLPGetOptions{}, "jnlp-get", "Get Java Console JNLP file", func(cli redfish.IRedfishDriver, args *JNLPGetOptions) error {
		jnlp, err := cli.GetConsoleJNLP(context.Background())
		if err != nil {
			return err
		}
		if len(args.Save) > 0 {
			return fileutils2.FilePutContents(args.Save, jnlp, false)
		} else {
			fmt.Println(jnlp)
			return nil
		}
	})
}
