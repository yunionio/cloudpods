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
	"yunion.io/x/onecloud/pkg/multicloud/hcs"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type VpcPeerListOptions struct {
		VpcId string
	}
	shellutils.R(&VpcPeerListOptions{}, "vpc-peer-list", "List vpc peers", func(cli *hcs.SRegion, args *VpcPeerListOptions) error {
		peers, err := cli.GetVpcPeerings(args.VpcId)
		if err != nil {
			return nil
		}
		printList(peers, 0, 0, 0, nil)
		return nil
	})

	type VpcPeerIdOption struct {
		ID string
	}

	shellutils.R(&VpcPeerIdOption{}, "vpc-peer-show", "Show vpc peer", func(cli *hcs.SRegion, args *VpcPeerIdOption) error {
		ret, err := cli.GetVpcPeering(args.ID)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	shellutils.R(&VpcPeerIdOption{}, "vpc-peer-delete", "Delete vpc peer", func(cli *hcs.SRegion, args *VpcPeerIdOption) error {
		return cli.DeleteVpcPeering(args.ID)
	})

}
