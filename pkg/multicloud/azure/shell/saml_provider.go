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

	"yunion.io/x/onecloud/pkg/multicloud/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type SAMLProviderListOptions struct {
	}
	shellutils.R(&SAMLProviderListOptions{}, "saml-provider-list", "List regions", func(cli *azure.SRegion, args *SAMLProviderListOptions) error {
		sps, err := cli.GetClient().ListSAMLProviders()
		if err != nil {
			return err
		}
		printList(sps, 0, 0, 0, nil)
		return nil
	})

	type SInvitateUser struct {
		EMAIL string
	}

	shellutils.R(&SInvitateUser{}, "invite-user", "Invitate user", func(cli *azure.SRegion, args *SInvitateUser) error {
		user, err := cli.GetClient().InviteUser(args.EMAIL)
		if err != nil {
			return err
		}
		printObject(user)
		fmt.Println("invite url: ", user.GetInviteUrl())
		return nil
	})

}
