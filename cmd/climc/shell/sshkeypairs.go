package shell

import (
	"fmt"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type SshkeypairQueryOptions struct {
		Project string `help:"get keypair for specific project"`
		Admin   bool   `help:"get admin keypair, sysadmin ONLY option"`
	}
	R(&SshkeypairQueryOptions{}, "sshkeypair-show", "Get ssh keypairs", func(s *mcclient.ClientSession, args *SshkeypairQueryOptions) error {
		query := jsonutils.NewDict()
		if args.Admin {
			query.Add(jsonutils.JSONTrue, "admin")
		}
		var keys jsonutils.JSONObject
		if len(args.Project) == 0 {
			listResult, err := modules.Sshkeypairs.List(s, query)
			if err != nil {
				return err
			}
			keys = listResult.Data[0]
		} else {
			result, err := modules.Sshkeypairs.GetById(s, args.Project, query)
			if err != nil {
				return err
			}
			keys = result
		}
		privKey, _ := keys.GetString("private_key")
		pubKey, _ := keys.GetString("public_key")

		fmt.Print(privKey)
		fmt.Print(pubKey)

		return nil
	})
}
