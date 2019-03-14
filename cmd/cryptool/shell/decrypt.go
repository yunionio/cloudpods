package shell

import (
	"fmt"

	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type DecryptSecretOptions struct {
		KEY    string `help:"secret key"`
		SECRET string `help:"secret to descrypt"`
	}
	shellutils.R(&DecryptSecretOptions{}, "decrypt", "Decrypt", func(args *DecryptSecretOptions) error {
		text, err := utils.DescryptAESBase64(args.KEY, args.SECRET)
		if err != nil {
			return err
		}
		fmt.Println(text)
		return nil
	})
}
