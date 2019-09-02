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

package ceph

import (
	"context"
	"strconv"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/multicloud/objectstore"
)

type SCephRadosBucket struct {
	*objectstore.SBucket
}

func (b *SCephRadosBucket) GetStats() cloudprovider.SBucketStats {
	_, hdr, _ := b.GetIBucketProvider().S3Client().BucketExists(b.Name)
	if hdr != nil {
		sizeBytesStr := hdr.Get("X-Rgw-Bytes-Used")
		sizeBytes, _ := strconv.ParseInt(sizeBytesStr, 10, 64)
		objCntStr := hdr.Get("X-Rgw-Object-Count")
		objCnt, _ := strconv.ParseInt(objCntStr, 10, 64)
		return cloudprovider.SBucketStats{
			SizeBytes:   sizeBytes,
			ObjectCount: int(objCnt),
		}
	}
	return b.SBucket.GetStats()
}

func (b *SCephRadosBucket) GetLimit() cloudprovider.SBucketStats {
	if cephCli, ok := b.GetIBucketProvider().(*SCephRadosClient); ok {
		quota, err := cephCli.adminApi.GetBucketQuota(context.Background(), cephCli.GetAccountId(), b.Name)
		if err == nil {
			limit := cloudprovider.SBucketStats{}
			if quota.Enabled.IsTrue() {
				limit.SizeBytes = quota.MaxSize
				limit.ObjectCount = quota.MaxObjects
			}
			return limit
		}
	}
	return b.SBucket.GetLimit()
}

func (b *SCephRadosBucket) SetLimit(limit cloudprovider.SBucketStats) error {
	/*if cephCli, ok := b.GetIBucketProvider().(*SCephRadosClient); ok {
		err := cephCli.adminApi.SetBucketQuota(context.Background(), cephCli.GetAccountId(), b.Name, limit.SizeBytes, limit.ObjectCount)
		return errors.Wrap(err, "cephCli.adminApi.SetBucketQuota")
	}
	return b.SBucket.SetLimit(limit)*/
	return httperrors.ErrNotSupported
}
