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
	"time"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/multicloud/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type MongoDBListOptions struct {
		MongoType string `choices:"sharding|replicate|serverless"`
		Offset    int
		Limit     int
	}
	shellutils.R(&MongoDBListOptions{}, "mongodb-list", "List mongodb", func(cli *aliyun.SRegion, args *MongoDBListOptions) error {
		dbs, _, err := cli.GetMongoDBs(args.MongoType, args.Limit, args.Offset)
		if err != nil {
			return err
		}
		printList(dbs, 0, 0, 0, []string{})
		return nil
	})

	type MongoDBIdOptions struct {
		ID string
	}

	shellutils.R(&MongoDBIdOptions{}, "mongodb-show", "Show mongodb", func(cli *aliyun.SRegion, args *MongoDBIdOptions) error {
		db, err := cli.GetMongoDB(args.ID)
		if err != nil {
			return errors.Wrapf(err, "GetMongoDB(%s)", args.ID)
		}
		printObject(db)
		return nil

	})

	shellutils.R(&MongoDBIdOptions{}, "mongodb-delete", "Delete mongodb", func(cli *aliyun.SRegion, args *MongoDBIdOptions) error {
		return cli.DeleteMongoDB(args.ID)
	})

	type MongoDBBackupListOptions struct {
		ID         string
		START      time.Time
		END        time.Time
		PageSize   int
		PageNumber int
	}

	shellutils.R(&MongoDBBackupListOptions{}, "mongodb-backup-list", "List mongodb backups", func(cli *aliyun.SRegion, args *MongoDBBackupListOptions) error {
		backups, _, err := cli.GetMongoDBBackups(args.ID, args.START, args.END, args.PageSize, args.PageNumber)
		if err != nil {
			return err
		}
		printList(backups, 0, 0, 0, nil)
		return nil
	})

	type MongoDBSkuListOptions struct {
	}

	shellutils.R(&MongoDBSkuListOptions{}, "mongodb-sku-list", "List mongodb skus", func(cli *aliyun.SRegion, args *MongoDBSkuListOptions) error {
		skus, err := cli.GetchMongoSkus()
		if err != nil {
			return err
		}
		printObject(skus)
		return nil
	})

}
