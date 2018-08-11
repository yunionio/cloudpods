package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {

	/**
	 * 通信地址验证
	 */
	type ContactsVerifyOptions struct {
		ID    string `help:"Verification process ID"`
		TOKEN string `help:"Temporary token issued to the user when the validation is triggered"`
	}
	R(&ContactsVerifyOptions{}, "contact-verify", "Trigger contact verify", func(s *mcclient.ClientSession, args *ContactsVerifyOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.TOKEN), "token")

		result, err := modules.Verifications.Get(s, args.ID, params)

		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

}
