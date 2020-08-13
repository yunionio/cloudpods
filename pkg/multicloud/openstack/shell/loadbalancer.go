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
	"yunion.io/x/onecloud/pkg/multicloud/openstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type LoadbalancerListOptions struct {
	}
	type LoadbalancerPoolListOptions struct {
	}
	type LoadbalancerListenerListOptions struct {
	}
	type LoadbalancerOptions struct {
		ID string `help:"LoadbalancerId"`
	}
	type LoadbalancerListenerOptions struct {
		ID string `help:"LoadbalancerListenerId"`
	}

	shellutils.R(&LoadbalancerListOptions{}, "lb-list", "List loadbalancers", func(cli *openstack.SRegion, args *LoadbalancerListOptions) error {
		loadbalancers, err := cli.GetLoadbalancers()
		if err != nil {
			return err
		}
		printObject(loadbalancers)
		return nil
	})
	shellutils.R(&LoadbalancerOptions{}, "lb-show", "Show loadbalancer", func(cli *openstack.SRegion, args *LoadbalancerOptions) error {
		loadbalancer, err := cli.GetLoadbalancerbyId(args.ID)
		if err != nil {
			return err
		}
		printObject(loadbalancer)
		return nil
	})
	shellutils.R(&LoadbalancerOptions{}, "lb-delete", "delete loadbalancer", func(cli *openstack.SRegion, args *LoadbalancerOptions) error {
		err := cli.DeleteLoadbalancer(args.ID)
		if err != nil {
			return err
		}
		return nil
	})
	shellutils.R(&LoadbalancerPoolListOptions{}, "lbpool-list", "List loadbalancers pool", func(cli *openstack.SRegion, args *LoadbalancerPoolListOptions) error {
		loadbalancers, err := cli.GetLoadbalancerPools()
		if err != nil {
			return err
		}
		printObject(loadbalancers)
		return nil
	})

	shellutils.R(&LoadbalancerListenerListOptions{}, "lblistener-list", "List loadbalancers listener", func(cli *openstack.SRegion, args *LoadbalancerListenerListOptions) error {
		loadbalancers, err := cli.GetLoadbalancerListeners()
		if err != nil {
			return err
		}
		printObject(loadbalancers)
		return nil
	})
	shellutils.R(&LoadbalancerOptions{}, "lblistener-delete", "Delete loadbalancer listener", func(cli *openstack.SRegion, args *LoadbalancerOptions) error {
		err := cli.DeleteLoadbalancerListener(args.ID)
		if err != nil {
			return err
		}
		return nil
	})
}
