// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package shell

import (
	"yunion.io/x/onecloud/pkg/multicloud/aliyun"
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
