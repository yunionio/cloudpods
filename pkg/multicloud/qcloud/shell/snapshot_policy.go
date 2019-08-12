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
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type SSnapshotPolicyListOptions struct {
		PolicyId string `help:"snapshot policy id"`
		Offset   int    `help:"offset"`
		Limit    int    `help:"limit"`
	}

	shellutils.R(&SSnapshotPolicyListOptions{}, "snapshot-policy-list", "list snapshot policy",
		func(cli *qcloud.SRegion, args *SSnapshotPolicyListOptions) error {
			snapshotPolicis, num, err := cli.GetSnapshotPolicies(args.PolicyId, args.Offset, args.Limit)
			if err != nil {
				return err
			}

			printList(snapshotPolicis, num, args.Offset, args.Limit, []string{})
			return nil
		},
	)

	type SSnapshotPolicyDeleteOptions struct {
		ID string `help:"snapshot id"`
	}
	shellutils.R(&SSnapshotPolicyDeleteOptions{}, "snapshot-policy-delete", "delete snapshot policy",
		func(cli *qcloud.SRegion, args *SSnapshotPolicyDeleteOptions) error {
			err := cli.DeleteSnapshotPolicy(args.ID)
			return err
		},
	)

	type SSnapshotPolicyCreateOptions struct {
		Name string `help:"snapshot name"`

		RetentionDays  int   `help:"retention days"`
		RepeatWeekdays []int `help:"auto snapshot which days of the week"`
		TimePoints     []int `help:"auto snapshot which hours of the day"`
	}
	shellutils.R(&SSnapshotPolicyCreateOptions{}, "snapshot-policy-create", "create snapshot policy",
		func(cli *qcloud.SRegion, args *SSnapshotPolicyCreateOptions) error {
			input := cloudprovider.SnapshotPolicyInput{
				RetentionDays:  args.RetentionDays,
				RepeatWeekdays: args.RepeatWeekdays,
				TimePoints:     args.TimePoints,
				PolicyName:     args.Name,
			}
			_, err := cli.CreateSnapshotPolicy(&input)
			if err != nil {
				return err
			}
			return nil
		},
	)

	type SSnapshotPolicyApplyOptions struct {
		SNAPSHOTPOLICYID string `help:"snapshot policy id"`
		DISKID           string `help:"disk id"`
	}
	shellutils.R(&SSnapshotPolicyApplyOptions{}, "snapshot-policy-apply", "apply snapshot policy",
		func(cli *qcloud.SRegion, args *SSnapshotPolicyApplyOptions) error {
			err := cli.ApplySnapshotPolicyToDisks(args.SNAPSHOTPOLICYID, args.DISKID)
			return err
		},
	)

	type SSnapshotPolicyCancelOptions struct {
		SNAPSHOTPOLICYID string `help:"snapshot policy id"`
		DISKID           string `help:"disk id"`
	}
	shellutils.R(&SSnapshotPolicyCancelOptions{}, "snapshot-policy-cancel", "cancel snapshot policy",
		func(cli *qcloud.SRegion, args *SSnapshotPolicyCancelOptions) error {
			err := cli.CancelSnapshotPolicyToDisks(args.SNAPSHOTPOLICYID, args.DISKID)
			return err
		},
	)
}
