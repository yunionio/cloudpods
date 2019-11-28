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

package google

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SLifecycleRuleAction struct {
	Type string
}

type SLifecycleRuleCondition struct {
	Age int
}

type SLifecycleRule struct {
	Action    SLifecycleRuleAction
	Condition SLifecycleRuleCondition
}

type SBucketPolicyOnly struct {
	Enabled bool
}

type SUniformBucketLevelAccess struct {
	Enabled bool
}

type SIamConfiguration struct {
	BucketPolicyOnly         SBucketPolicyOnly
	UniformBucketLevelAccess SUniformBucketLevelAccess
}

type SLifecycle struct {
	Rule []SLifecycleRule
}

type SBucket struct {
	Kind             string
	SelfLink         string
	Id               string
	Name             string
	ProjectNumber    string
	Metageneration   string
	Location         string
	StorageClass     string
	Etag             string
	TimeCreated      time.Time
	Updated          time.Time
	Lifecycle        SLifecycle
	IamConfiguration SIamConfiguration
	LocationType     string
}

func (region *SRegion) GetBucket(name string) (*SBucket, error) {
	resource := "b/" + name
	bucket := &SBucket{}
	err := region.StorageGet(resource, bucket)
	if err != nil {
		return nil, errors.Wrap(err, "GetBucket")
	}
	return bucket, nil
}

func (region *SRegion) GetBuckets(maxResults int, pageToken string) ([]SBucket, error) {
	buckets := []SBucket{}
	params := map[string]string{
		"project": region.GetProjectId(),
	}
	err := region.StorageList("b", params, maxResults, pageToken, &buckets)
	if err != nil {
		return nil, err
	}
	return buckets, nil
}

func (region *SRegion) CreateBucket(name string, storageClass string) (*SBucket, error) {
	body := map[string]interface{}{
		"name":     name,
		"location": region.Name,
	}
	if len(storageClass) > 0 {
		body["storageClass"] = storageClass
	}
	params := url.Values{}
	params.Set("project", region.GetProjectId())
	bucket := &SBucket{}
	err := region.StorageInsert(fmt.Sprintf("b?%s", params.Encode()), jsonutils.Marshal(body), bucket)
	if err != nil {
		return nil, err
	}
	return bucket, nil
}

func (region *SRegion) UploadObject(bucket string, params url.Values, header http.Header, input io.Reader) error {
	resource := fmt.Sprintf("b/%s/o", bucket)
	if len(params) > 0 {
		resource = fmt.Sprintf("%s?%s", resource, params.Encode())
	}
	return region.client.storageUpload(resource, header, input)
}

func (region *SRegion) PutObject(bucket string, name string, input io.Reader, contType string, sizeBytes int64, cannedAcl cloudprovider.TBucketACLType) error {
	params := url.Values{}
	params.Set("name", name)
	params.Set("uploadType", "media")
	switch cannedAcl {
	case cloudprovider.ACLPrivate:
		params.Set("predefinedAcl", "private")
	case cloudprovider.ACLAuthRead:
		params.Set("predefinedAcl", "authenticatedRead")
	case cloudprovider.ACLPublicRead:
		params.Set("predefinedAcl", "publicRead")
	case cloudprovider.ACLPublicReadWrite:
		return cloudprovider.ErrNotSupported
	}

	header := http.Header{}
	header.Set("Content-Length", fmt.Sprintf("%v", sizeBytes))
	header.Set("Content-Type", "application/octet-stream")

	return region.UploadObject(bucket, params, header, input)
}

func (region *SRegion) DeleteBucket(name string) error {
	return region.StorageDelete("b/" + name)
}
