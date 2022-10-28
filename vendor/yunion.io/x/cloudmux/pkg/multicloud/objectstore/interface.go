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

package objectstore

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/s3cli"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type IBucketProvider interface {
	cloudprovider.ICloudRegion

	NewBucket(bucket s3cli.BucketInfo) cloudprovider.ICloudBucket

	GetEndpoint() string

	S3Client() *s3cli.Client

	About() jsonutils.JSONObject
	GetVersion() string
	GetAccountId() string
	GetSubAccounts() ([]cloudprovider.SSubAccount, error)

	GetObjectAcl(bucket, key string) (cloudprovider.TBucketACLType, error)
	SetObjectAcl(bucket, key string, cannedAcl cloudprovider.TBucketACLType) error
	GetIBucketAcl(name string) (cloudprovider.TBucketACLType, error)
	SetIBucketAcl(name string, cannedAcl cloudprovider.TBucketACLType) error

	// GetCapabilities() []string
}
