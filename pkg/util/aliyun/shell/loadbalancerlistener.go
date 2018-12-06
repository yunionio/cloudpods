package shell

import (
	"yunion.io/x/onecloud/pkg/util/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type LoadbalancerListenerListOptions struct {
		ID   string `help:"ID of loadbalancer"`
		PORT int    `help:"PORT of listenerPort"`
	}
	shellutils.R(&LoadbalancerListenerListOptions{}, "lb-http-listener-show", "Show LoadbalancerHTTPListener", func(cli *aliyun.SRegion, args *LoadbalancerListenerListOptions) error {
		listener, err := cli.GetLoadbalancerHTTPListener(args.ID, args.PORT)
		if err != nil {
			return err
		}
		printObject(listener)
		return nil
	})

	shellutils.R(&LoadbalancerListenerListOptions{}, "lb-https-listener-show", "Show LoadbalancerHTTPSListener", func(cli *aliyun.SRegion, args *LoadbalancerListenerListOptions) error {
		listener, err := cli.GetLoadbalancerHTTPSListener(args.ID, args.PORT)
		if err != nil {
			return err
		}
		printObject(listener)
		return nil
	})

	shellutils.R(&LoadbalancerListenerListOptions{}, "lb-tcp-listener-show", "Show LoadbalancerTCPListener", func(cli *aliyun.SRegion, args *LoadbalancerListenerListOptions) error {
		listener, err := cli.GetLoadbalancerTCPListener(args.ID, args.PORT)
		if err != nil {
			return err
		}
		printObject(listener)
		return nil
	})

	shellutils.R(&LoadbalancerListenerListOptions{}, "lb-udp-listener-show", "Show LoadbalancerUDPListener", func(cli *aliyun.SRegion, args *LoadbalancerListenerListOptions) error {
		listener, err := cli.GetLoadbalancerUDPListener(args.ID, args.PORT)
		if err != nil {
			return err
		}
		printObject(listener)
		return nil
	})

}
