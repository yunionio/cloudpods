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

package remotefile

import (
	"context"
	"io"
	"net/http"
	"time"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SBucket struct {
	SResourceBase

	region *SRegion

	RegionId     string
	MaxPart      int
	MaxPartBytes int64
	Acl          string
	Location     string
	StorageClass string
	AccessUrls   []cloudprovider.SBucketAccessUrl

	Stats cloudprovider.SBucketStats
	Limit cloudprovider.SBucketStats
}

func (self *SBucket) MaxPartCount() int {
	return self.MaxPart
}

func (self *SBucket) MaxPartSizeBytes() int64 {
	return self.MaxPartBytes
}

func (self *SBucket) GetAcl() cloudprovider.TBucketACLType {
	return cloudprovider.TBucketACLType(self.Acl)
}

func (self *SBucket) GetLocation() string {
	return self.Location
}

func (self *SBucket) GetIRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SBucket) GetStorageClass() string {
	return self.StorageClass
}

func (self *SBucket) GetAccessUrls() []cloudprovider.SBucketAccessUrl {
	return self.AccessUrls
}

func (self *SBucket) GetStats() cloudprovider.SBucketStats {
	return self.Stats
}

func (self *SBucket) GetLimit() cloudprovider.SBucketStats {
	return self.Limit
}

func (self *SBucket) SetLimit(limit cloudprovider.SBucketStats) error {
	return cloudprovider.ErrNotSupported
}

func (self *SBucket) LimitSupport() cloudprovider.SBucketStats {
	return self.Limit
}

func (self *SBucket) AbortMultipartUpload(ctx context.Context, key string, uploadId string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SBucket) SetAcl(acl cloudprovider.TBucketACLType) error {
	return cloudprovider.ErrNotSupported
}

func (self *SBucket) ListObjects(prefix string, marker string, delimiter string, maxCount int) (cloudprovider.SListObjectResult, error) {
	return cloudprovider.SListObjectResult{}, cloudprovider.ErrNotSupported
}

func (self *SBucket) CopyObject(ctx context.Context, destKey string, srcBucket, srcKey string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) error {
	return cloudprovider.ErrNotSupported
}

func (self *SBucket) GetObject(ctx context.Context, key string, rangeOpt *cloudprovider.SGetObjectRange) (io.ReadCloser, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SBucket) DeleteObject(ctx context.Context, keys string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SBucket) GetTempUrl(method string, key string, expire time.Duration) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SBucket) PutObject(ctx context.Context, key string, input io.Reader, sizeBytes int64, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) error {
	return cloudprovider.ErrNotSupported
}

func (self *SBucket) NewMultipartUpload(ctx context.Context, key string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SBucket) UploadPart(ctx context.Context, key string, uploadId string, partIndex int, input io.Reader, partSize int64, offset, totalSize int64) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SBucket) CopyPart(ctx context.Context, key string, uploadId string, partIndex int, srcBucketName string, srcKey string, srcOffset int64, srcLength int64) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SBucket) CompleteMultipartUpload(ctx context.Context, key string, uploadId string, partEtags []string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SBucket) SetWebsite(conf cloudprovider.SBucketWebsiteConf) error {
	return cloudprovider.ErrNotSupported
}

func (self *SBucket) GetWebsiteConf() (cloudprovider.SBucketWebsiteConf, error) {
	return cloudprovider.SBucketWebsiteConf{}, cloudprovider.ErrNotSupported
}

func (self *SBucket) DeleteWebSiteConf() error {
	return cloudprovider.ErrNotSupported
}

func (self *SBucket) SetCORS(rules []cloudprovider.SBucketCORSRule) error {
	return cloudprovider.ErrNotSupported
}

func (self *SBucket) GetCORSRules() ([]cloudprovider.SBucketCORSRule, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SBucket) DeleteCORS() error {
	return cloudprovider.ErrNotSupported
}

func (self *SBucket) SetReferer(conf cloudprovider.SBucketRefererConf) error {
	return cloudprovider.ErrNotSupported
}

func (self *SBucket) GetReferer() (cloudprovider.SBucketRefererConf, error) {
	return cloudprovider.SBucketRefererConf{}, cloudprovider.ErrNotSupported
}

func (self *SBucket) GetCdnDomains() ([]cloudprovider.SCdnDomain, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SBucket) GetPolicy() ([]cloudprovider.SBucketPolicyStatement, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SBucket) SetPolicy(policy cloudprovider.SBucketPolicyStatementInput) error {
	return cloudprovider.ErrNotSupported
}

func (self *SBucket) DeletePolicy(id []string) ([]cloudprovider.SBucketPolicyStatement, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SBucket) ListMultipartUploads() ([]cloudprovider.SBucketMultipartUploads, error) {
	return nil, cloudprovider.ErrNotSupported
}
