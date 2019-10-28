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

package models

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/s3cli"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/s3gateway/session"
	"yunion.io/x/onecloud/pkg/util/hashcache"
)

type SBucketManagerDelegate struct {
	buckets *hashcache.Cache
}

var BucketManager *SBucketManagerDelegate

func init() {
	BucketManager = &SBucketManagerDelegate{
		buckets: hashcache.NewCache(2048, time.Minute*15),
	}
}

/*
{
	"access_urls":[{"description":"bucket domain","primary":true,"url":"https://yunion-billing-reports.s3.cn-northwest-1.amazonaws.com.cn"},{"description":"s3 domain","primary":false,"url":"https://s3.cn-northwest-1.amazonaws.com.cn/yunion-billing-reports"}],
	"account":"aws-cn",
	"account_id":"edc90a61-7f8a-4be7-84f1-8f3ac70ef5e6",
	"acl":"private",
	"brand":"Aws",
	"can_delete":false,
	"can_update":true,
	"cloud_env":"public",
	"cloudregion_id":"4cbf92a5-337b-4cc6-82c7-86e5427b69e3",
	"created_at":"2019-03-11T10:31:26.000000Z",
	"domain_id":"default",
	"external_id":"yunion-billing-reports",
	"id":"056bb8c6-527d-4554-8939-12eb6aa803bd",
	"is_emulated":false,
	"is_system":false,
	"location":"cn-northwest-1",
	"manager":"aws-cn",
	"manager_domain":"Default",
	"manager_domain_id":"default",
	"manager_id":"d8df39fa-b212-43c1-8d44-aeef897e216d",
	"manager_project":"system",
	"manager_project_id":"5d65667d112e47249ae66dbd7bc07030",
	"name":"yunion-billing-reports",
	"object_cnt":44,
	"object_cnt_limit":0,
	"project_domain":"Default",
	"project_src":"cloud",
	"provider":"Aws",
	"region":"AWS 中国(宁夏)",
	"region_ext_id":"cn-northwest-1",
	"region_id":"4cbf92a5-337b-4cc6-82c7-86e5427b69e3",
	"size_bytes":3332448,
	"size_bytes_limit":0,
	"status":"ready",
	"tenant":"system",
	"tenant_id":"5d65667d112e47249ae66dbd7bc07030",
	"update_version":1,
	"updated_at":"2019-08-18T15:52:42.000000Z",
}
*/
type SBucketDelegate struct {
	SBaseModelDelegate

	Location  string
	ManagerId string

	ObjectCnt int
	SizeBytes int64

	ObjectCntLimit int
	SizeBytesLimit int64

	RegionExternalId string
	ExternalId       string
}

func (manager *SBucketManagerDelegate) List(ctx context.Context, userCred mcclient.TokenCredential) ([]*SBucketDelegate, error) {
	s := session.GetSession(ctx, userCred)
	offset := 0
	total := -1
	ret := make([]*SBucketDelegate, 0)
	for total < 0 || offset < total {
		params := struct {
			Limit  int
			Offset int
		}{}
		params.Limit = 1000
		params.Offset = offset
		result, err := modules.Buckets.List(s, jsonutils.Marshal(params))
		if err != nil {
			return nil, errors.Wrap(err, "List")
		}
		total = result.Total
		offset += len(result.Data)
		for i := range result.Data {
			bucket := &SBucketDelegate{}
			err := result.Data[i].Unmarshal(bucket)
			if err != nil {
				return nil, errors.Wrap(err, "Unmarshal")
			}
			ret = append(ret, bucket)
			manager.buckets.AtomicSet(bucket.Name, bucket)
		}
	}
	return ret, nil
}

func (manager *SBucketManagerDelegate) GetByName(ctx context.Context, userCred mcclient.TokenCredential, name string) (*SBucketDelegate, error) {
	val := manager.buckets.AtomicGet(name)
	if !gotypes.IsNil(val) {
		return val.(*SBucketDelegate), nil
	}
	s := session.GetSession(ctx, userCred)
	result, err := modules.Buckets.PerformAction(s, name, "sync", nil)
	if err != nil {
		return nil, errors.Wrap(err, "modules.Buckets.Get")
	}
	bucket := &SBucketDelegate{}
	err = result.Unmarshal(bucket)
	if err != nil {
		return nil, errors.Wrap(err, "result.Unmarshal")
	}
	manager.buckets.AtomicSet(bucket.Name, bucket)
	return bucket, nil
}

func (manager *SBucketManagerDelegate) DeleteByName(ctx context.Context, userCred mcclient.TokenCredential, name string) error {
	s := session.GetSession(ctx, userCred)
	_, err := modules.Buckets.Delete(s, name, nil)
	if err != nil {
		return errors.Wrap(err, "modules.Buckets.Delete")
	}
	manager.buckets.AtomicRemove(name)
	return nil
}

func (manager *SBucketManagerDelegate) Invalidate(name string) {
	manager.buckets.AtomicRemove(name)
}

func (bucket *SBucketDelegate) getManager(ctx context.Context, userCred mcclient.TokenCredential) (*SCloudproviderDelegate, error) {
	return CloudproviderManager.GetById(ctx, userCred, bucket.ManagerId)
}

func (bucket *SBucketDelegate) GetIBucket(ctx context.Context, userCred mcclient.TokenCredential) (cloudprovider.ICloudBucket, error) {
	manager, err := bucket.getManager(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "bucket.getManager")
	}
	driver, err := manager.GetProvider()
	if err != nil {
		return nil, errors.Wrap(err, "cloudprovider.GetProvider")
	}
	var iRegion cloudprovider.ICloudRegion
	if len(bucket.RegionExternalId) == 0 {
		iRegion, err = driver.GetOnPremiseIRegion()
	} else {
		iRegion, err = driver.GetIRegionById(bucket.RegionExternalId)
	}
	if err != nil {
		return nil, errors.Wrap(err, "driver.GetIRegionById")
	}
	iBucket, err := iRegion.GetIBucketById(bucket.ExternalId)
	if err != nil {
		return nil, errors.Wrap(err, "iRegion.GetIBucketById")
	}
	return iBucket, nil
}

func (bucket *SBucketDelegate) ListObject(ctx context.Context, userCred mcclient.TokenCredential, input *s3cli.ListObjectInput) (*s3cli.ListBucketResult, error) {
	ibucket, err := bucket.GetIBucket(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "getIBucket")
	}
	result, err := ibucket.ListObjects(input.Prefix, input.Marker, input.Delimiter, int(input.MaxKeys))
	if err != nil {
		return nil, errors.Wrap(err, "ibucket.ListObjects")
	}
	ret := s3cli.ListBucketResult{}
	ret.IsTruncated = result.IsTruncated
	ret.MaxKeys = input.MaxKeys
	ret.Delimiter = input.Delimiter
	ret.Prefix = input.Prefix
	ret.Marker = input.Marker
	ret.CommonPrefixes = make([]s3cli.CommonPrefix, len(result.CommonPrefixes))
	for i := range result.CommonPrefixes {
		ret.CommonPrefixes[i] = s3cli.CommonPrefix{
			Prefix: result.CommonPrefixes[i].GetKey(),
		}
	}
	ret.Contents = make([]s3cli.ObjectInfo, len(result.Objects))
	for i := range result.Objects {
		obj := result.Objects[i]
		ret.Contents[i] = s3cli.ObjectInfo{
			Key:          obj.GetKey(),
			ETag:         obj.GetETag(),
			Size:         obj.GetSizeBytes(),
			LastModified: obj.GetLastModified(),
			ContentType:  obj.GetContentType(),
			StorageClass: obj.GetStorageClass(),
		}
	}
	return &ret, nil
}

func (bucket *SBucketDelegate) IsOutOfLimit() error {
	if bucket.ObjectCntLimit > 0 && bucket.ObjectCnt >= bucket.ObjectCntLimit {
		return errors.Wrap(httperrors.ErrOutOfLimit, "object_count")
	}
	if bucket.SizeBytesLimit > 0 && bucket.SizeBytes >= bucket.SizeBytesLimit {
		return errors.Wrap(httperrors.ErrOutOfLimit, "size_bytes")
	}
	return nil
}

func (bucket *SBucketDelegate) Invalidate() {
	BucketManager.Invalidate(bucket.Name)
}
