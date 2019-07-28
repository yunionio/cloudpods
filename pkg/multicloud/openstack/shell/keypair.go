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
	type KeypairListOptions struct {
	}
	shellutils.R(&KeypairListOptions{}, "keypair-list", "List keypairs", func(cli *openstack.SRegion, args *KeypairListOptions) error {
		keypairs, err := cli.GetKeypairs()
		if err != nil {
			return err
		}
		printList(keypairs, 0, 0, 0, []string{})
		return nil
	})

	type KeypairCreateOptions struct {
		NAME      string
		PublicKey string
		Type      string `help:"keypair type" choices:"ssh|x509"`
	}

	shellutils.R(&KeypairCreateOptions{}, "keypair-create", "Create keypair", func(cli *openstack.SRegion, args *KeypairCreateOptions) error {
		keypair, err := cli.CreateKeypair(args.NAME, args.PublicKey, args.Type)
		if err != nil {
			return err
		}
		printObject(keypair)
		return nil
	})

	type KeypairOptions struct {
		NAME string `help:"Keypair name"`
	}

	shellutils.R(&KeypairOptions{}, "keypair-show", "Show keypair", func(cli *openstack.SRegion, args *KeypairOptions) error {
		keypair, err := cli.GetKeypair(args.NAME)
		if err != nil {
			return err
		}
		printObject(keypair)
		return nil
	})

	shellutils.R(&KeypairOptions{}, "keypair-delete", "Delete keypair", func(cli *openstack.SRegion, args *KeypairOptions) error {
		return cli.DeleteKeypair(args.NAME)
	})

}
