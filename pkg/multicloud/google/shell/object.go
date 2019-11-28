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
	"os"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/google"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ObjectPutOptions struct {
		BUCKET      string
		FILE        string
		ContentType string
		Acl         string `choices:"private|public-read|public-read-write|authenticated-read"`
	}

	shellutils.R(&ObjectPutOptions{}, "object-put", "Put object to buckets", func(cli *google.SRegion, args *ObjectPutOptions) error {
		file, err := os.Open(args.FILE)
		if err != nil {
			return errors.Wrap(err, "so.Open")
		}
		stat, err := file.Stat()
		if err != nil {
			return errors.Wrap(err, "file.Stat")
		}
		return cli.PutObject(args.BUCKET, args.FILE, file, args.ContentType, stat.Size(), cloudprovider.TBucketACLType(args.Acl))
	})

}
