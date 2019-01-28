package shell

import (
	"yunion.io/x/onecloud/pkg/util/openstack"
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
