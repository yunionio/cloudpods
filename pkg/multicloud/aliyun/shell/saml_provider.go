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
	"yunion.io/x/onecloud/pkg/multicloud/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type SamlProviderListOptions struct {
		Marker   string `help:"Marker"`
		MaxItems int    `help:"Max items"`
	}
	shellutils.R(&SamlProviderListOptions{}, "saml-provider-list", "List saml provider", func(cli *aliyun.SRegion, args *SamlProviderListOptions) error {
		result, _, err := cli.GetClient().ListSAMLProviders(args.Marker, args.MaxItems)
		if err != nil {
			return err
		}
		printList(result, 0, 0, 0, []string{})
		return nil
	})

	type SamlProviderDeleteOptions struct {
		NAME string `help:"SAML Provider Name"`
	}

	shellutils.R(&SamlProviderDeleteOptions{}, "saml-provider-delete", "Delete saml provider", func(cli *aliyun.SRegion, args *SamlProviderDeleteOptions) error {
		return cli.GetClient().DeleteSAMLProvider(args.NAME)
	})

	type SAMLProviderCreateOptions struct {
		NAME    string
		METADAT string
		Desc    string
	}

	shellutils.R(&SAMLProviderCreateOptions{}, "saml-provider-create", "Create saml provider", func(cli *aliyun.SRegion, args *SAMLProviderCreateOptions) error {
		sp, err := cli.GetClient().CreateSAMLProvider(args.NAME, args.METADAT, args.Desc)
		if err != nil {
			return err
		}
		printObject(sp)
		return nil
	})

}
