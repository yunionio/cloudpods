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
	"yunion.io/x/onecloud/pkg/multicloud/google"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ElbListOptions struct {
	}
	shellutils.R(&ElbListOptions{}, "lb-list", "List loadbalancers", func(cli *google.SRegion, args *ElbListOptions) error {
		elbs, err := cli.GetRegionalLoadbalancers()
		if err != nil {
			return err
		}

		printList(elbs, len(elbs), 0, 0, []string{})
		return nil
	})

	type ElbShowOptions struct {
		RESOURCEID string `json:"resourceid"`
	}
	shellutils.R(&ElbShowOptions{}, "lb-bss", "List all loadbalancer backend services", func(cli *google.SRegion, args *ElbShowOptions) error {
		lb, err := cli.GetLoadbalancer(args.RESOURCEID)
		if err != nil {
			return err
		}

		elbs, err := lb.GetBackendServices()
		if err != nil {
			return err
		}

		printList(elbs, len(elbs), 0, 0, []string{})
		return nil
	})

	shellutils.R(&ElbShowOptions{}, "lb-frs", "List all loadbalancer forward rules", func(cli *google.SRegion, args *ElbShowOptions) error {
		lb, err := cli.GetLoadbalancer(args.RESOURCEID)
		if err != nil {
			return err
		}

		elbs, err := lb.GetForwardingRules()
		if err != nil {
			return err
		}

		printList(elbs, len(elbs), 0, 0, []string{})
		return nil
	})

	shellutils.R(&ElbShowOptions{}, "lb-http", "List loadbalancer https proxies", func(cli *google.SRegion, args *ElbShowOptions) error {
		elb, err := cli.GetLoadbalancer(args.RESOURCEID)
		if err != nil {
			return err
		}

		hps, e := elb.GetTargetHttpProxies()
		if e != nil {
			return e
		}
		printList(hps, len(hps), 0, 0, []string{})
		return nil
	})

	shellutils.R(&ElbShowOptions{}, "lb-https", "List loadbalancer https proxies", func(cli *google.SRegion, args *ElbShowOptions) error {
		elb, err := cli.GetLoadbalancer(args.RESOURCEID)
		if err != nil {
			return err
		}

		hps, e := elb.GetTargetHttpsProxies()
		if e != nil {
			return e
		}
		printList(hps, len(hps), 0, 0, []string{})
		return nil
	})

	shellutils.R(&ElbShowOptions{}, "lb-igs", "List all loadbalancer instance groups", func(cli *google.SRegion, args *ElbShowOptions) error {
		elb, err := cli.GetLoadbalancer(args.RESOURCEID)
		if err != nil {
			return err
		}

		hps, e := elb.GetInstanceGroups()
		if e != nil {
			return e
		}
		printList(hps, len(hps), 0, 0, []string{})
		return nil
	})

	shellutils.R(&ElbShowOptions{}, "lbl-list", "List all loadbalancer listeners", func(cli *google.SRegion, args *ElbShowOptions) error {
		elb, err := cli.GetLoadbalancer(args.RESOURCEID)
		if err != nil {
			return err
		}

		hps, e := elb.GetLoadbalancerListeners()
		if e != nil {
			return e
		}
		printList(hps, len(hps), 0, 0, []string{})
		return nil
	})

	shellutils.R(&ElbShowOptions{}, "lblr-list", "List all loadbalancer listener rules", func(cli *google.SRegion, args *ElbShowOptions) error {
		elb, err := cli.GetLoadbalancer(args.RESOURCEID)
		if err != nil {
			return err
		}

		hps, e := elb.GetLoadbalancerListeners()
		if e != nil {
			return e
		}

		rules := make([]google.SLoadbalancerListenerRule, 0)

		for i := range hps {
			_rules, ee := hps[i].GetLoadbalancerListenerRules()
			if e != nil {
				return ee
			}

			rules = append(rules, _rules...)
		}

		printList(rules, len(rules), 0, 0, []string{})
		return nil
	})

	shellutils.R(&ElbShowOptions{}, "lbbg-list", "List all loadbalancer backendgroups", func(cli *google.SRegion, args *ElbShowOptions) error {
		lb, err := cli.GetLoadbalancer(args.RESOURCEID)
		if err != nil {
			return err
		}

		lbbg, e := lb.GetLoadbalancerBackendGroups()
		if e != nil {
			return e
		}

		printList(lbbg, len(lbbg), 0, 0, []string{})
		return nil
	})

	shellutils.R(&ElbShowOptions{}, "lbb-list", "List all loadbalancer backends", func(cli *google.SRegion, args *ElbShowOptions) error {
		lb, err := cli.GetLoadbalancer(args.RESOURCEID)
		if err != nil {
			return err
		}

		lbbg, e := lb.GetLoadbalancerBackendGroups()
		if e != nil {
			return e
		}

		lbbs := make([]google.SLoadbalancerBackend, 0)
		for i := range lbbg {
			backends, err := lbbg[i].GetLoadbalancerBackends()
			if err != nil {
				return err
			}

			lbbs = append(lbbs, backends...)
		}

		printList(lbbs, len(lbbs), 0, 0, []string{})
		return nil
	})

	shellutils.R(&ElbShowOptions{}, "lbhc-list", "List all loadbalancer health checks", func(cli *google.SRegion, args *ElbShowOptions) error {
		lb, err := cli.GetLoadbalancer(args.RESOURCEID)
		if err != nil {
			return err
		}

		hcs, e := lb.GetHealthChecks()
		if e != nil {
			return e
		}

		printList(hcs, len(hcs), 0, 0, []string{})
		return nil
	})

	shellutils.R(&ElbListOptions{}, "cert-list", "List region certificates", func(cli *google.SRegion, args *ElbListOptions) error {
		certs, err := cli.GetRegionalSslCertificates("")
		if err != nil {
			return err
		}

		printList(certs, len(certs), 0, 0, []string{})
		return nil
	})
}
