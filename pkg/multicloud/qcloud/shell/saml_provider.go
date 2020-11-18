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
	"yunion.io/x/onecloud/pkg/multicloud/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type SAMLProviderListOptions struct {
	}
	shellutils.R(&SAMLProviderListOptions{}, "saml-provider-list", "List saml provider", func(cli *qcloud.SRegion, args *SAMLProviderListOptions) error {
		result, err := cli.GetClient().ListSAMLProviders()
		if err != nil {
			return err
		}
		printList(result, 0, 0, 0, nil)
		return nil
	})

	type SAMLProviderCreateOptions struct {
		NAME     string
		Desc     string
		METADATA string
	}

	shellutils.R(&SAMLProviderCreateOptions{}, "saml-provider-create", "Create saml provider", func(cli *qcloud.SRegion, args *SAMLProviderCreateOptions) error {
		result, err := cli.GetClient().CreateSAMLProvider(args.NAME, args.METADATA, args.Desc)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type SAMLProviderNameOptions struct {
		NAME string
	}

	shellutils.R(&SAMLProviderNameOptions{}, "saml-provider-show", "Show saml provider", func(cli *qcloud.SRegion, args *SAMLProviderNameOptions) error {
		result, err := cli.GetClient().GetSAMLProvider(args.NAME)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	shellutils.R(&SAMLProviderNameOptions{}, "saml-provider-delete", "Delete saml provider", func(cli *qcloud.SRegion, args *SAMLProviderNameOptions) error {
		return cli.GetClient().DeleteSAMLProvider(args.NAME)
	})

	type SAMLProviderUpdateOptions struct {
		NAME     string
		Desc     string
		Metadata string
	}

	shellutils.R(&SAMLProviderUpdateOptions{}, "saml-provider-update", "Update saml provider", func(cli *qcloud.SRegion, args *SAMLProviderUpdateOptions) error {
		return cli.GetClient().UpdateSAMLProvider(args.NAME, args.Metadata, args.Desc)
	})

}
