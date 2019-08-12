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
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type SSnapshotpolicyDiskListOfDiskOption struct {
		DISKID string `help:"disk id"`
	}
	R(&SSnapshotpolicyDiskListOfDiskOption{}, "disk-snapshot-policy-list", "List snapshot policy attached",
		func(s *mcclient.ClientSession, opts *SSnapshotpolicyDiskListOfDiskOption) error {
			result, err := modules.SnapshotPolicyDisk.ListDescendent(s, opts.DISKID, jsonutils.NewDict())
			if err != nil {
				return err
			}
			printList(result, modules.SnapshotPolicyDisk.GetColumns(s))
			return nil
		},
	)

	type SSnapshotpolicyDiskListOfSnapshotpolicyOption struct {
		SNAPSHOTPOLICYID string `help:"snapshot policy id"`
	}
	R(&SSnapshotpolicyDiskListOfSnapshotpolicyOption{}, "snapshot-policy-disk-list", "List disk attached",
		func(s *mcclient.ClientSession, opts *SSnapshotpolicyDiskListOfSnapshotpolicyOption) error {
			result, err := modules.SnapshotPolicyDisk.ListDescendent2(s, opts.SNAPSHOTPOLICYID, jsonutils.NewDict())
			if err != nil {
				return err
			}
			printList(result, modules.SnapshotPolicyDisk.GetColumns(s))
			return nil
		},
	)

	type SSnapshotpolicyDiskAttachOption struct {
		DISKID           string `help:"disk id"`
		SNAPSHOTPOLICYID string `help:"snapshot policy id"`
	}
	R(&SSnapshotpolicyDiskAttachOption{}, "disk-snapshot-policy-attach", "attach snapshot policy to disk",
		func(s *mcclient.ClientSession, opts *SSnapshotpolicyDiskAttachOption) error {
			_, err := modules.SnapshotPolicyDisk.Attach(s, opts.DISKID, opts.SNAPSHOTPOLICYID, jsonutils.NewDict())
			if err != nil {
				return err
			}
			return nil
		},
	)

	type SSnapshotpolicyDiskDetachOption struct {
		DISKID           string `help:"disk id"`
		SNAPSHOTPOLICYID string `help:"snapshot policy id"`
	}
	R(&SSnapshotpolicyDiskDetachOption{}, "disk-snapshot-policy-detach", "detach snapshot policy to disk",
		func(s *mcclient.ClientSession, opts *SSnapshotpolicyDiskDetachOption) error {
			_, err := modules.SnapshotPolicyDisk.Detach(s, opts.DISKID, opts.SNAPSHOTPOLICYID, jsonutils.NewDict())
			if err != nil {
				return err
			}
			return nil
		},
	)

}
