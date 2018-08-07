package shell

import (
	"github.com/yunionio/onecloud/pkg/util/shellutils"
	"github.com/yunionio/onecloud/pkg/util/esxi"
)

func init() {
	type HostListOptions struct {
		DATACENTER string `help:"List hosts in datacenter"`
	}
	shellutils.R(&HostListOptions{}, "host-list", "List hosts in datacenter", func(cli *esxi.SESXiClient, args *HostListOptions) error {
		dc, err := cli.FindDatacenterById(args.DATACENTER)
		if err != nil {
			return err
		}
		hosts, err := dc.GetIHosts()
		if err != nil {
			return err
		}
		printList(hosts, nil)
		return nil
	})

	type HostShowOptions struct {
		IP string `help:"Host IP"`
	}
	shellutils.R(&HostShowOptions{}, "host-show", "Show details of a host by IP", func(cli *esxi.SESXiClient, args *HostShowOptions) error {
		host, err := cli.FindHostByIp(args.IP)
		if err != nil {
			return err
		}
		printObject(host)
		return nil
	})
}