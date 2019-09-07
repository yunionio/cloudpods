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
	"fmt"
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type CredentialListOptions struct {
		Scope      string `help:"scope" choices:"project|domain|system"`
		Type       string `help:"credential type" choices:"totp|recovery|aksk"`
		User       string `help:"filter by user"`
		UserDomain string `help:"the domain of user"`
	}
	R(&CredentialListOptions{}, "credential-list", "List all credentials", func(s *mcclient.ClientSession, args *CredentialListOptions) error {
		query := jsonutils.NewDict()
		if len(args.Type) > 0 {
			query.Add(jsonutils.NewString(args.Type), "type")
		}
		if len(args.Scope) > 0 {
			query.Add(jsonutils.NewString(args.Scope), "scope")
		}
		if len(args.User) > 0 {
			if len(args.UserDomain) > 0 {
				query.Add(jsonutils.NewString(args.UserDomain), "domain_id")
			}
			query.Add(jsonutils.NewString(args.User), "user_id")
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

	type CredentialAkSkOptions struct {
		User          string `help:"User"`
		UserDomain    string `help:"domain of user"`
		Project       string `help:"Project"`
		ProjectDomain string `help:"domain of user"`
	}
	R(&CredentialAkSkOptions{}, "credential-create-aksk", "Create AccessKey/Secret credential", func(s *mcclient.ClientSession, args *CredentialAkSkOptions) error {
		var uid string
		var pid string
		var err error
		if len(args.User) > 0 {
			uid, err = modules.UsersV3.FetchId(s, args.User, args.UserDomain)
			if err != nil {
				return err
			}
		}
		if len(args.Project) > 0 {
			pid, err = modules.Projects.FetchId(s, args.Project, args.ProjectDomain)
			if err != nil {
				return err
			}
		}
		secret, err := modules.Credentials.CreateAccessKeySecret(s, uid, pid, time.Time{})
		if err != nil {
			return err
		}
		printObject(jsonutils.Marshal(&secret))
		return nil
	})

	R(&CredentialAkSkOptions{}, "credential-get-aksk", "Get AccessKey/Secret credential for user and project", func(s *mcclient.ClientSession, args *CredentialAkSkOptions) error {
		var uid string
		var err error
		if len(args.User) > 0 {
			uid, err = modules.UsersV3.FetchId(s, args.User, args.UserDomain)
			if err != nil {
				return err
			}
		}
		var pid string
		if len(args.Project) > 0 {
			pid, err = modules.Projects.FetchId(s, args.Project, args.ProjectDomain)
			if err != nil {
				return err
			}
		}
		secrets, err := modules.Credentials.GetAccessKeySecrets(s, uid, pid)
		if err != nil {
			return err
		}
		result := modulebase.ListResult{}
		result.Data = make([]jsonutils.JSONObject, len(secrets))
		for i := range secrets {
			result.Data[i] = jsonutils.Marshal(secrets[i])
			result.Data[i].(*jsonutils.JSONDict).Add(jsonutils.NewString(secrets[i].KeyId), "key_id")
			result.Data[i].(*jsonutils.JSONDict).Add(jsonutils.NewString(secrets[i].ProjectId), "project_id")
			result.Data[i].(*jsonutils.JSONDict).Add(jsonutils.NewTimeString(secrets[i].TimeStamp), "time_stamp")
		}
		printList(&result, nil)
		return nil
	})

	R(&CredentialAkSkOptions{}, "credential-remove-aksk", "Remove AccessKey/Secret credential for user and project", func(s *mcclient.ClientSession, args *CredentialAkSkOptions) error {
		uid, err := modules.UsersV3.FetchId(s, args.User, args.UserDomain)
		if err != nil {
			return err
		}
		var pid string
		if len(args.Project) > 0 {
			pid, err = modules.Projects.FetchId(s, args.Project, args.ProjectDomain)
			if err != nil {
				return err
			}
		}
		err = modules.Credentials.RemoveAccessKeySecrets(s, uid, pid)
		if err != nil {
			return err
		}
		fmt.Println("success")
		return nil
	})

	type CredentialDeleteOptions struct {
		ID string `help:"ID of credentail"`
	}
	R(&CredentialDeleteOptions{}, "credential-delete", "Delete credential", func(s *mcclient.ClientSession, args *CredentialDeleteOptions) error {
		result, err := modules.Credentials.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CredentialUpdateOptions struct {
		ID      string `help:"ID of credentail"`
		Enable  bool   `help:"Enable credential"`
		Disable bool   `help:"Disable credential"`
		Name    string `help:"new name of credential"`
		Desc    string `help:"new description of credential"`
	}
	R(&CredentialUpdateOptions{}, "credential-update", "Enable/disable credential", func(s *mcclient.ClientSession, args *CredentialUpdateOptions) error {
		params := jsonutils.NewDict()
		if args.Enable {
			params.Add(jsonutils.JSONTrue, "enabled")
		} else if args.Disable {
			params.Add(jsonutils.JSONFalse, "enabled")
		}
		if args.Name != "" {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if args.Desc != "" {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		result, err := modules.Credentials.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
