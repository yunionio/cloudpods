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
	"yunion.io/x/onecloud/pkg/multicloud/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type NatListOptions struct {
		Ids      []string
		VpcId    string
		SubnetId string
	}
	shellutils.R(&NatListOptions{}, "natgateway-list", "List natgateway", func(cli *aws.SRegion, args *NatListOptions) error {
		nats, err := cli.GetNatGateways(args.Ids, args.VpcId, args.SubnetId)
		if err != nil {
			return err
		}
		printList(nats, 0, 0, 0, nil)
		return nil
	})

	type NatIdOptions struct {
		ID string
	}

	shellutils.R(&NatIdOptions{}, "natgateway-delete", "Delete natgateway", func(cli *aws.SRegion, args *NatIdOptions) error {
		return cli.DeleteNatgateway(args.ID)
	})

}
