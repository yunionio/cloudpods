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

package ksyun

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ks3sdklib/aws-sdk-go/aws"
	"github.com/ks3sdklib/aws-sdk-go/aws/credentials"
	"github.com/ks3sdklib/aws-sdk-go/service/s3"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/fileutils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SBucket struct {
	multicloud.SBaseBucket
	SKsyunTags

	region *SRegion

	CreationDate       time.Time
	Name               string
	Region             string
	Type               string
	VisitType          string
	DataRedundancyType string
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

func (b *SBucket) GetAcl() cloudprovider.TBucketACLType {
	return b.region.GetBucketAcl(b.Name)
}

func (region *SRegion) GetBucketAcl(bucket string) cloudprovider.TBucketACLType {
	svc := region.getS3Client()
	input := &s3.GetBucketACLInput{
		Bucket: &bucket,
	}
	resp, err := svc.GetBucketACL(input)
	if err != nil {
		return cloudprovider.ACLUnknown
	}
	return cloudprovider.TBucketACLType(s3.GetCannedACL(resp.Grants))
}

func (b *SBucket) GetLocation() string {
	return b.Region
}

func (b *SBucket) GetIRegion() cloudprovider.ICloudRegion {
	return b.region
}

func (b *SBucket) GetCreatedAt() time.Time {
	return b.CreationDate
}

func (b *SBucket) GetStorageClass() string {
	return b.DataRedundancyType
}

// https://docs.ksyun.com/documents/6761
func (b *SBucket) GetAccessUrls() []cloudprovider.SBucketAccessUrl {
	ret := []cloudprovider.SBucketAccessUrl{
		{
			Url:         fmt.Sprintf("%s.%s", b.Name, s3RegionEndpointMap[b.region.Region]),
			Description: "ExtranetEndpoint",
			Primary:     true,
		},
		{
			Url:         fmt.Sprintf("%s.%s", b.Name, strings.Replace(s3RegionEndpointMap[b.region.Region], ".ksyuncs.com", "-internal.ksyuncs.com", 1)),
			Description: "IntranetEndpoint",
		},
	}
	return ret
}

func (b *SBucket) GetStats() cloudprovider.SBucketStats {
	stats, err := cloudprovider.GetIBucketStats(b)
	if err != nil {
		log.Errorf("GetStats fail %s", err)
	}
	return stats
}

func (region *SRegion) SetBucketAcl(bucket string, acl cloudprovider.TBucketACLType) error {
	svc := region.getS3Client()
	aclStr := string(acl)
	input := &s3.PutBucketACLInput{
		Bucket: &bucket,
		ACL:    &aclStr,
	}
	_, err := svc.PutBucketACL(input)
	return err
}

func (b *SBucket) SetAcl(aclStr cloudprovider.TBucketACLType) error {
	return b.region.SetBucketAcl(b.Name, aclStr)
}

func (b *SBucket) ListObjects(prefix string, marker string, delimiter string, maxCount int) (cloudprovider.SListObjectResult, error) {
	result := cloudprovider.SListObjectResult{}
	svc := b.region.getS3Client()
	input := &s3.ListObjectsInput{
		Bucket: &b.Name,
	}
	if len(prefix) > 0 {
		input.Prefix = &prefix
	}
	if len(delimiter) > 0 {
		input.Delimiter = &delimiter
	}
	if len(marker) > 0 {
		input.Marker = &marker
	}
	if maxCount > 0 {
		cnt := int64(maxCount)
		input.MaxKeys = &cnt
	}
	resp, err := svc.ListObjects(input)
	if err != nil {
		return result, err
	}
	result.Objects = make([]cloudprovider.ICloudObject, 0)
	for _, object := range resp.Contents {
		ksObj := cloudprovider.SBaseCloudObject{}
		if object.StorageClass != nil {
			ksObj.StorageClass = *object.StorageClass
		}
		if object.Key != nil {
			ksObj.Key = *object.Key
		}
		if object.Size != nil {
			ksObj.SizeBytes = *object.Size
		}
		if object.ETag != nil {
			ksObj.ETag = *object.ETag
		}
		if object.LastModified != nil {
			ksObj.LastModified = *object.LastModified
		}

		obj := &SObject{
			bucket:           b,
			SBaseCloudObject: ksObj,
		}
		result.Objects = append(result.Objects, obj)
	}
	if resp.CommonPrefixes != nil {
		result.CommonPrefixes = make([]cloudprovider.ICloudObject, len(resp.CommonPrefixes))
		for i, commonPrefix := range resp.CommonPrefixes {
			result.CommonPrefixes[i] = &SObject{
				bucket:           b,
				SBaseCloudObject: cloudprovider.SBaseCloudObject{Key: *commonPrefix.Prefix},
			}
		}
	}
	if resp.IsTruncated != nil {
		result.IsTruncated = *resp.IsTruncated
	}
	if resp.NextMarker != nil {
		result.NextMarker = *resp.NextMarker
	}
	return result, nil
}

func (b *SBucket) PutObject(ctx context.Context, key string, reader io.Reader, sizeBytes int64, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) error {
	svc := b.region.getS3Client()
	input := &s3.PutObjectInput{
		Bucket: &b.Name,
		Key:    &key,
	}
	if sizeBytes > 0 {
		input.ContentLength = &sizeBytes
	}
	if meta != nil {
		input.Metadata = make(map[string]*string)
		for k, v := range meta {
			input.Metadata[k] = &v[0]
		}
	}
	if len(cannedAcl) == 0 {
		cannedAcl = b.GetAcl()
	}
	acl := string(cannedAcl)
	input.ACL = &acl
	if len(storageClassStr) > 0 {
		storageClass := string(storageClassStr)
		input.StorageClass = &storageClass
	}
	seeker, err := fileutils.NewReadSeeker(reader, sizeBytes)
	if err != nil {
		return errors.Wrap(err, "newFakeSeeker")
	}
	defer seeker.Close()
	input.Body = seeker
	_, err = svc.PutObject(input)
	if err != nil {
		return errors.Wrap(err, "PutObject")
	}
	return nil
}

func (b *SBucket) NewMultipartUpload(ctx context.Context, key string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) (string, error) {
	svc := b.region.getS3Client()
	input := &s3.CreateMultipartUploadInput{
		Bucket: &b.Name,
		Key:    &key,
	}
	if meta != nil {
		input.Metadata = make(map[string]*string)
		for k, v := range meta {
			input.Metadata[k] = &v[0]
		}
	}
	if len(cannedAcl) == 0 {
		cannedAcl = b.GetAcl()
	}
	acl := string(cannedAcl)
	input.ACL = &acl
	if len(storageClassStr) > 0 {
		storageClass := string(storageClassStr)
		input.StorageClass = &storageClass
	}
	output, err := svc.CreateMultipartUpload(input)
	if err != nil {
		return "", errors.Wrap(err, "CreateMultipartUpload")
	}
	return *output.UploadID, nil
}

func (b *SBucket) UploadPart(ctx context.Context, key string, uploadId string, partIndex int, reader io.Reader, partSize int64, offset, totalSize int64) (string, error) {
	svc := b.region.getS3Client()
	pn := int64(partIndex)
	input := &s3.UploadPartInput{
		Bucket:     &b.Name,
		Key:        &key,
		UploadID:   &uploadId,
		PartNumber: &pn,
	}
	seeker, err := fileutils.NewReadSeeker(reader, partSize)
	if err != nil {
		return "", errors.Wrap(err, "newFakeSeeker")
	}
	defer seeker.Close()
	input.Body = seeker
	output, err := svc.UploadPart(input)
	if err != nil {
		return "", errors.Wrap(err, "UploadPart")
	}
	return *output.ETag, nil
}

func (b *SBucket) CompleteMultipartUpload(ctx context.Context, key string, uploadId string, partEtags []string) error {
	svc := b.region.getS3Client()
	input := &s3.CompleteMultipartUploadInput{
		Bucket:          &b.Name,
		Key:             &key,
		UploadID:        &uploadId,
		MultipartUpload: &s3.CompletedMultipartUpload{},
	}
	parts := make([]*s3.CompletedPart, len(partEtags))
	for i := range partEtags {
		pn := int64(i + 1)
		parts[i] = &s3.CompletedPart{
			PartNumber: &pn,
			ETag:       &partEtags[i],
		}
	}
	input.MultipartUpload.Parts = parts
	_, err := svc.CompleteMultipartUpload(input)
	return err
}

func (b *SBucket) AbortMultipartUpload(ctx context.Context, key string, uploadId string) error {
	svc := b.region.getS3Client()
	input := &s3.AbortMultipartUploadInput{
		Bucket:   &b.Name,
		Key:      &key,
		UploadID: &uploadId,
	}
	_, err := svc.AbortMultipartUpload(input)
	return err
}

func (b *SBucket) DeleteObject(ctx context.Context, key string) error {
	svc := b.region.getS3Client()
	input := &s3.DeleteObjectInput{
		Bucket: &b.Name,
		Key:    &key,
	}
	_, err := svc.DeleteObject(input)
	return err
}

func (b *SBucket) GetTempUrl(method string, key string, expire time.Duration) (string, error) {
	svc := b.region.getS3Client()
	input := &s3.GeneratePresignedUrlInput{
		Bucket: &b.Name,
		Key:    &key,
	}
	input.HTTPMethod = s3.HTTPMethod(method)
	input.Expires = int64(expire / time.Second)
	return svc.GeneratePresignedUrl(input)
}

func (b *SBucket) CopyObject(ctx context.Context, destKey string, srcBucket, srcKey string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) error {
	svc := b.region.getS3Client()
	input := &s3.CopyObjectInput{
		Bucket:       &b.Name,
		Key:          &destKey,
		SourceBucket: &srcBucket,
		SourceKey:    &srcKey,
	}
	if meta != nil {
		input.Metadata = make(map[string]*string)
		for k, v := range meta {
			input.Metadata[k] = &v[0]
		}
	}
	if cannedAcl != cloudprovider.ACLPrivate {
		acl := string(cannedAcl)
		input.ACL = &acl
	}
	if storageClassStr != "" {
		storageClass := string(storageClassStr)
		input.StorageClass = &storageClass
	}
	_, err := svc.CopyObject(input)
	return err
}

func (b *SBucket) GetObject(ctx context.Context, key string, rangeOpt *cloudprovider.SGetObjectRange) (io.ReadCloser, error) {
	svc := b.region.getS3Client()
	input := &s3.GetObjectInput{
		Bucket: &b.Name,
		Key:    &key,
	}
	if rangeOpt != nil {
		rangeInput := rangeOpt.String()
		input.Range = &rangeInput
	}
	output, err := svc.GetObject(input)
	if err != nil {
		return nil, errors.Wrap(err, "GetObject")
	}
	return output.Body, nil
}

func (b *SBucket) CopyPart(ctx context.Context, key string, uploadId string, partNumber int, srcBucket string, srcKey string, srcOffset int64, srcLength int64) (string, error) {
	svc := b.region.getS3Client()
	pn := int64(partNumber)
	copySourceRange := fmt.Sprintf("bytes=%d-%d", srcOffset, srcOffset+srcLength-1)
	input := &s3.UploadPartCopyInput{
		Bucket:          &b.Name,
		Key:             &key,
		UploadID:        &uploadId,
		PartNumber:      &pn,
		SourceBucket:    &srcBucket,
		SourceKey:       &srcKey,
		CopySourceRange: &copySourceRange,
	}
	output, err := svc.UploadPartCopy(input)
	if err != nil {
		return "", errors.Wrap(err, "UploadPartCopy")
	}
	return *output.CopyPartResult.ETag, nil

}

func (b *SBucket) GetTags() (map[string]string, error) {
	svc := b.region.getS3Client()
	input := &s3.GetBucketTaggingInput{
		Bucket: &b.Name,
	}
	output, err := svc.GetBucketTagging(input)
	if err != nil {
		return nil, errors.Wrap(err, "GetBucketTagging")
	}
	result := map[string]string{}
	if output.Tagging == nil {
		return nil, nil
	}
	for _, tag := range output.Tagging.TagSet {
		result[*tag.Key] = *tag.Value
	}
	return result, nil
}

func (b *SBucket) SetTags(tags map[string]string, replace bool) error {
	svc := b.region.getS3Client()
	input := &s3.PutBucketTaggingInput{
		Bucket: &b.Name,
	}
	if replace {
		input.Tagging = &s3.Tagging{
			TagSet: make([]*s3.Tag, 0),
		}
	}
	for k, v := range tags {
		input.Tagging.TagSet = append(input.Tagging.TagSet, &s3.Tag{Key: &k, Value: &v})
	}
	_, err := svc.PutBucketTagging(input)
	log.Infof("put tagging %s error: %v", jsonutils.Marshal(input).String(), err)
	if err != nil {
		return errors.Wrapf(err, "PutBucketTagging(%s)", b.Name)
	}
	return nil
}

func (b *SBucket) ListMultipartUploads() ([]cloudprovider.SBucketMultipartUploads, error) {
	result := []cloudprovider.SBucketMultipartUploads{}
	svc := b.region.getS3Client()
	keyMarker := ""
	uploadIDMarker := ""
	for {
		input := &s3.ListMultipartUploadsInput{
			Bucket:         &b.Name,
			KeyMarker:      &keyMarker,
			UploadIDMarker: &uploadIDMarker,
		}
		output, err := svc.ListMultipartUploads(input)
		if err != nil {
			return nil, errors.Wrap(err, "ListMultipartUploads")
		}
		for _, upload := range output.Uploads {
			result = append(result, cloudprovider.SBucketMultipartUploads{
				ObjectName: *upload.Key,
				UploadID:   *upload.UploadID,
				Initiated:  *upload.Initiated,
			})
		}
		keyMarker = *output.NextKeyMarker
		uploadIDMarker = *output.NextUploadIDMarker
		if output.IsTruncated == nil || !*output.IsTruncated {
			break
		}
	}
	return result, nil
}

func (region *SRegion) GetIBuckets() ([]cloudprovider.ICloudBucket, error) {
	buckets, err := region.client.GetBuckets()
	if err != nil {
		return nil, err
	}
	ret := make([]cloudprovider.ICloudBucket, 0)
	for i := range buckets {
		buckets[i].region = region
		if strings.EqualFold(buckets[i].Region, s3RegionMap[region.Region]) {
			ret = append(ret, &buckets[i])
		}
	}
	return ret, nil
}

func (region *SRegion) getS3Client() *s3.S3 {
	return region.client.getS3Client(region.Region)
}

// https://docs.ksyun.com/documents/6761?type=3
var s3RegionMap = map[string]string{
	"":               "BEIJING",
	"ap-singapore-1": "SINGAPORE",
	"cn-beijing-6":   "BEIJING",
	"cn-guangzhou-1": "GUANGZHOU",
	"cn-shanghai-2":  "SHANGHAI",
	"cn-northwest-1": "QINGYANG",
}
var s3RegionEndpointMap = map[string]string{
	"":               "ks3-cn-beijing.ksyuncs.com",
	"ap-singapore-1": "ks3-sgp.ksyuncs.com",
	"cn-beijing-6":   "ks3-cn-beijing.ksyuncs.com",
	"cn-guangzhou-1": "ks3-cn-guangzhou.ksyuncs.com",
	"cn-shanghai-2":  "ks3-cn-shanghai.ksyuncs.com",
	"cn-northwest-1": "ks3-cn-qingyang.ksyuncs.com",
}

func (cli *SKsyunClient) getS3Client(regionId string) *s3.S3 {
	aksk := credentials.NewStaticCredentials(cli.accessKeyId, cli.accessKeySecret, "")
	cfg := aws.Config{
		Region:        s3RegionMap[regionId],
		Credentials:   aksk,
		Endpoint:      s3RegionEndpointMap[regionId],
		HTTPClient:    cli.getDefaultClient(),
		SignerVersion: "V4_UNSIGNED_PAYLOAD_SIGNER",
		MaxRetries:    1,
	}
	if cli.debug {
		cfg.LogLevel = aws.Debug
	}
	return s3.New(&cfg)
}

func (cli *SKsyunClient) GetBuckets() ([]SBucket, error) {
	svc := cli.getS3Client("")
	input := &s3.ListBucketsInput{}
	resp, err := svc.ListBuckets(input)
	if err != nil {
		return nil, err
	}
	ret := make([]SBucket, 0)
	for _, b := range resp.Buckets {
		ret = append(ret, SBucket{
			Name:               *b.Name,
			Region:             *b.Region,
			Type:               *b.Type,
			VisitType:          *b.VisitType,
			DataRedundancyType: *b.DataRedundancyType,
			CreationDate:       *b.CreationDate,
		})
	}
	return ret, nil
}

func (region *SRegion) CreateIBucket(name string, storageClassStr string, aclStr string) error {
	svc := region.getS3Client()
	input := &s3.CreateBucketInput{
		Bucket: &name,
	}
	if aclStr != "" {
		acl := string(aclStr)
		input.ACL = &acl
	}
	_, err := svc.CreateBucket(input)
	return err
}

func (region *SRegion) GetIBucketById(name string) (cloudprovider.ICloudBucket, error) {
	buckets, err := region.GetIBuckets()
	if err != nil {
		return nil, errors.Wrap(err, "region.GetBuckets")
	}
	for _, b := range buckets {
		if b.GetName() == name {
			return b, nil
		}
	}
	return nil, errors.Wrap(cloudprovider.ErrNotFound, "Bucket Not Found")
}

func (region *SRegion) GetIBucketByName(name string) (cloudprovider.ICloudBucket, error) {
	return region.GetIBucketById(name)
}

func (region *SRegion) DeleteIBucket(name string) error {
	svc := region.getS3Client()
	input := &s3.DeleteBucketInput{
		Bucket: &name,
	}
	_, err := svc.DeleteBucket(input)
	return err
}

func (region *SRegion) IBucketExist(name string) (bool, error) {
	svc := region.getS3Client()
	input := &s3.HeadBucketInput{
		Bucket: &name,
	}
	_, err := svc.HeadBucket(input)
	if err != nil {
		return false, err
	}
	return true, nil
}
