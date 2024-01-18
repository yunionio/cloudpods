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

package aws

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/fileutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/s3cli"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SBucket struct {
	multicloud.SBaseBucket
	AwsTags

	region *SRegion

	Name         string
	CreationDate time.Time
	Location     string

	acl cloudprovider.TBucketACLType
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

func (b *SBucket) GetCreatedAt() time.Time {
	return b.CreationDate
}

func (b *SBucket) GetStorageClass() string {
	return ""
}

func s3ToCannedAcl(acls []*s3.Grant) cloudprovider.TBucketACLType {
	switch {
	case len(acls) == 1:
		if acls[0].Grantee.URI == nil && *acls[0].Permission == s3cli.PERMISSION_FULL_CONTROL {
			return cloudprovider.ACLPrivate
		}
	case len(acls) == 2:
		for _, g := range acls {
			if *g.Grantee.Type == s3cli.GRANTEE_TYPE_GROUP && *g.Grantee.URI == s3cli.GRANTEE_GROUP_URI_AUTH_USERS && *g.Permission == s3cli.PERMISSION_READ {
				return cloudprovider.ACLAuthRead
			}
			if *g.Grantee.Type == s3cli.GRANTEE_TYPE_GROUP && *g.Grantee.URI == s3cli.GRANTEE_GROUP_URI_ALL_USERS && *g.Permission == s3cli.PERMISSION_READ {
				return cloudprovider.ACLPublicRead
			}
		}
	case len(acls) == 3:
		for _, g := range acls {
			if *g.Grantee.Type == s3cli.GRANTEE_TYPE_GROUP && *g.Grantee.URI == s3cli.GRANTEE_GROUP_URI_ALL_USERS && *g.Permission == s3cli.PERMISSION_WRITE {
				return cloudprovider.ACLPublicReadWrite
			}
		}
	}
	return cloudprovider.ACLUnknown
}

func (b *SBucket) GetAcl() cloudprovider.TBucketACLType {
	acl := cloudprovider.ACLPrivate
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		log.Errorf("b.region.GetS3Client fail %s", err)
		return acl
	}
	input := &s3.GetBucketAclInput{}
	input.SetBucket(b.Name)
	output, err := s3cli.GetBucketAcl(input)
	if err != nil {
		log.Errorf("s3cli.GetBucketAcl fail %s", err)
		return acl
	}
	return s3ToCannedAcl(output.Grants)
}

func (b *SBucket) SetAcl(aclStr cloudprovider.TBucketACLType) error {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "b.region.GetS3Client")
	}
	input := &s3.PutBucketAclInput{}
	input.SetBucket(b.Name)
	input.SetACL(string(aclStr))
	_, err = s3cli.PutBucketAcl(input)
	if err != nil {
		return errors.Wrap(err, "PutBucketAcl")
	}
	return nil
}

func (b *SBucket) GetAccessUrls() []cloudprovider.SBucketAccessUrl {
	return []cloudprovider.SBucketAccessUrl{
		{
			Url:         fmt.Sprintf("https://%s.%s", b.Name, b.region.getS3Endpoint()),
			Description: "bucket domain",
			Primary:     true,
		},
		{
			Url:         fmt.Sprintf("https://%s/%s", b.region.getS3Endpoint(), b.Name),
			Description: "s3 domain",
		},
	}
}

func (b *SBucket) GetWebsiteUrl() string {
	return fmt.Sprintf("http://%s.%s", b.Name, b.region.getS3WebsiteEndpoint())
}

func (b *SBucket) GetStats() cloudprovider.SBucketStats {
	stats, _ := cloudprovider.GetIBucketStats(b)
	return stats
}

func (b *SBucket) ListObjects(prefix string, marker string, delimiter string, maxCount int) (cloudprovider.SListObjectResult, error) {
	result := cloudprovider.SListObjectResult{}
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return result, errors.Wrap(err, "GetS3Client")
	}
	input := &s3.ListObjectsInput{}
	input.SetBucket(b.Name)
	if len(prefix) > 0 {
		input.SetPrefix(prefix)
	}
	if len(marker) > 0 {
		input.SetMarker(marker)
	}
	if len(delimiter) > 0 {
		input.SetDelimiter(delimiter)
	}
	if maxCount > 0 {
		input.SetMaxKeys(int64(maxCount))
	}
	oResult, err := s3cli.ListObjects(input)
	if err != nil {
		return result, errors.Wrap(err, "ListObjects")
	}
	result.Objects = make([]cloudprovider.ICloudObject, 0)
	for _, object := range oResult.Contents {
		obj := &SObject{
			bucket: b,
			SBaseCloudObject: cloudprovider.SBaseCloudObject{
				StorageClass: *object.StorageClass,
				Key:          *object.Key,
				SizeBytes:    *object.Size,
				ETag:         *object.ETag,
				LastModified: *object.LastModified,
			},
		}
		result.Objects = append(result.Objects, obj)
	}
	if oResult.CommonPrefixes != nil {
		result.CommonPrefixes = make([]cloudprovider.ICloudObject, len(oResult.CommonPrefixes))
		for i, commPrefix := range oResult.CommonPrefixes {
			result.CommonPrefixes[i] = &SObject{
				bucket:           b,
				SBaseCloudObject: cloudprovider.SBaseCloudObject{Key: *commPrefix.Prefix},
			}
		}
	}
	if oResult.IsTruncated != nil {
		result.IsTruncated = *oResult.IsTruncated
	}
	if oResult.NextMarker != nil {
		result.NextMarker = *oResult.NextMarker
	}
	return result, nil
}

func (b *SBucket) PutObject(ctx context.Context, key string, body io.Reader, sizeBytes int64, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) error {
	if sizeBytes < 0 {
		return errors.Error("content length expected")
	}
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "GetS3Client")
	}
	input := &s3.PutObjectInput{}
	input.SetBucket(b.Name)
	input.SetKey(key)
	seeker, err := fileutils.NewReadSeeker(body, sizeBytes)
	if err != nil {
		return errors.Wrap(err, "newFakeSeeker")
	}
	defer seeker.Close()
	input.SetBody(seeker)
	input.SetContentLength(sizeBytes)
	if meta != nil {
		metaHdr := make(map[string]*string)
		for k, v := range meta {
			if len(v) == 0 || len(v[0]) == 0 {
				continue
			}
			switch http.CanonicalHeaderKey(k) {
			case cloudprovider.META_HEADER_CACHE_CONTROL:
				input.SetCacheControl(v[0])
			case cloudprovider.META_HEADER_CONTENT_TYPE:
				input.SetContentType(v[0])
			case cloudprovider.META_HEADER_CONTENT_MD5:
				input.SetContentMD5(v[0])
			case cloudprovider.META_HEADER_CONTENT_LANGUAGE:
				input.SetContentLanguage(v[0])
			case cloudprovider.META_HEADER_CONTENT_ENCODING:
				input.SetContentEncoding(v[0])
			case cloudprovider.META_HEADER_CONTENT_DISPOSITION:
				input.SetContentDisposition(v[0])
			default:
				metaHdr[k] = &v[0]
			}
		}
		if len(metaHdr) > 0 {
			input.SetMetadata(metaHdr)
		}
	}
	if len(cannedAcl) == 0 {
		cannedAcl = b.GetAcl()
	}
	input.SetACL(string(cannedAcl))
	if len(storageClassStr) > 0 {
		input.SetStorageClass(storageClassStr)
	}
	_, err = s3cli.PutObjectWithContext(ctx, input)
	if err != nil {
		return errors.Wrap(err, "PutObjectWithContext")
	}
	return nil
}

func (b *SBucket) NewMultipartUpload(ctx context.Context, key string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) (string, error) {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return "", errors.Wrap(err, "GetS3Client")
	}
	input := &s3.CreateMultipartUploadInput{}
	input.SetBucket(b.Name)
	input.SetKey(key)
	if meta != nil {
		metaHdr := make(map[string]*string)
		for k, v := range meta {
			if len(v) == 0 || len(v[0]) == 0 {
				continue
			}
			switch http.CanonicalHeaderKey(k) {
			case cloudprovider.META_HEADER_CACHE_CONTROL:
				input.SetCacheControl(v[0])
			case cloudprovider.META_HEADER_CONTENT_TYPE:
				input.SetContentType(v[0])
			case cloudprovider.META_HEADER_CONTENT_LANGUAGE:
				input.SetContentLanguage(v[0])
			case cloudprovider.META_HEADER_CONTENT_ENCODING:
				input.SetContentEncoding(v[0])
			case cloudprovider.META_HEADER_CONTENT_DISPOSITION:
				input.SetContentDisposition(v[0])
			default:
				metaHdr[k] = &v[0]
			}
		}
		if len(metaHdr) > 0 {
			input.SetMetadata(metaHdr)
		}
	}
	if len(cannedAcl) == 0 {
		cannedAcl = b.GetAcl()
	}
	input.SetACL(string(cannedAcl))
	if len(storageClassStr) > 0 {
		input.SetStorageClass(storageClassStr)
	}
	output, err := s3cli.CreateMultipartUploadWithContext(ctx, input)
	if err != nil {
		return "", errors.Wrap(err, "CreateMultipartUpload")
	}
	return *output.UploadId, nil
}

func (b *SBucket) UploadPart(ctx context.Context, key string, uploadId string, partIndex int, part io.Reader, partSize int64, offset, totalSize int64) (string, error) {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return "", errors.Wrap(err, "GetS3Client")
	}
	input := &s3.UploadPartInput{}
	input.SetBucket(b.Name)
	input.SetKey(key)
	input.SetUploadId(uploadId)
	input.SetPartNumber(int64(partIndex))
	seeker, err := fileutils.NewReadSeeker(part, partSize)
	if err != nil {
		return "", errors.Wrap(err, "newFakeSeeker")
	}
	defer seeker.Close()
	input.SetBody(seeker)
	input.SetContentLength(partSize)
	output, err := s3cli.UploadPartWithContext(ctx, input)
	if err != nil {
		return "", errors.Wrap(err, "UploadPartWithContext")
	}
	return *output.ETag, nil
}

func (b *SBucket) CompleteMultipartUpload(ctx context.Context, key string, uploadId string, partEtags []string) error {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "GetS3Client")
	}
	input := &s3.CompleteMultipartUploadInput{}
	input.SetBucket(b.Name)
	input.SetKey(key)
	input.SetUploadId(uploadId)
	uploads := &s3.CompletedMultipartUpload{}
	parts := make([]*s3.CompletedPart, len(partEtags))
	for i := range partEtags {
		parts[i] = &s3.CompletedPart{}
		parts[i].SetPartNumber(int64(i + 1))
		parts[i].SetETag(partEtags[i])
	}
	uploads.SetParts(parts)
	input.SetMultipartUpload(uploads)
	_, err = s3cli.CompleteMultipartUploadWithContext(ctx, input)
	if err != nil {
		return errors.Wrap(err, "CompleteMultipartUploadWithContext")
	}
	return nil
}

func (b *SBucket) AbortMultipartUpload(ctx context.Context, key string, uploadId string) error {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "GetS3Client")
	}
	input := &s3.AbortMultipartUploadInput{}
	input.SetBucket(b.Name)
	input.SetKey(key)
	input.SetUploadId(uploadId)
	_, err = s3cli.AbortMultipartUploadWithContext(ctx, input)
	if err != nil {
		return errors.Wrap(err, "AbortMultipartUploadWithContext")
	}
	return nil
}

func (b *SBucket) DeleteObject(ctx context.Context, key string) error {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "GetS3Client")
	}
	input := &s3.DeleteObjectInput{}
	input.SetBucket(b.Name)
	input.SetKey(key)
	_, err = s3cli.DeleteObjectWithContext(ctx, input)
	if err != nil {
		return errors.Wrap(err, "DeleteObject")
	}
	return nil
}

func (b *SBucket) GetTempUrl(method string, key string, expire time.Duration) (string, error) {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return "", errors.Wrap(err, "GetS3Client")
	}
	var request *request.Request
	switch method {
	case "GET":
		input := &s3.GetObjectInput{}
		input.SetBucket(b.Name)
		input.SetKey(key)
		request, _ = s3cli.GetObjectRequest(input)
	case "PUT":
		input := &s3.PutObjectInput{}
		input.SetBucket(b.Name)
		input.SetKey(key)
		request, _ = s3cli.PutObjectRequest(input)
	case "DELETE":
		input := &s3.DeleteObjectInput{}
		input.SetBucket(b.Name)
		input.SetKey(key)
		request, _ = s3cli.DeleteObjectRequest(input)
	default:
		return "", errors.Error("unsupported method")
	}
	url, _, err := request.PresignRequest(expire)
	if err != nil {
		return "", errors.Wrap(err, "request.PresignRequest")
	}
	return url, nil
}

func (b *SBucket) CopyObject(ctx context.Context, destKey string, srcBucket, srcKey string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) error {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "GetS3Client")
	}
	log.Debugf("copy from %s/%s to %s/%s", srcBucket, srcKey, b.Name, destKey)
	input := &s3.CopyObjectInput{}
	input.SetBucket(b.Name)
	input.SetKey(destKey)
	input.SetCopySource(fmt.Sprintf("%s/%s", srcBucket, url.PathEscape(srcKey)))
	input.SetStorageClass(storageClassStr)
	if len(cannedAcl) == 0 {
		cannedAcl = b.GetAcl()
	}
	input.SetACL(string(cannedAcl))
	var metaDir string
	if meta != nil {
		metaHdr := make(map[string]*string)
		for k, v := range meta {
			if len(v) == 0 || len(v[0]) == 0 {
				continue
			}
			switch http.CanonicalHeaderKey(k) {
			case cloudprovider.META_HEADER_CACHE_CONTROL:
				input.SetCacheControl(v[0])
			case cloudprovider.META_HEADER_CONTENT_TYPE:
				input.SetContentType(v[0])
			case cloudprovider.META_HEADER_CONTENT_LANGUAGE:
				input.SetContentLanguage(v[0])
			case cloudprovider.META_HEADER_CONTENT_ENCODING:
				input.SetContentEncoding(v[0])
			case cloudprovider.META_HEADER_CONTENT_DISPOSITION:
				input.SetContentDisposition(v[0])
			default:
				metaHdr[k] = &v[0]
			}
		}
		if len(metaHdr) > 0 {
			input.SetMetadata(metaHdr)
		}
		metaDir = "REPLACE"
	} else {
		metaDir = "COPY"
	}
	input.SetMetadataDirective(metaDir)
	_, err = s3cli.CopyObject(input)
	if err != nil {
		return errors.Wrap(err, "CopyObject")
	}
	return nil
}

func (b *SBucket) GetObject(ctx context.Context, key string, rangeOpt *cloudprovider.SGetObjectRange) (io.ReadCloser, error) {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return nil, errors.Wrap(err, "GetS3Client")
	}
	input := &s3.GetObjectInput{}
	input.SetBucket(b.Name)
	input.SetKey(key)
	if rangeOpt != nil {
		input.SetRange(rangeOpt.String())
	}
	output, err := s3cli.GetObject(input)
	if err != nil {
		return nil, errors.Wrap(err, "GetObject")
	}
	return output.Body, nil
}

func (b *SBucket) CopyPart(ctx context.Context, key string, uploadId string, partNumber int, srcBucket string, srcKey string, srcOffset int64, srcLength int64) (string, error) {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return "", errors.Wrap(err, "GetS3Client")
	}
	input := &s3.UploadPartCopyInput{}
	input.SetBucket(b.Name)
	input.SetKey(key)
	input.SetUploadId(uploadId)
	input.SetPartNumber(int64(partNumber))
	input.SetCopySource(fmt.Sprintf("/%s/%s", srcBucket, url.PathEscape(srcKey)))
	if srcLength > 0 {
		input.SetCopySourceRange(fmt.Sprintf("bytes=%d-%d", srcOffset, srcOffset+srcLength-1))
	}
	output, err := s3cli.UploadPartCopy(input)
	if err != nil {
		return "", errors.Wrap(err, "s3cli.UploadPartCopy")
	}
	return *output.CopyPartResult.ETag, nil
}

func (b *SBucket) SetWebsite(websitConf cloudprovider.SBucketWebsiteConf) error {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "GetS3Client")
	}
	s3WebConf := s3.WebsiteConfiguration{}
	s3WebConf.SetIndexDocument(&s3.IndexDocument{Suffix: &websitConf.Index})
	s3WebConf.SetErrorDocument(&s3.ErrorDocument{Key: &websitConf.ErrorDocument})
	input := s3.PutBucketWebsiteInput{}
	input.SetBucket(b.Name)
	input.SetWebsiteConfiguration(&s3WebConf)
	_, err = s3cli.PutBucketWebsite(&input)
	if err != nil {
		return errors.Wrapf(err, "s3cli.PutBucketWebsite(%s)", jsonutils.Marshal(input).String())
	}
	return nil
}

func (b *SBucket) GetWebsiteConf() (cloudprovider.SBucketWebsiteConf, error) {
	result := cloudprovider.SBucketWebsiteConf{}
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return result, errors.Wrap(err, "GetS3Client")
	}
	input := s3.GetBucketWebsiteInput{}
	input.SetBucket(b.Name)
	webconfResult, err := s3cli.GetBucketWebsite(&input)
	if err != nil {
		return result, errors.Wrapf(err, "s3cli.GetBucketWebsite(%s)", b.Name)
	}

	if webconfResult.IndexDocument != nil && webconfResult.IndexDocument.Suffix != nil {
		result.Index = *webconfResult.IndexDocument.Suffix
	}
	if webconfResult.ErrorDocument != nil && webconfResult.ErrorDocument.Key != nil {
		result.ErrorDocument = *webconfResult.ErrorDocument.Key
	}
	result.Url = b.GetWebsiteUrl()
	return result, nil
}

func (b *SBucket) DeleteWebSiteConf() error {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "GetS3Client")
	}
	input := s3.DeleteBucketWebsiteInput{}
	input.SetBucket(b.Name)
	_, err = s3cli.DeleteBucketWebsite(&input)
	if err != nil {
		return errors.Wrapf(err, "s3cli.DeleteBucketWebsite(%s)", b.Name)
	}
	return nil
}

func InputToAwsApiSliceString(input []string) []*string {
	result := []*string{}
	for i := range input {
		result = append(result, &input[i])
	}
	return result
}

func InputToAwsApiInt64(input int64) *int64 {
	return &input
}

func AwsApiSliceStringToOutput(input []*string) []string {
	result := []string{}
	for i := range input {
		if input[i] != nil {
			result = append(result, *input[i])
		} else {
			result = append(result, "")
		}
	}
	return result
}

func AwsApiInt64ToOutput(input *int64) int64 {
	if input == nil {
		return 0
	}
	return *input
}

func (b *SBucket) SetCORS(rules []cloudprovider.SBucketCORSRule) error {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "GetS3Client")
	}
	opts := []*s3.CORSRule{}
	for i := range rules {
		opts = append(opts, &s3.CORSRule{
			AllowedOrigins: InputToAwsApiSliceString(rules[i].AllowedOrigins),
			AllowedMethods: InputToAwsApiSliceString(rules[i].AllowedMethods),
			AllowedHeaders: InputToAwsApiSliceString(rules[i].AllowedHeaders),
			MaxAgeSeconds:  InputToAwsApiInt64(int64(rules[i].MaxAgeSeconds)),
			ExposeHeaders:  InputToAwsApiSliceString(rules[i].ExposeHeaders),
		})
	}

	input := s3.PutBucketCorsInput{}
	input.SetBucket(b.Name)
	input.SetCORSConfiguration(&s3.CORSConfiguration{CORSRules: opts})
	_, err = s3cli.PutBucketCors(&input)
	if err != nil {
		return errors.Wrapf(err, "s3cli.PutBucketCors(%s)", input)
	}
	return nil
}

func (b *SBucket) GetCORSRules() ([]cloudprovider.SBucketCORSRule, error) {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return nil, errors.Wrap(err, "GetS3Client")
	}
	input := s3.GetBucketCorsInput{}
	input.SetBucket(b.Name)
	conf, err := s3cli.GetBucketCors(&input)
	if err != nil {
		if !strings.Contains(err.Error(), "NoSuchCORSConfiguration") {
			return nil, errors.Wrapf(err, "s3cli.GetBucketCors(%s)", b.Name)
		}
	}
	if conf == nil {
		return []cloudprovider.SBucketCORSRule{}, nil
	}
	result := []cloudprovider.SBucketCORSRule{}
	for i := range conf.CORSRules {
		result = append(result, cloudprovider.SBucketCORSRule{
			AllowedOrigins: AwsApiSliceStringToOutput(conf.CORSRules[i].AllowedOrigins),
			AllowedMethods: AwsApiSliceStringToOutput(conf.CORSRules[i].AllowedMethods),
			AllowedHeaders: AwsApiSliceStringToOutput(conf.CORSRules[i].AllowedHeaders),
			MaxAgeSeconds:  int(AwsApiInt64ToOutput(conf.CORSRules[i].MaxAgeSeconds)),
			ExposeHeaders:  AwsApiSliceStringToOutput(conf.CORSRules[i].ExposeHeaders),
			Id:             strconv.Itoa(i),
		})
	}
	return result, nil
}

func (b *SBucket) DeleteCORS() error {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "GetS3Client")
	}

	input := s3.DeleteBucketCorsInput{}
	input.SetBucket(b.Name)
	_, err = s3cli.DeleteBucketCors(&input)
	if err != nil {
		return errors.Wrapf(err, "s3cli.DeleteBucketCors(%s)", b.Name)
	}
	return nil
}

func (b *SBucket) GetTags() (map[string]string, error) {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return nil, errors.Wrap(err, "GetS3Client")
	}
	tagresult, err := s3cli.GetBucketTagging(&s3.GetBucketTaggingInput{Bucket: &b.Name})
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchTagSet") {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "osscli.GetBucketTagging(%s)", b.Name)
	}
	if tagresult == nil {
		return nil, nil
	}
	result := map[string]string{}
	for i := range tagresult.TagSet {
		if tagresult.TagSet[i].Key != nil && tagresult.TagSet[i].Value != nil {
			result[*tagresult.TagSet[i].Key] = *tagresult.TagSet[i].Value
		}

	}
	return result, nil
}

func (b *SBucket) SetTags(tags map[string]string, replace bool) error {
	if !replace {
		return cloudprovider.ErrNotSupported
	}
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "GetS3Client")
	}

	_, err = s3cli.DeleteBucketTagging(&s3.DeleteBucketTaggingInput{Bucket: &b.Name})
	if err != nil {
		return errors.Wrapf(err, "DeleteBucketTagging")
	}

	if len(tags) == 0 {
		return nil
	}

	input := s3.PutBucketTaggingInput{Tagging: &s3.Tagging{}}
	input.Bucket = &b.Name
	apiTagKeys := []string{}
	apiTagValues := []string{}
	for k, v := range tags {
		apiTagKeys = append(apiTagKeys, k)
		apiTagValues = append(apiTagValues, v)

	}
	for i := range apiTagKeys {
		input.Tagging.TagSet = append(input.Tagging.TagSet, &s3.Tag{Key: &apiTagKeys[i], Value: &apiTagValues[i]})
	}

	_, err = s3cli.PutBucketTagging(&input)
	if err != nil {
		return errors.Wrapf(err, "obscli.SetBucketTagging(%s)", jsonutils.Marshal(input))
	}
	return nil
}

func (b *SBucket) ListMultipartUploads() ([]cloudprovider.SBucketMultipartUploads, error) {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return nil, errors.Wrap(err, "GetS3Client")
	}
	result := []cloudprovider.SBucketMultipartUploads{}

	input := s3.ListMultipartUploadsInput{}
	input.SetBucket(b.Name)
	keyMarker := ""
	uploadIDMarker := ""
	for {
		if len(keyMarker) > 0 {
			input.SetKeyMarker(keyMarker)
		}
		if len(uploadIDMarker) > 0 {
			input.SetUploadIdMarker(uploadIDMarker)
		}
		output, err := s3cli.ListMultipartUploads(&input)
		if err != nil {
			return nil, errors.Wrap(err, " coscli.Bucket.ListMultipartUploads(context.Background(), &input)")
		}
		if output == nil {
			return nil, nil
		}
		for i := range output.Uploads {
			temp := cloudprovider.SBucketMultipartUploads{}
			if output.Uploads[i].Key != nil {
				temp.ObjectName = *output.Uploads[i].Key
			}
			if output.Uploads[i].Initiator != nil {
				temp.Initiator = *output.Uploads[i].Initiator.DisplayName
			}
			if output.Uploads[i].Initiated != nil {
				temp.Initiated = *output.Uploads[i].Initiated
			}
			if output.Uploads[i].UploadId != nil {
				temp.UploadID = *output.Uploads[i].UploadId
			}
			result = append(result, temp)
		}
		if output.NextKeyMarker != nil {
			keyMarker = *output.NextKeyMarker
		}
		if output.NextUploadIdMarker != nil {
			uploadIDMarker = *output.NextUploadIdMarker
		}

		if output.IsTruncated == nil || !*output.IsTruncated {
			break
		}
	}

	return result, nil
}

type SBucketPolicyStatement struct {
	Version   string                          `json:"Version"`
	Id        string                          `json:"Id"`
	Statement []SBucketPolicyStatementDetails `json:"Statement"`
}

type SBucketPolicyStatementDetails struct {
	Sid       string                            `json:"Sid"`
	Principal map[string][]string               `json:"Principal"`
	Action    []string                          `json:"Action"`
	Resource  []string                          `json:"Resource"`
	Effect    string                            `json:"Effect"`
	Condition map[string]map[string]interface{} `json:"Condition"`
}

func (b *SBucket) GetPolicy() ([]cloudprovider.SBucketPolicyStatement, error) {
	res := []cloudprovider.SBucketPolicyStatement{}
	policies, err := b.getPolicy()
	if err != nil {
		if errors.Cause(err) == errors.ErrNotFound {
			return res, nil
		}
		return nil, errors.Wrap(err, "get policy")
	}
	for i, policy := range policies {
		temp := cloudprovider.SBucketPolicyStatement{}
		temp.Action = policy.Action
		temp.Principal = policy.Principal
		temp.PrincipalId = getLocalPrincipalId(policy.Principal["AWS"])
		temp.PrincipalNames = getLocalPrincipalNames(policy.Principal["AWS"])
		temp.Effect = policy.Effect
		temp.Resource = policy.Resource
		temp.ResourcePath = policy.Resource
		temp.CannedAction = b.actionToCannedAction(policy.Action)
		temp.Id = fmt.Sprintf("%d", i)
		temp.Condition = policy.Condition
		res = append(res, temp)
	}
	return res, nil
}

func (b *SBucket) getPolicy() ([]SBucketPolicyStatementDetails, error) {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return nil, errors.Wrap(err, "GetS3Client")
	}
	input := s3.GetBucketPolicyInput{}
	input.SetBucket(b.Name)
	conf, err := s3cli.GetBucketPolicy(&input)
	if err != nil {
		if !strings.Contains(err.Error(), "NoSuch") {
			return nil, errors.Wrapf(err, "s3cli.GetBucketCors(%s)", b.Name)
		}
	}
	if conf == nil {
		return []SBucketPolicyStatementDetails{}, nil
	}
	if conf.Policy == nil {
		return nil, errors.ErrNotFound
	}
	obj, err := jsonutils.Parse([]byte(*conf.Policy))
	if err != nil {
		return nil, errors.Wrap(err, "parse policy")
	}
	policies := []SBucketPolicyStatementDetails{}
	err = obj.Unmarshal(&policies, "Statement")
	if err != nil {
		return nil, errors.Wrap(err, "Statement")
	}
	err = obj.Unmarshal(&policies, "Statement")
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}
	return policies, nil
}

func (b *SBucket) SetPolicy(policy cloudprovider.SBucketPolicyStatementInput) error {
	old, err := b.getPolicy()
	if err != nil && err != errors.ErrNotFound {
		return errors.Wrap(err, "getPolicy")
	}
	if old == nil {
		old = []SBucketPolicyStatementDetails{}
	}
	ids := []string{}
	for i := range policy.PrincipalId {
		id := strings.Split(policy.PrincipalId[i], ":")
		if len(id) == 1 {
			ids = append(ids, id...)
		}
		if len(id) == 2 {
			// 没有主账号id,设为owner id
			if len(id[0]) == 0 {
				id[0] = b.region.client.GetAccountId()
			}
			// 没有子账号，默认和主账号相同
			if len(id[1]) == 0 {
				id[1] = "*"
			}
			ids = append(ids, id[1])
		}
		if len(id) > 2 {
			return errors.Wrap(cloudprovider.ErrNotSupported, "Invalida PrincipalId Input")
		}
	}

	old = append(old, SBucketPolicyStatementDetails{
		Effect:   policy.Effect,
		Action:   b.getAction(policy.CannedAction),
		Resource: b.getResources(policy.ResourcePath),
		Sid:      utils.GenRequestId(20),
		Principal: map[string][]string{
			// "AWS": b.getPrincipal(policy.PrincipalId, policy.AccountId),
			"AWS": ids,
		},
		Condition: policy.Condition,
	})
	return b.setPolicy(old)
}

func (b *SBucket) setPolicy(policies []SBucketPolicyStatementDetails) error {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "GetS3Client")
	}
	input := s3.PutBucketPolicyInput{}
	input.Bucket = &b.Name
	// example
	// test := `{"Statement":[{"Action":["s3:GetBucketAcl"],"Effect":"Allow","Principal":{"Service":["config.amazonaws.com"]},"Resource":["arn:aws-cn:s3:::config-bucket-2xxxxx6"],"Sid":"AWSConfigBucketPermissionsCheck"},{"Action":["s3:PutObject"],"Effect":"Allow","Principal":{"Service":["config.amazonaws.com"]},"Resource":["arn:aws-cn:s3:::config-bucket-2xxxxx6/AWSLogs/2xxxxx6/Config/*"],"Sid":"AWSConfigBucketDelivery"},{"Action":["s3:PutObject"],"Effect":"Allow","Principal":{"Service":["config.amazonaws.com"]},"Resource":["arn:aws-cn:s3:::config-bucket-2xxxxx6/AWSLogs/2xxxxx6/Config/*"],"Sid":"test"}]}`
	param := SBucketPolicyStatement{}
	param.Statement = policies

	policyStr := jsonutils.Marshal(param).String()
	input.Policy = &policyStr
	_, err = s3cli.PutBucketPolicy(&input)
	if err != nil {
		return errors.Wrap(err, "PutBucketPolicy")
	}
	return nil
}

func (b *SBucket) DeletePolicy(id []string) ([]cloudprovider.SBucketPolicyStatement, error) {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return nil, errors.Wrap(err, "GetS3Client")
	}

	policies, err := b.getPolicy()
	if err != nil {
		return nil, errors.Wrap(err, "GetPolicy")
	}
	needKeep := []SBucketPolicyStatementDetails{}
	for i, policy := range policies {
		if utils.IsInStringArray(fmt.Sprintf("%d", i), id) {
			continue
		}
		needKeep = append(needKeep, policy)
	}
	_, err = s3cli.DeleteBucketPolicy(&s3.DeleteBucketPolicyInput{Bucket: &b.Name})
	if err != nil {
		return nil, errors.Wrap(err, "DeleteBucketPolicy")
	}

	if len(needKeep) > 0 {
		err = b.setPolicy(needKeep)
		if err != nil {
			return nil, errors.Wrap(err, "setPolicy")
		}
	}
	res := []cloudprovider.SBucketPolicyStatement{}
	for _, policy := range needKeep {
		temp := cloudprovider.SBucketPolicyStatement{}
		temp.Action = policy.Action
		temp.Principal = policy.Principal
		temp.Effect = policy.Effect
		temp.Resource = policy.Resource
		temp.ResourcePath = policy.Resource
		temp.CannedAction = b.actionToCannedAction(policy.Action)
		temp.Id = policy.Sid
		temp.Condition = policy.Condition
		res = append(res, temp)
	}
	return res, nil
}

func (b *SBucket) getResources(paths []string) []string {
	res := []string{}
	for _, path := range paths {
		res = append(res, fmt.Sprintf("arn:%s:s3:::%s%s", b.region.GetARNPartition(), b.Name, path))
	}
	return res
}

func (b *SBucket) getPrincipal(principalIds []string, accountId string) []string {
	res := []string{}
	for _, id := range principalIds {
		res = append(res, fmt.Sprintf("arn:%s:iam::%s:user/%s", b.region.GetARNPartition(), accountId, id))
	}
	return res
}

func (b *SBucket) getAwsAction(actions []string) []string {
	res := []string{}
	for _, action := range actions {
		res = append(res, fmt.Sprintf("s3:%s", action))
	}
	return res
}

var readActions = []string{
	"s3:Get*",
	"s3:List*",
}

var readWriteActions = []string{
	"s3:Get*",
	"s3:List*",
	"s3:Create*",
	"s3:Put*",
	"s3:Delete*",
	"s3:Create*",
	"s3:AbortMultipartUpload",
}

var fullControlActions = []string{
	"s3:*",
}

func (b *SBucket) getAction(s string) []string {
	switch s {
	case "Read":
		return readActions
	case "ReadWrite":
		return readWriteActions
	case "FullControl":
		return fullControlActions
	default:
		return nil
	}
}

func (b *SBucket) actionToCannedAction(actions []string) string {
	if len(actions) == len(readActions) {
		for _, action := range actions {
			if !utils.IsInStringArray(action, readActions) {
				return ""
			}
		}
		return "Read"
	} else if len(actions) == len(fullControlActions) {
		for _, action := range actions {
			if !utils.IsInStringArray(action, fullControlActions) {
				return ""
			}
		}
		return "FullControl"
	} else if len(actions) == len(readWriteActions) {
		for _, action := range actions {
			if !utils.IsInStringArray(action, readWriteActions) {
				return ""
			}
		}
		return "ReadWrite"
	}
	return ""
}

/*
	example: in:arn:aws-cn:iam::248697896586:user/yunion-test
			out:[248697896586:yunion-test]
*/

func getLocalPrincipalId(principals []string) []string {
	res := []string{}
	for _, principal := range principals {
		temp := strings.Split(principal, "::")
		temp1 := strings.Split(temp[1], ":user/")
		if len(temp1) > 1 {
			if temp1[1] == "*" {
				temp1[1] = temp1[0]
			}
			res = append(res, fmt.Sprintf("%s:%s", temp1[0], temp1[1]))
		} else {
			res = append(res, temp[1])
		}
	}
	return res
}

/*
	example: in:arn:aws-cn:iam::248697896586:user/yunion-test
			 out:["248697896586:yunion-test":"yunion-test"]
*/

func getLocalPrincipalNames(principals []string) map[string]string {
	res := map[string]string{}
	for _, principal := range principals {
		temp := strings.Split(principal, "::")
		temp1 := strings.Split(temp[1], ":user/")
		if len(temp1) > 1 {
			if temp1[1] == "*" {
				temp1[1] = temp1[0]
			}
			res[fmt.Sprintf("%s:%s", temp1[0], temp1[1])] = temp1[1]
		}
	}
	return res
}
