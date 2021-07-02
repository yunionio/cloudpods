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
	type MongoDBListOptions struct {
		Ids    []string
		Offset int
		Limit  int
	}
	shellutils.R(&MongoDBListOptions{}, "mongodb-list", "List mongodb", func(cli *qcloud.SRegion, args *MongoDBListOptions) error {
		dbs, _, err := cli.GetMongoDBs(args.Ids, args.Limit, args.Offset)
		if err != nil {
			return err
		}
		printList(dbs, 0, 0, 0, []string{})
		return nil
	})

	type MongoDBIdOptions struct {
		ID string
	}

	shellutils.R(&MongoDBIdOptions{}, "mongodb-isoloate", "Isolate mongodb", func(cli *qcloud.SRegion, args *MongoDBIdOptions) error {
		return cli.IsolateMongoDB(args.ID)
	})

	shellutils.R(&MongoDBIdOptions{}, "mongodb-offline", "Offlie mongodb", func(cli *qcloud.SRegion, args *MongoDBIdOptions) error {
		return cli.OfflineIsolatedMongoDB(args.ID)
	})

	shellutils.R(&MongoDBIdOptions{}, "mongodb-delete", "Delete mongodb", func(cli *qcloud.SRegion, args *MongoDBIdOptions) error {
		return cli.DeleteMongoDB(args.ID)
	})

}
