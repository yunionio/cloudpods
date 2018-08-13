package shell

import (
	"yunion.io/x/onecloud/pkg/util/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type KeyPairListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&KeyPairListOptions{}, "keypair-list", "List keypairs", func(cli *aliyun.SRegion, args *KeyPairListOptions) error {
		keypairs, total, e := cli.GetKeypairs("", "", args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(keypairs, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type KeyPairImportOptions struct {
		NAME   string `help:"Name of new keypair"`
		PUBKEY string `help:"Public key string"`
	}
	shellutils.R(&KeyPairImportOptions{}, "keypair-import", "Import a keypair", func(cli *aliyun.SRegion, args *KeyPairImportOptions) error {
		keypair, err := cli.ImportKeypair(args.NAME, args.PUBKEY)
		if err != nil {
			return err
		}
		printObject(keypair)
		return nil
	})
}
