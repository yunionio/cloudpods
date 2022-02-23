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
	type DBInstanceSnapshotListOptions struct {
		DBInstanceId string
		Id           string
	}

	shellutils.R(&DBInstanceSnapshotListOptions{}, "dbinstance-snapshot-list", "List rds intance snapshots", func(cli *aws.SRegion, args *DBInstanceSnapshotListOptions) error {
		snapshots, err := cli.GetDBInstanceSnapshots(args.DBInstanceId, args.Id)
		if err != nil {
			return err
		}
		printList(snapshots, 0, 0, 0, []string{})
		return nil
	})

}
