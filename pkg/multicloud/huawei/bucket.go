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

package huawei

import (
	"context"
	"fmt"
	"io"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/s3cli"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/multicloud/huawei/obs"
)

type SBucket struct {
	multicloud.SBaseBucket

	region *SRegion

	Name         string
	Location     string
	CreationDate time.Time
}

func (b *SBucket) GetProjectId() string {
	return ""
}

func (b *SBucket) GetGlobalId() string {
	return b.Name
}

func (b *SBucket) GetName() string {
	return b.Name
}

func (b *SBucket) GetLocation() string {
	return b.Location
}

func (b *SBucket) GetIRegion() cloudprovider.ICloudRegion {
	return b.region
}

func (b *SBucket) GetCreateAt() time.Time {
	return b.CreationDate
}

func (b *SBucket) GetStorageClass() string {
	obscli, err := b.region.getOBSClient()
	if err != nil {
		log.Errorf("b.region.getOBSClient error %s", err)
		return ""
	}
	output, err := obscli.GetBucketStoragePolicy(b.Name)
	if err != nil {
		log.Errorf("obscli.GetBucketStoragePolicy error %s", err)
	}
	return output.StorageClass
}

func obsAcl2CannedAcl(acls []obs.Grant) cloudprovider.TBucketACLType {
	switch {
	case len(acls) == 1:
		if acls[0].Grantee.URI == "" && acls[0].Permission == s3cli.PERMISSION_FULL_CONTROL {
			return cloudprovider.ACLPrivate
		}
	case len(acls) == 2:
		for _, g := range acls {
			if g.Grantee.URI == s3cli.GRANTEE_GROUP_URI_AUTH_USERS && g.Permission == s3cli.PERMISSION_READ {
				return cloudprovider.ACLAuthRead
			}
			if g.Grantee.URI == s3cli.GRANTEE_GROUP_URI_ALL_USERS && g.Permission == s3cli.PERMISSION_READ {
				return cloudprovider.ACLPublicRead
			}
		}
	case len(acls) == 3:
		for _, g := range acls {
			if g.Grantee.URI == s3cli.GRANTEE_GROUP_URI_ALL_USERS && g.Permission == s3cli.PERMISSION_WRITE {
				return cloudprovider.ACLPublicReadWrite
			}
		}
	}
	return cloudprovider.ACLUnknown
}

func (b *SBucket) GetAcl() cloudprovider.TBucketACLType {
	acl := cloudprovider.ACLPrivate
	obscli, err := b.region.getOBSClient()
	if err != nil {
		log.Errorf("b.region.getOBSClient error %s", err)
		return acl
	}
	output, err := obscli.GetBucketAcl(b.Name)
	if err != nil {
		log.Errorf("obscli.GetBucketAcl error %s", err)
		return acl
	}
	acl = obsAcl2CannedAcl(output.Grants)
	return acl
}

func (b *SBucket) SetAcl(acl cloudprovider.TBucketACLType) error {
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return errors.Wrap(err, "b.region.getOBSClient")
	}
	input := &obs.SetBucketAclInput{}
	input.Bucket = b.Name
	input.ACL = obs.AclType(string(acl))
	_, err = obscli.SetBucketAcl(input)
	if err != nil {
		return errors.Wrap(err, "obscli.SetBucketAcl")
	}
	return nil
}

func (b *SBucket) GetAccessUrls() []cloudprovider.SBucketAccessUrl {
	return []cloudprovider.SBucketAccessUrl{
		{
			Url:         fmt.Sprintf("https://%s.%s", b.Name, b.region.getOBSEndpoint()),
			Description: "bucket url",
		},
		{
			Url:         fmt.Sprintf("https://%s/%s", b.region.getOBSEndpoint(), b.Name),
			Description: "obs url",
		},
	}
}

func (b *SBucket) GetStats() cloudprovider.SBucketStats {
	stats := cloudprovider.SBucketStats{}
	obscli, err := b.region.getOBSClient()
	if err != nil {
		log.Errorf("b.region.getOBSClient error %s", err)
		stats.SizeBytes = -1
		stats.ObjectCount = -1
		return stats
	}
	output, err := obscli.GetBucketStorageInfo(b.Name)
	if err != nil {
		log.Errorf("obscli.GetBucketStorageInfo error %s", err)
		stats.SizeBytes = -1
		stats.ObjectCount = -1
		return stats
	}
	stats.SizeBytes = output.Size
	stats.ObjectCount = output.ObjectNumber
	return stats
}

func (b *SBucket) GetIObjects(prefix string, isRecursive bool) ([]cloudprovider.ICloudObject, error) {
	return cloudprovider.GetIObjects(b, prefix, isRecursive)
}

func (b *SBucket) ListObjects(prefix string, marker string, delimiter string, maxCount int) (cloudprovider.SListObjectResult, error) {
	result := cloudprovider.SListObjectResult{}
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return result, errors.Wrap(err, "GetOBSClient")
	}
	input := &obs.ListObjectsInput{}
	input.Bucket = b.Name
	if len(prefix) > 0 {
		input.Prefix = prefix
	}
	if len(marker) > 0 {
		input.Marker = marker
	}
	if len(delimiter) > 0 {
		input.Delimiter = delimiter
	}
	if maxCount > 0 {
		input.MaxKeys = maxCount
	}
	oResult, err := obscli.ListObjects(input)
	if err != nil {
		return result, errors.Wrap(err, "ListObjects")
	}
	result.Objects = make([]cloudprovider.ICloudObject, 0)
	for _, object := range oResult.Contents {
		obj := &SObject{
			bucket: b,
			SBaseCloudObject: cloudprovider.SBaseCloudObject{
				StorageClass: string(object.StorageClass),
				Key:          object.Key,
				SizeBytes:    object.Size,
				ETag:         object.ETag,
				LastModified: object.LastModified,
				ContentType:  "",
			},
		}
		result.Objects = append(result.Objects, obj)
	}
	if oResult.CommonPrefixes != nil {
		result.CommonPrefixes = make([]cloudprovider.ICloudObject, 0)
		for _, commonPrefix := range oResult.CommonPrefixes {
			obj := &SObject{
				bucket: b,
				SBaseCloudObject: cloudprovider.SBaseCloudObject{
					Key: commonPrefix,
				},
			}
			result.CommonPrefixes = append(result.CommonPrefixes, obj)
		}
	}
	result.IsTruncated = oResult.IsTruncated
	result.NextMarker = oResult.NextMarker
	return result, nil
}

func (b *SBucket) PutObject(ctx context.Context, key string, reader io.Reader, contType string, storageClassStr string) error {
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return errors.Wrap(err, "GetOBSClient")
	}
	input := &obs.PutObjectInput{}
	input.Bucket = b.Name
	input.Key = key
	input.Body = reader
	if len(storageClassStr) > 0 {
		input.StorageClass, err = str2StorageClass(storageClassStr)
		if err != nil {
			return err
		}
	}
	if len(contType) > 0 {
		input.ContentType = contType
	}
	_, err = obscli.PutObject(input)
	if err != nil {
		return errors.Wrap(err, "PutObject")
	}
	return nil
}

func (b *SBucket) DeleteObject(ctx context.Context, key string) error {
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return errors.Wrap(err, "GetOBSClient")
	}
	input := &obs.DeleteObjectInput{}
	input.Bucket = b.Name
	input.Key = key
	_, err = obscli.DeleteObject(input)
	if err != nil {
		return errors.Wrap(err, "DeleteObject")
	}
	return nil
}

func (b *SBucket) GetTempUrl(method string, key string, expire time.Duration) (string, error) {
	obscli, err := b.region.getOBSClient()
	if err != nil {
		return "", errors.Wrap(err, "GetOBSClient")
	}
	input := obs.CreateSignedUrlInput{}
	input.Bucket = b.Name
	input.Key = key
	input.Expires = int(expire / time.Second)
	switch method {
	case "GET":
		input.Method = obs.HttpMethodGet
	case "PUT":
		input.Method = obs.HttpMethodPut
	case "DELETE":
		input.Method = obs.HttpMethodDelete
	default:
		return "", errors.Error("unsupported method")
	}
	output, err := obscli.CreateSignedUrl(&input)
	return output.SignedUrl, nil
}
