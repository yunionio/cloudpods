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
	"strings"

	"yunion.io/x/onecloud/pkg/multicloud/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type LoadbalancerListOptions struct {
	}
	shellutils.R(&LoadbalancerListOptions{}, "lb-list", "List loadbalancers", func(cli *azure.SRegion, args *LoadbalancerListOptions) error {
		lbs, err := cli.GetILoadBalancers()
		if err != nil {
			return err
		}
		printList(lbs, len(lbs), 0, 0, []string{})
		return nil
	})

	type LoadbalancerOptions struct {
		ID string `help:"ID of loadbalancer"`
	}
	shellutils.R(&LoadbalancerOptions{}, "lb-show", "Show loadbalancer", func(cli *azure.SRegion, args *LoadbalancerOptions) error {
		lb, err := cli.GetILoadBalancerById(args.ID)
		if err != nil {
			return err
		}
		printObject(lb)
		return nil
	})

	shellutils.R(&LoadbalancerListOptions{}, "lbcert-list", "List loadbalancers certs", func(cli *azure.SRegion, args *LoadbalancerListOptions) error {
		certs, err := cli.GetILoadBalancerCertificates()
		if err != nil {
			return err
		}
		printList(certs, len(certs), 0, 0, []string{})
		return nil
	})

	type LoadbalancerCertOptions struct {
		ID string `help:"ID of loadbalancer cert"`
	}

	shellutils.R(&LoadbalancerCertOptions{}, "lbcert-show", "Show loadbalancer cert", func(cli *azure.SRegion, args *LoadbalancerCertOptions) error {
		cert, err := cli.GetILoadBalancerCertificateById(args.ID)
		if err != nil {
			return err
		}
		printObject(cert)
		return nil
	})

	shellutils.R(&LoadbalancerOptions{}, "lbl-list", "List loadbalancer listeners", func(cli *azure.SRegion, args *LoadbalancerOptions) error {
		lb, err := cli.GetILoadBalancerById(args.ID)
		if err != nil {
			return err
		}
		lbl, err := lb.GetILoadBalancerListeners()
		if err != nil {
			return err
		}
		printList(lbl, len(lbl), 0, 0, []string{})
		return nil
	})

	shellutils.R(&LoadbalancerOptions{}, "lbbg-list", "List loadbalancer listeners", func(cli *azure.SRegion, args *LoadbalancerOptions) error {
		lb, err := cli.GetILoadBalancerById(args.ID)
		if err != nil {
			return err
		}
		lbbgs, err := lb.GetILoadBalancerBackendGroups()
		if err != nil {
			return err
		}
		printList(lbbgs, len(lbbgs), 0, 0, []string{})
		return nil
	})

	type LoadbalancerBackendOptions struct {
		BGID string `help:"ID of loadbalancer backendgroup"`
	}
	shellutils.R(&LoadbalancerBackendOptions{}, "lbb-list", "List loadbalancer listeners", func(cli *azure.SRegion, args *LoadbalancerBackendOptions) error {
		lb, err := cli.GetILoadBalancerById(strings.Split(args.BGID, "/backendAddressPools")[0])
		if err != nil {
			return err
		}
		lbbg, err := lb.GetILoadBalancerBackendGroupById(args.BGID)
		if err != nil {
			return err
		}

		lbbs, err := lbbg.GetILoadbalancerBackends()
		if err != nil {
			return err
		}

		printList(lbbs, len(lbbs), 0, 0, []string{})
		return nil
	})

	type LoadbalancerRuleOptions struct {
		LISTENID string `help:"ID of loadbalancer listener"`
	}
	shellutils.R(&LoadbalancerRuleOptions{}, "lbr-list", "List loadbalancer listener rules", func(cli *azure.SRegion, args *LoadbalancerRuleOptions) error {
		lbId := ""
		if strings.Index(args.LISTENID, "/requestRoutingRules") > 0 {
			lbId = strings.Split(args.LISTENID, "/requestRoutingRules")[0]
		} else {
			return nil
		}
		lb, err := cli.GetILoadBalancerById(lbId)
		if err != nil {
			return err
		}
		lbl, err := lb.GetILoadBalancerListenerById(args.LISTENID)
		if err != nil {
			return err
		}

		lbrs, err := lbl.GetILoadbalancerListenerRules()
		if err != nil {
			return err
		}

		printList(lbrs, len(lbrs), 0, 0, []string{})
		return nil
	})
}
