package shell

import (
	"yunion.io/x/onecloud/pkg/util/huawei"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	// todo: 需要进一步确认
	type KeyPairListOptions struct {
	}
	shellutils.R(&KeyPairListOptions{}, "keypair-list", "List keypairs", func(cli *huawei.SRegion, args *KeyPairListOptions) error {
		keypairs, total, e := cli.GetKeypairs()
		if e != nil {
			return e
		}
		printList(keypairs, total, 0, 0, []string{})
		return nil
	})

	type KeyPairImportOptions struct {
		NAME   string `help:"Name of new keypair"`
		PUBKEY string `help:"Public key string"`
	}
	shellutils.R(&KeyPairImportOptions{}, "keypair-import", "Import a keypair", func(cli *huawei.SRegion, args *KeyPairImportOptions) error {
		keypair, err := cli.ImportKeypair(args.NAME, args.PUBKEY)
		if err != nil {
			return err
		}
		printObject(keypair)
		return nil
	})
}
