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
	type ClouduserListOptions struct {
		Marker     string
		MaxItems   int
		PathPrefix string
	}
	shellutils.R(&ClouduserListOptions{}, "cloud-user-list", "List cloudusers", func(cli *aws.SRegion, args *ClouduserListOptions) error {
		users, err := cli.GetClient().GetCloudusers(args.Marker, args.MaxItems, args.PathPrefix)
		if err != nil {
			return err
		}
		printList(users.Users, 0, 0, 0, nil)
		return nil
	})

	type ClouduserCreateOptions struct {
		NAME string
		Path string
	}

	shellutils.R(&ClouduserCreateOptions{}, "cloud-user-create", "Create clouduser", func(cli *aws.SRegion, args *ClouduserCreateOptions) error {
		user, err := cli.GetClient().CreateClouduser(args.Path, args.NAME)
		if err != nil {
			return err
		}
		printObject(user)
		return nil
	})

	type ClouduserOptions struct {
		NAME string
	}

	shellutils.R(&ClouduserOptions{}, "cloud-user-show", "Show clouduser details", func(cli *aws.SRegion, args *ClouduserOptions) error {
		user, err := cli.GetClient().GetClouduser(args.NAME)
		if err != nil {
			return err
		}
		printObject(user)
		return nil
	})

	shellutils.R(&ClouduserOptions{}, "cloud-user-loginprofile", "Show clouduser loginprofile", func(cli *aws.SRegion, args *ClouduserOptions) error {
		profile, err := cli.GetClient().GetLoginProfile(args.NAME)
		if err != nil {
			return err
		}
		printObject(profile)
		return nil
	})

	shellutils.R(&ClouduserOptions{}, "cloud-user-delete-loginprofile", "Delete clouduser loginprofile", func(cli *aws.SRegion, args *ClouduserOptions) error {
		return cli.GetClient().DeleteLoginProfile(args.NAME)
	})

	shellutils.R(&ClouduserOptions{}, "cloud-user-delete", "Delete clouduser", func(cli *aws.SRegion, args *ClouduserOptions) error {
		return cli.GetClient().DeleteClouduser(args.NAME)
	})

	type ClouduserLoginProfileCreateOptions struct {
		NAME     string
		PASSWORD string
	}

	shellutils.R(&ClouduserLoginProfileCreateOptions{}, "cloud-user-create-loginprofile", "Create clouduser loginprofile", func(cli *aws.SRegion, args *ClouduserLoginProfileCreateOptions) error {
		profile, err := cli.GetClient().CreateLoginProfile(args.NAME, args.PASSWORD)
		if err != nil {
			return err
		}
		printObject(profile)
		return nil
	})

	shellutils.R(&ClouduserLoginProfileCreateOptions{}, "cloud-user-reset-password", "Reset clouduser password", func(cli *aws.SRegion, args *ClouduserLoginProfileCreateOptions) error {
		return cli.GetClient().ResetClouduserPassword(args.NAME, args.PASSWORD)
	})

	type ClouduserPolicyListOptions struct {
		NAME     string
		Marker   string
		MaxItems int
	}

	shellutils.R(&ClouduserPolicyListOptions{}, "cloud-user-policy-list", "List clouduser policies", func(cli *aws.SRegion, args *ClouduserPolicyListOptions) error {
		policies, err := cli.GetClient().ListUserpolicies(args.NAME, args.Marker, args.MaxItems)
		if err != nil {
			return err
		}
		printObject(policies)
		return nil
	})

	type ClouduserAttachedPolicyListOptions struct {
		NAME       string
		Marker     string
		MaxItems   int
		PathPrefix string
	}

	shellutils.R(&ClouduserAttachedPolicyListOptions{}, "cloud-user-attached-policy-list", "List clouduser attached policies", func(cli *aws.SRegion, args *ClouduserAttachedPolicyListOptions) error {
		policies, err := cli.GetClient().ListAttachedUserPolicies(args.NAME, args.Marker, args.MaxItems, args.PathPrefix)
		if err != nil {
			return err
		}
		printList(policies.AttachedPolicies, 0, 0, 0, nil)
		return nil
	})

	type PolicyListOptions struct {
		MaxItems          int
		Marker            string
		OnlyAttached      bool
		PathPrefix        string
		PolicyUsageFilter string `choices:"PermissionsPolicy|PermissionsBoundary"`
		Scope             string `choices:"All|AWS|Local"`
	}

	shellutils.R(&PolicyListOptions{}, "cloud-policy-list", "List policies", func(cli *aws.SRegion, args *PolicyListOptions) error {
		policies, err := cli.GetClient().ListPolicies(args.Marker, args.MaxItems, args.OnlyAttached, args.PathPrefix, args.PolicyUsageFilter, args.Scope)
		if err != nil {
			return err
		}
		printList(policies.Policies, 0, 0, 0, nil)
		return nil
	})

	type ClouduserPolicyOptions struct {
		NAME       string
		POLICY_ARN string
	}

	shellutils.R(&ClouduserPolicyOptions{}, "cloud-user-attach-policy", "Attach policy for clouduser", func(cli *aws.SRegion, args *ClouduserPolicyOptions) error {
		return cli.GetClient().AttachPolicy(args.NAME, args.POLICY_ARN)
	})

	shellutils.R(&ClouduserPolicyOptions{}, "cloud-user-detach-policy", "Detach policy from clouduser", func(cli *aws.SRegion, args *ClouduserPolicyOptions) error {
		return cli.GetClient().DetachPolicy(args.NAME, args.POLICY_ARN)
	})

}
