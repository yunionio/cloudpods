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

	"yunion.io/x/pkg/errors"
	"yunion.io/x/s3cli"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/s3gateway/models"
)

func headBucket(ctx context.Context, userCred mcclient.TokenCredential, bucketName string) error {
	_, err := models.BucketManager.GetByName(ctx, userCred, bucketName)
	if err != nil {
		return errors.Wrap(err, "models.BucketManager.GetByName")
	}
	return nil
}

func removeBucket(ctx context.Context, userCred mcclient.TokenCredential, bucket string) error {
	return models.BucketManager.DeleteByName(ctx, userCred, bucket)
}

func bucketAcl(ctx context.Context, userCred mcclient.TokenCredential, bucketName string) (*s3cli.AccessControlPolicy, error) {
	bucket, err := models.BucketManager.GetByName(ctx, userCred, bucketName)
	if err != nil {
		return nil, errors.Wrap(err, "models.BucketManager.GetByName")
	}
	iBucket, err := bucket.GetIBucket(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "bucket.GetIBucket")
	}

	result := str2Acl(userCred, iBucket.GetAcl())

	return result, nil
}

func listBucketUploads(ctx context.Context, userCred mcclient.TokenCredential, bucketName string, input *s3cli.ListMultipartUploadsInput) (*s3cli.ListMultipartUploadsResult, error) {
	result := s3cli.ListMultipartUploadsResult{}
	result.Bucket = bucketName
	result.Delimiter = input.Delimiter
	result.MaxUploads = input.MaxUploads
	result.KeyMarker = input.KeyMarker
	result.Prefix = input.Prefix
	result.UploadIDMarker = input.UploadIdMarker
	result.EncodingType = input.EncodingType
	return &result, nil
}
