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
	"yunion.io/x/onecloud/pkg/multicloud/zstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type DiskOfferingOptions struct {
		SizeGb int
	}
	shellutils.R(&DiskOfferingOptions{}, "disk-offering-list", "List disk offerings", func(cli *zstack.SRegion, args *DiskOfferingOptions) error {
		offerings, err := cli.GetDiskOfferings(args.SizeGb)
		if err != nil {
			return err
		}
		printList(offerings, len(offerings), 0, 0, []string{})
		return nil
	})

	type DiskOfferingCreateOptions struct {
		SIZE_GB int
	}

	shellutils.R(&DiskOfferingCreateOptions{}, "disk-offering-create", "Create disk offering", func(cli *zstack.SRegion, args *DiskOfferingCreateOptions) error {
		offering, err := cli.CreateDiskOffering(args.SIZE_GB)
		if err != nil {
			return err
		}
		printObject(offering)
		return nil
	})

	type DiskOfferingDeleteOptions struct {
		ID string
	}

	shellutils.R(&DiskOfferingDeleteOptions{}, "disk-offering-delete", "Delete disk offering", func(cli *zstack.SRegion, args *DiskOfferingDeleteOptions) error {
		return cli.DeleteDiskOffering(args.ID)
	})

}
