package shell

import (
	"yunion.io/x/onecloud/pkg/util/qcloud"
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
