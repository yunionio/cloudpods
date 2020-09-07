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
	"yunion.io/x/onecloud/pkg/multicloud/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type SAMLProviderListOptions struct {
	}
	shellutils.R(&SAMLProviderListOptions{}, "saml-provider-list", "List saml providers", func(cli *aws.SRegion, args *SAMLProviderListOptions) error {
		samls, err := cli.GetClient().ListSAMLProviders()
		if err != nil {
			return err
		}
		printList(samls, 0, 0, 0, []string{})
		return nil
	})

	type SAMLProviderArnOptions struct {
		ARN string
	}

	shellutils.R(&SAMLProviderArnOptions{}, "saml-provider-show", "Show saml provider", func(cli *aws.SRegion, args *SAMLProviderArnOptions) error {
		saml, err := cli.GetClient().GetSAMLProvider(args.ARN)
		if err != nil {
			return err
		}
		printObject(saml)
		return nil
	})

	shellutils.R(&SAMLProviderArnOptions{}, "saml-provider-delete", "Delete saml provider", func(cli *aws.SRegion, args *SAMLProviderArnOptions) error {
		return cli.GetClient().DeleteSAMLProvider(args.ARN)
	})

	type SAMLProviderCreateOptions struct {
		NAME     string
		METADATA string
	}

	shellutils.R(&SAMLProviderCreateOptions{}, "saml-provider-create", "Create saml provider", func(cli *aws.SRegion, args *SAMLProviderCreateOptions) error {
		saml, err := cli.GetClient().CreateSAMLProvider(args.NAME, args.METADATA)
		if err != nil {
			return err
		}
		printObject(saml)
		return nil
	})

	type SAMLProviderUpdateOptions struct {
		ARN      string
		METADATA string
	}

	shellutils.R(&SAMLProviderUpdateOptions{}, "saml-provider-update", "Update saml provider", func(cli *aws.SRegion, args *SAMLProviderUpdateOptions) error {
		saml, err := cli.GetClient().UpdateSAMLProvider(args.ARN, args.METADATA)
		if err != nil {
			return err
		}
		printObject(saml)
		return nil
	})

}
