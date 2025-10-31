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

package baidu

import (
	"context"
	"net/http"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SObject struct {
	cloudprovider.SBaseCloudObject

	bucket *SBucket

	Key          string
	LastModified time.Time
	ETag         string
	Size         int64
	StorageClass string
	Owner        struct {
		Id          string
		DisplayName string
	}
}

func (o *SObject) GetKey() string {
	return o.Key
}

func (o *SObject) GetSizeBytes() int64 {
	return o.Size
}

func (o *SObject) GetLastModified() time.Time {
	return o.LastModified
}

func (o *SObject) GetStorageClass() string {
	return o.StorageClass
}

func (o *SObject) GetETag() string {
	return o.ETag
}

func (o *SObject) GetOwner() struct {
	Id          string
	DisplayName string
} {
	return o.Owner
}

func (o *SObject) GetAcl() cloudprovider.TBucketACLType {
	return cloudprovider.ACLPrivate
}

func (o *SObject) SetAcl(acl cloudprovider.TBucketACLType) error {
	return o.bucket.SetAcl(acl)
}

func (o *SObject) GetMeta() http.Header {
	return o.Meta
}

func (o *SObject) SetMeta(ctx context.Context, meta http.Header) error {
	return cloudprovider.ErrNotImplemented
}

func (o *SObject) GetIBucket() cloudprovider.ICloudBucket {
	return o.bucket
}
