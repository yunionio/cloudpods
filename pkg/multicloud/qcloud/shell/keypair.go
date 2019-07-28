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
	"yunion.io/x/onecloud/pkg/multicloud/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type KeyPairListOptions struct {
		Name   string   `help:"Keypair Name"`
		IDs    []string `help:"Keypari ids"`
		Offset int      `help:"List offset"`
		Limit  int      `help:"List limit"`
	}
	shellutils.R(&KeyPairListOptions{}, "keypair-list", "List keypair", func(cli *qcloud.SRegion, args *KeyPairListOptions) error {
		keypairs, total, err := cli.GetKeypairs(args.Name, args.IDs, args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(keypairs, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type KeyPairCreateOptions struct {
		NAME string `help:"Keypair Name"`
	}
	shellutils.R(&KeyPairCreateOptions{}, "keypair-create", "Create keypair", func(cli *qcloud.SRegion, args *KeyPairCreateOptions) error {
		keypair, err := cli.CreateKeyPair(args.NAME)
		if err != nil {
			return err
		}
		printObject(keypair)
		return nil
	})

	type KeyPairAssociateOptions struct {
		KEYPAIRID  string `help:"Keypair ID"`
		INSTANCEID string `help:"Instance ID"`
	}

	shellutils.R(&KeyPairAssociateOptions{}, "keypair-associate-instance", "Attach Keypair to a instance", func(cli *qcloud.SRegion, args *KeyPairAssociateOptions) error {
		return cli.AttachKeypair(args.INSTANCEID, args.KEYPAIRID)
	})

}
