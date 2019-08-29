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

package handlers

import (
	"context"
	"net/http"
	"sort"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/s3cli"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/s3gateway/models"
)

func initMultipartUpload(ctx context.Context, userCred mcclient.TokenCredential, hdr http.Header, bucketName string, key string) (*s3cli.InitiateMultipartUploadResult, http.Header, error) {
	bucket, err := models.BucketManager.GetByName(ctx, userCred, bucketName)
	if err != nil {
		return nil, nil, errors.Wrap(err, "models.BucketManager.GetByName")
	}
	iBucket, err := bucket.GetIBucket(ctx, userCred)
	if err != nil {
		return nil, nil, errors.Wrap(err, "bucket.GetIBucket")
	}
	contType := hdr.Get(http.CanonicalHeaderKey("content-type"))
	aclStr := hdr.Get(http.CanonicalHeaderKey("x-amz-acl"))
	storageClassStr := hdr.Get(http.CanonicalHeaderKey("x-amz-storage-class"))
	uploadId, err := iBucket.NewMultipartUpload(ctx, key, contType, cloudprovider.TBucketACLType(aclStr), storageClassStr)
	if err != nil {
		return nil, nil, errors.Wrap(err, "NewMultipartUpload")
	}
	result := s3cli.InitiateMultipartUploadResult{}
	result.Bucket = bucketName
	result.Key = key
	result.UploadID = uploadId
	return &result, nil, nil
}

type SMultiparts []s3cli.CompletePart

func (a SMultiparts) Len() int           { return len(a) }
func (a SMultiparts) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SMultiparts) Less(i, j int) bool { return a[i].PartNumber < a[j].PartNumber }

func completeMultipartUpload(ctx context.Context, userCred mcclient.TokenCredential, hdr http.Header, bucketName string, key string, uploadId string, request *s3cli.CompleteMultipartUpload) (*s3cli.CompleteMultipartUploadResult, http.Header, error) {
	bucket, err := models.BucketManager.GetByName(ctx, userCred, bucketName)
	if err != nil {
		return nil, nil, errors.Wrap(err, "models.BucketManager.GetByName")
	}
	iBucket, err := bucket.GetIBucket(ctx, userCred)
	if err != nil {
		return nil, nil, errors.Wrap(err, "bucket.GetIBucket")
	}
	sort.Sort(SMultiparts(request.Parts))
	partEtags := make([]string, len(request.Parts))
	for i := range request.Parts {
		partEtags[i] = request.Parts[i].ETag
	}
	err = iBucket.CompleteMultipartUpload(ctx, key, uploadId, partEtags)
	if err != nil {
		return nil, nil, errors.Wrap(err, "CompleteMultipartUpload")
	}
	obj, err := cloudprovider.GetIObject(iBucket, key)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cloudprovider.GetIObject")
	}
	result := s3cli.CompleteMultipartUploadResult{}
	result.Bucket = bucketName
	result.Key = key
	result.ETag = obj.GetETag()
	result.Location = iBucket.GetLocation()
	return &result, nil, nil
}
