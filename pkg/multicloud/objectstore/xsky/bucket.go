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

package xsky

import (
	"context"
	"strconv"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/objectstore"
)

type SXskyBucket struct {
	*objectstore.SBucket

	client *SXskyClient
}

func (b *SXskyBucket) GetStats() cloudprovider.SBucketStats {
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

func (b *SXskyBucket) LimitSupport() cloudprovider.SBucketStats {
	return cloudprovider.SBucketStats{
		SizeBytes:   1,
		ObjectCount: 1,
	}
}

func (b *SXskyBucket) GetLimit() cloudprovider.SBucketStats {
	limit := cloudprovider.SBucketStats{}
	bucket, err := b.client.adminApi.getBucketByName(context.Background(), b.Name)
	if err != nil {
		log.Errorf("b.client.adminApi.getBucketByName error %s", err)
	} else {
		limit.SizeBytes = bucket.QuotaMaxSize
		limit.ObjectCount = bucket.QuotaMaxObjects
	}
	return limit
}

func (b *SXskyBucket) SetLimit(limit cloudprovider.SBucketStats) error {
	bucket, err := b.client.adminApi.getBucketByName(context.Background(), b.Name)
	if err != nil {
		return errors.Wrap(err, "b.client.adminApi.getBucketByName")
	}
	input := sBucketQuotaInput{}
	input.OsBucket.QuotaMaxObjects = limit.ObjectCount
	input.OsBucket.QuotaMaxSize = limit.SizeBytes
	err = b.client.adminApi.setBucketQuota(context.Background(), bucket.Id, input)
	if err != nil {
		return errors.Wrap(err, "b.client.adminApi.setBucketQuota")
	}
	cloudprovider.Wait(time.Second, 30*time.Second, func() (bool, error) {
		target := b.GetLimit()
		if target.SizeBytes == limit.SizeBytes && target.ObjectCount == limit.ObjectCount {
			return true, nil
		}
		return false, nil
	})
	return nil
}
