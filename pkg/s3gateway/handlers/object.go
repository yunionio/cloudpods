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
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/s3cli"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/s3gateway/models"
)

func headObject(ctx context.Context, userCred mcclient.TokenCredential, bucketName string, key string) (http.Header, error) {
	bucket, err := models.BucketManager.GetByName(ctx, userCred, bucketName)
	if err != nil {
		return nil, errors.Wrap(err, "models.BucketManager.GetByName")
	}
	iBucket, err := bucket.GetIBucket(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "bucket.GetIBucket")
	}
	obj, err := cloudprovider.GetIObject(iBucket, key)
	if err != nil {
		return nil, errors.Wrap(err, "cloudprovider.GetIObject")
	}
	hdr := cloudprovider.MetaToHttpHeader(cloudprovider.META_HEADER_PREFIX, obj.GetMeta())
	hdr.Set(http.CanonicalHeaderKey("x-amz-acl"), string(obj.GetAcl()))
	hdr.Set(http.CanonicalHeaderKey("x-amz-storage-class"), obj.GetStorageClass())
	hdr.Set(http.CanonicalHeaderKey("content-length"), strconv.FormatInt(obj.GetSizeBytes(), 10))
	hdr.Set(http.CanonicalHeaderKey("etag"), obj.GetETag())
	hdr.Set(http.CanonicalHeaderKey("last-modified"), obj.GetLastModified().Format(timeutils.RFC2882Format))
	return hdr, nil
}

func uploadObject(ctx context.Context, userCred mcclient.TokenCredential, bucketName string, key string, header http.Header, body io.Reader, uploadId string, partNumber int) (http.Header, error) {
	bucket, err := models.BucketManager.GetByName(ctx, userCred, bucketName)
	if err != nil {
		return nil, errors.Wrap(err, "models.BucketManager.GetByName")
	}

	err = bucket.IsOutOfLimit()
	if err != nil {
		return nil, errors.Wrap(err, "IsOutOfLimit")
	}

	iBucket, err := bucket.GetIBucket(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "bucket.GetIBucket")
	}

	contLenStr := header.Get(http.CanonicalHeaderKey("Content-Length"))
	contLen, err := strconv.ParseInt(contLenStr, 10, 64)
	if err != nil {
		return nil, errors.Wrap(httperrors.ErrBadRequest, "missing content length")
	}
	respHdr := http.Header{}
	if len(uploadId) > 0 {
		etag, err := iBucket.UploadPart(ctx, key, uploadId, partNumber, body, contLen)
		if err != nil {
			return nil, errors.Wrap(err, "iBucket.UploadPart")
		}
		respHdr.Set("ETag", etag)
	} else {
		meta := cloudprovider.FetchMetaFromHttpHeader(cloudprovider.META_HEADER_PREFIX, header)
		aclStr := header.Get(http.CanonicalHeaderKey("x-amz-acl"))
		storageClassStr := header.Get(http.CanonicalHeaderKey("x-amz-storage-class"))
		err = iBucket.PutObject(ctx, key, body, contLen, cloudprovider.TBucketACLType(aclStr), storageClassStr, meta)
		if err != nil {
			return nil, errors.Wrap(err, "iBucket.PutObject")
		}
		obj, err := cloudprovider.GetIObject(iBucket, key)
		if err != nil {
			return nil, errors.Wrap(err, "cloudprovider.GetIObject")
		}
		respHdr.Set("ETag", obj.GetETag())
	}

	bucket.Invalidate()

	return respHdr, nil
}

const (
	MIN_PART_BYTES = 1000 * 1000 * 10 // 100 MB
	MAX_PART_COUNT = 10000
)

func copyObject(ctx context.Context, userCred mcclient.TokenCredential, bucketName string, key string, copySource string, hdr http.Header, uploadId string, partNumber int) (interface{}, http.Header, error) {
	log.Debugf("CopyObject %s => %s/%s %s %d %s", copySource, bucketName, key, uploadId, partNumber, hdr)
	srcSegs := appsrv.SplitPath(copySource)
	srcBucketName := srcSegs[0]
	srcKey := strings.Join(srcSegs[1:], "/")
	if strings.HasSuffix(copySource, "/") {
		srcKey += "/"
	}
	var err error
	srcKey, err = url.PathUnescape(srcKey)
	if err != nil {
		return nil, nil, errors.Wrap(err, "url.PathUnescape")
	}

	srcBucket, err := models.BucketManager.GetByName(ctx, userCred, srcBucketName)
	if err != nil {
		return nil, nil, errors.Wrap(err, "source bucket GetByName")
	}
	iSrcBucket, err := srcBucket.GetIBucket(ctx, userCred)
	if err != nil {
		return nil, nil, errors.Wrap(err, "srcBucket.GetIBucket")
	}
	srcObj, err := cloudprovider.GetIObject(iSrcBucket, srcKey)
	if err != nil {
		return nil, nil, errors.Wrap(err, "src cloudprovider.GetIObject")
	}

	dstBucket, err := models.BucketManager.GetByName(ctx, userCred, bucketName)
	if err != nil {
		return nil, nil, errors.Wrap(err, "dest bucket GetByName")
	}

	err = dstBucket.IsOutOfLimit()
	if err != nil {
		return nil, nil, errors.Wrap(err, "IsOutOfLimit")
	}

	iDstBucket, err := dstBucket.GetIBucket(ctx, userCred)
	if err != nil {
		return nil, nil, errors.Wrap(err, "dstBucket.GetIBucket")
	}

	sizeBytes := srcObj.GetSizeBytes()
	rangeStr := hdr.Get(http.CanonicalHeaderKey("x-amz-copy-source-range"))
	rangeOpt, err := getRangeOpt(rangeStr, sizeBytes)
	if err != nil {
		return nil, nil, errors.Wrap(err, rangeStr)
	}

	if rangeOpt != nil {
		if len(uploadId) == 0 {
			return nil, nil, errors.Wrap(httperrors.ErrBadRequest, "range copy must be a multipart upload")
		}
		// upload directory
		var etag string
		if dstBucket.ManagerId == srcBucket.ManagerId && dstBucket.RegionExternalId == srcBucket.RegionExternalId {
			etag, err = iDstBucket.CopyPart(ctx, key, uploadId, partNumber, iSrcBucket.GetName(), srcKey, rangeOpt.Start, rangeOpt.SizeBytes())
		} else {
			etag, err = cloudprovider.CopyPart(ctx, iDstBucket, key, uploadId, partNumber, iSrcBucket, srcKey, rangeOpt)
		}
		if err != nil {
			return nil, nil, errors.Wrap(err, "copyPart fail")
		}
		result := s3cli.CopyPartResult{
			ETag:         etag,
			LastModified: srcObj.GetLastModified(),
		}
		return &result, nil, nil
	} else {
		meta := cloudprovider.FetchMetaFromHttpHeader(cloudprovider.META_HEADER_PREFIX, hdr)
		if dstBucket.ManagerId == srcBucket.ManagerId && dstBucket.RegionExternalId == srcBucket.RegionExternalId {
			err = iDstBucket.CopyObject(ctx, key, iSrcBucket.GetName(), srcKey, srcObj.GetAcl(), srcObj.GetStorageClass(), meta)
			if err != nil {
				return nil, nil, errors.Wrap(err, "iDstBucket.CopyObject")
			}
		} else {
			err = cloudprovider.CopyObject(ctx, 0, iDstBucket, key, iSrcBucket, srcKey, meta, false)
			if err != nil {
				return nil, nil, errors.Wrap(err, "cloudprovider.CopyObject")
			}
		}

		dstBucket.Invalidate()

		dstObj, err := cloudprovider.GetIObject(iDstBucket, key)
		if err != nil {
			return nil, nil, errors.Wrap(err, "cloudprovider.GetIObject")
		}
		result := s3cli.CopyObjectResult{
			ETag:         dstObj.GetETag(),
			LastModified: dstObj.GetLastModified(),
		}
		return &result, nil, nil
	}
}

func deleteObjectTags(ctx context.Context, userCred mcclient.TokenCredential, bucket string, key string) (*s3cli.Tagging, error) {
	return nil, nil
}

func removeObject(ctx context.Context, userCred mcclient.TokenCredential, bucketName string, key string) error {
	bucket, err := models.BucketManager.GetByName(ctx, userCred, bucketName)
	if err != nil {
		return errors.Wrap(err, "models.BucketManager.GetByName")
	}
	iBucket, err := bucket.GetIBucket(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "bucket.GetIBucket")
	}
	err = iBucket.DeleteObject(ctx, key)
	if err != nil {
		return errors.Wrap(err, "DeleteObject")
	}

	bucket.Invalidate()

	return nil
}

func objectAcl(ctx context.Context, userCred mcclient.TokenCredential, bucketName string, objKey string) (*s3cli.AccessControlPolicy, error) {
	bucket, err := models.BucketManager.GetByName(ctx, userCred, bucketName)
	if err != nil {
		return nil, errors.Wrap(err, "models.BucketManager.GetByName")
	}
	iBucket, err := bucket.GetIBucket(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "bucket.GetIBucket")
	}
	obj, err := cloudprovider.GetIObject(iBucket, objKey)
	if err != nil {
		return nil, errors.Wrap(err, "cloudprovider.GetIObject")
	}

	result := str2Acl(userCred, obj.GetAcl())

	return result, nil
}

func str2Acl(userCred mcclient.TokenCredential, aclStr cloudprovider.TBucketACLType) *s3cli.AccessControlPolicy {
	result := s3cli.AccessControlPolicy{}
	result.Owner.DisplayName = userCred.GetProjectName()
	result.Owner.ID = userCred.GetProjectId()

	fullControl := s3cli.Grant{}
	fullControl.Permission = s3cli.PERMISSION_FULL_CONTROL
	fullControl.Grantee.Type = s3cli.GRANTEE_TYPE_USER
	fullControl.Grantee.ID = userCred.GetProjectId()
	fullControl.Grantee.DisplayName = userCred.GetProjectName()

	publicRead := s3cli.Grant{}
	publicRead.Permission = s3cli.PERMISSION_READ
	publicRead.Grantee.Type = s3cli.GRANTEE_TYPE_GROUP
	publicRead.Grantee.URI = s3cli.GRANTEE_GROUP_URI_ALL_USERS

	publicWrite := s3cli.Grant{}
	publicWrite.Permission = s3cli.PERMISSION_WRITE
	publicWrite.Grantee.Type = s3cli.GRANTEE_TYPE_GROUP
	publicWrite.Grantee.URI = s3cli.GRANTEE_GROUP_URI_ALL_USERS

	authRead := s3cli.Grant{}
	authRead.Permission = s3cli.PERMISSION_READ
	authRead.Grantee.Type = s3cli.GRANTEE_TYPE_GROUP
	authRead.Grantee.URI = s3cli.GRANTEE_GROUP_URI_AUTH_USERS

	switch aclStr {
	case cloudprovider.ACLPrivate:
		result.AccessControlList.Grant = []s3cli.Grant{
			fullControl,
		}
	case cloudprovider.ACLAuthRead:
		result.AccessControlList.Grant = []s3cli.Grant{
			fullControl,
			authRead,
		}
	case cloudprovider.ACLPublicRead:
		result.AccessControlList.Grant = []s3cli.Grant{
			fullControl,
			publicRead,
		}
	case cloudprovider.ACLPublicReadWrite:
		result.AccessControlList.Grant = []s3cli.Grant{
			fullControl,
			publicRead,
			publicWrite,
		}
	}
	return &result
}
