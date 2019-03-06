package shell

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type CredentialListOptions struct {
		Type       string `help:"credential type" choices:"totp|recovery|ec2"`
		User       string `help:"filter by user"`
		UserDomain string `help:"the domain of user"`
	}
	R(&CredentialListOptions{}, "credential-list", "List all credentials", func(s *mcclient.ClientSession, args *CredentialListOptions) error {
		query := jsonutils.NewDict()
		if len(args.Type) > 0 {
			query.Add(jsonutils.NewString(args.Type), "type")
		}
		var err error
		if len(args.User) > 0 {
			domainId := "default"
			if len(args.UserDomain) > 0 {
				domainId, err = modules.Domains.GetId(s, args.UserDomain, nil)
				if err != nil {
					return err
				}
			}
			userQuery := jsonutils.NewDict()
			userQuery.Add(jsonutils.NewString(domainId), "domain_id")
			userId, err := modules.UsersV3.GetId(s, args.User, userQuery)
			if err != nil {
				return err
			}
			query.Add(jsonutils.NewString(userId), "user_id")
		}
		results, err := modules.Credentials.List(s, query)
		if err != nil {
			return err
		}
		printList(results, nil)
		return nil
	})

	type CredentialTOTPOptions struct {
		USER       string `help:"User"`
		UserDomain string `help:"domain of user"`
	}
	R(&CredentialTOTPOptions{}, "credential-create-totp", "Create totp credential", func(s *mcclient.ClientSession, args *CredentialTOTPOptions) error {
		uid, err := modules.UsersV3.FetchId(s, args.USER, args.UserDomain)
		if err != nil {
			return err
		}
		secret, err := modules.Credentials.CreateTotpSecret(s, uid)
		if err != nil {
			return err
		}
		fmt.Println("secret:", secret)
		return nil
	})

	R(&CredentialTOTPOptions{}, "credential-get-totp", "Get totp credential for user", func(s *mcclient.ClientSession, args *CredentialTOTPOptions) error {
		uid, err := modules.UsersV3.FetchId(s, args.USER, args.UserDomain)
		if err != nil {
			return err
		}
		secret, err := modules.Credentials.GetTotpSecret(s, uid)
		if err != nil {
			return err
		}
		fmt.Println("secret:", secret)
		return nil
	})

	R(&CredentialTOTPOptions{}, "credential-remove-totp", "Remove totp credential for user", func(s *mcclient.ClientSession, args *CredentialTOTPOptions) error {
		uid, err := modules.UsersV3.FetchId(s, args.USER, args.UserDomain)
		if err != nil {
			return err
		}
		err = modules.Credentials.RemoveTotpSecrets(s, uid)
		if err != nil {
			return err
		}
		fmt.Println("success")
		return nil
	})

	type CredentialCreateRecoverySecretsOptions struct {
		USER       string   `help:"User"`
		UserDomain string   `help:"domain of user"`
		Question   []string `help:"questions"`
		Answer     []string `help:"answers"`
	}
	R(&CredentialCreateRecoverySecretsOptions{}, "credential-save-recovery-secrets", "save recovery secrets for user", func(s *mcclient.ClientSession, args *CredentialCreateRecoverySecretsOptions) error {
		uid, err := modules.UsersV3.FetchId(s, args.USER, args.UserDomain)
		if err != nil {
			return err
		}
		if len(args.Question) == 0 || len(args.Answer) == 0 {
			return fmt.Errorf("no recovery secrets provided")
		}
		if len(args.Question) != len(args.Answer) {
			return fmt.Errorf("number of questions and answers does not match")
		}
		secrets := make([]modules.SRecoverySecret, len(args.Question))
		for i := range args.Question {
			secrets[i] = modules.SRecoverySecret{
				Question: args.Question[i],
				Answer:   args.Answer[i],
			}
		}
		err = modules.Credentials.SaveRecoverySecrets(s, uid, secrets)
		if err != nil {
			return err
		}
		fmt.Println("success")
		return nil
	})

	R(&CredentialTOTPOptions{}, "credential-get-recovery-secrets", "Get totp credential for user", func(s *mcclient.ClientSession, args *CredentialTOTPOptions) error {
		uid, err := modules.UsersV3.FetchId(s, args.USER, args.UserDomain)
		if err != nil {
			return err
		}
		secret, err := modules.Credentials.GetRecoverySecrets(s, uid)
		if err != nil {
			return err
		}
		fmt.Println("secret:", secret)
		return nil
	})

	R(&CredentialTOTPOptions{}, "credential-remove-recovery-secrets", "Remove totp credential for user", func(s *mcclient.ClientSession, args *CredentialTOTPOptions) error {
		uid, err := modules.UsersV3.FetchId(s, args.USER, args.UserDomain)
		if err != nil {
			return err
		}
		err = modules.Credentials.RemoveRecoverySecrets(s, uid)
		if err != nil {
			return err
		}
		fmt.Println("success")
		return nil
	})
}
