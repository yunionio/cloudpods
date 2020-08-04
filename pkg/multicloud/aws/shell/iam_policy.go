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
	type PolicyListOptions struct {
		Limit             int
		Offset            string
		OnlyAttached      bool
		PathPrefix        string
		PolicyUsageFilter string `choices:"PermissionsPolicy|PermissionsBoundary"`
		Scope             string `choices:"All|AWS|Local"`
	}

	shellutils.R(&PolicyListOptions{}, "cloud-policy-list", "List policies", func(cli *aws.SRegion, args *PolicyListOptions) error {
		policies, err := cli.GetClient().ListPolicies(args.Offset, args.Limit, args.OnlyAttached, args.PathPrefix, args.PolicyUsageFilter, args.Scope)
		if err != nil {
			return err
		}
		printList(policies.Policies, 0, 0, 0, nil)
		return nil
	})

	type PolicyVersionListOptions struct {
		Offset string
		Limit  int
		ARN    string
	}
	shellutils.R(&PolicyVersionListOptions{}, "cloud-policy-version-list", "List policy versions", func(cli *aws.SRegion, args *PolicyVersionListOptions) error {
		versions, err := cli.GetClient().ListPolicyVersions(args.Offset, args.Limit, args.ARN)
		if err != nil {
			return err
		}
		printList(versions.Versions, 0, 0, 0, nil)
		return nil
	})

	type PolicyVersionOptions struct {
		ARN     string
		VERSION string
	}

	shellutils.R(&PolicyVersionOptions{}, "cloud-policy-version-show", "Show policy version details", func(cli *aws.SRegion, args *PolicyVersionOptions) error {
		version, err := cli.GetClient().GetPolicyVersion(args.ARN, args.VERSION)
		if err != nil {
			return err
		}
		printObject(version)
		return nil
	})

	type PolicyArnOptions struct {
		ARN string
	}

	shellutils.R(&PolicyArnOptions{}, "cloud-policy-show", "Show policy details by policy arn", func(cli *aws.SRegion, args *PolicyArnOptions) error {
		policy, err := cli.GetClient().GetPolicy(args.ARN)
		if err != nil {
			return err
		}
		printObject(policy)
		return nil
	})

	shellutils.R(&PolicyArnOptions{}, "cloud-policy-delete", "Delete policy", func(cli *aws.SRegion, args *PolicyArnOptions) error {
		return cli.GetClient().DeletePolicy(args.ARN)
	})

	type PolicyVersionSHowOptions struct {
		ARN     string
		VERSION string
	}

	shellutils.R(&PolicyVersionSHowOptions{}, "cloud-policy-version-show", "Show policy version details", func(cli *aws.SRegion, args *PolicyVersionSHowOptions) error {
		version, err := cli.GetClient().GetPolicyVersion(args.ARN, args.VERSION)
		if err != nil {
			return err
		}
		printObject(version)
		return nil
	})

	type PolicyCreateOptions struct {
		NAME     string
		DOCUMENT string
		Path     string
		Desc     string
	}

	shellutils.R(&PolicyCreateOptions{}, "cloud-policy-create", "Create policy", func(cli *aws.SRegion, args *PolicyCreateOptions) error {
		policy, err := cli.GetClient().CreatePolicy(args.NAME, args.DOCUMENT, args.Path, args.Desc)
		if err != nil {
			return err
		}
		printObject(policy)
		return nil
	})

	type PolicyVersionCreateOptions struct {
		ARN       string
		DOCUMENT  string
		IsDefault bool
	}

	shellutils.R(&PolicyVersionCreateOptions{}, "cloud-policy-version-create", "Create policy version", func(cli *aws.SRegion, args *PolicyVersionCreateOptions) error {
		return cli.GetClient().CreatePolicyVersion(args.ARN, args.DOCUMENT, args.IsDefault)
	})

}
