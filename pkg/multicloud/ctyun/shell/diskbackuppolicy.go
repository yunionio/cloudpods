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
	"yunion.io/x/onecloud/pkg/multicloud/ctyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type VPolicyListOptions struct {
	}
	shellutils.R(&VPolicyListOptions{}, "policy-list", "List polices", func(cli *ctyun.SRegion, args *VPolicyListOptions) error {
		polices, e := cli.GetDiskBackupPolices()
		if e != nil {
			return e
		}
		printList(polices, 0, 0, 0, nil)
		return nil
	})

	type PolicyCreateOptions struct {
		Name          string `help:"policy name"`
		StartTime     string `help:"startTime"`
		Frequency     string `help:"frequency"`
		RententionNum string `help:"rententionNum"`
		FirstBackup   string `help:"firstBackup"`
		Status        string `help:"status"`
	}
	shellutils.R(&PolicyCreateOptions{}, "policy-create", "Create policy", func(cli *ctyun.SRegion, args *PolicyCreateOptions) error {
		e := cli.CreateDiskBackupPolicy(args.Name, args.StartTime, args.Frequency, args.RententionNum, args.FirstBackup, args.Status)
		if e != nil {
			return e
		}

		return nil
	})

	type BindingDiskBackupPolicyOptions struct {
		PolicyId     string `help:"policy id"`
		ResourceId   string `help:"resource id"`
		ResourceType string `help:"resource type"`
	}
	shellutils.R(&BindingDiskBackupPolicyOptions{}, "policy-bind", "Binding policy", func(cli *ctyun.SRegion, args *BindingDiskBackupPolicyOptions) error {
		e := cli.BindingDiskBackupPolicy(args.PolicyId, args.ResourceId, args.ResourceType)
		if e != nil {
			return e
		}

		return nil
	})

	type UnBindDiskBackupPolicyOptions struct {
		PolicyId   string `help:"policy id"`
		ResourceId string `help:"resourceId"`
	}
	shellutils.R(&UnBindDiskBackupPolicyOptions{}, "policy-unbind", "Unbind policy", func(cli *ctyun.SRegion, args *UnBindDiskBackupPolicyOptions) error {
		e := cli.UnBindDiskBackupPolicy(args.PolicyId, args.ResourceId)
		if e != nil {
			return e
		}

		return nil
	})
}
