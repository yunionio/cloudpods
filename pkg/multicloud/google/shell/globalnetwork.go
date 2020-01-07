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
	type GlobalNetworkListOptions struct {
		MaxResults int
		PageToken  string
	}
	shellutils.R(&GlobalNetworkListOptions{}, "global-network-list", "List globalnetworks", func(cli *google.SRegion, args *GlobalNetworkListOptions) error {
		globalnetworks, err := cli.GetClient().GetGlobalNetworks(args.MaxResults, args.PageToken)
		if err != nil {
			return err
		}
		printList(globalnetworks, 0, 0, 0, nil)
		return nil
	})

	type GlobalNetworkShowOptions struct {
		ID string
	}
	shellutils.R(&GlobalNetworkShowOptions{}, "global-network-show", "Show globalnetwork", func(cli *google.SRegion, args *GlobalNetworkShowOptions) error {
		globalnetwork, err := cli.GetClient().GetGlobalNetwork(args.ID)
		if err != nil {
			return err
		}
		printObject(globalnetwork)
		return nil
	})

	type GlobalNetworkCreateOptions struct {
		NAME string
		Desc string
	}

	shellutils.R(&GlobalNetworkCreateOptions{}, "global-network-create", "Create globalnetwork", func(cli *google.SRegion, args *GlobalNetworkCreateOptions) error {
		globalnetwork, err := cli.CreateGlobalNetwork(args.NAME, args.Desc)
		if err != nil {
			return err
		}
		printObject(globalnetwork)
		return nil
	})

}
