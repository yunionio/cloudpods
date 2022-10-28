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

package aliyun

import (
	"context"
	"net/http"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	OSS_META_HEADER = "x-oss-meta-"
)

type SObject struct {
	bucket *SBucket

	cloudprovider.SBaseCloudObject
}

func (o *SObject) GetIBucket() cloudprovider.ICloudBucket {
	return o.bucket
}

func (o *SObject) GetAcl() cloudprovider.TBucketACLType {
	acl := cloudprovider.ACLPrivate
	osscli, err := o.bucket.region.GetOssClient()
	if err != nil {
		log.Errorf("o.bucket.region.GetOssClient error %s", err)
		return acl
	}
	bucket, err := osscli.Bucket(o.bucket.Name)
	if err != nil {
		log.Errorf("osscli.Bucket error %s", err)
		return acl
	}
	result, err := bucket.GetObjectACL(o.Key)
	if err != nil {
		log.Errorf("bucket.GetObjectACL error %s", err)
		return acl
	}
	if result.ACL == string(oss.ACLDefault) {
		return o.bucket.GetAcl()
	}
	acl = cloudprovider.TBucketACLType(result.ACL)
	return acl
}

func (o *SObject) SetAcl(aclStr cloudprovider.TBucketACLType) error {
	acl, err := str2Acl(string(aclStr))
	if err != nil {
		return errors.Wrap(err, "str2Acl")
	}
	osscli, err := o.bucket.region.GetOssClient()
	if err != nil {
		return errors.Wrap(err, "o.bucket.region.GetOssClient")
	}
	bucket, err := osscli.Bucket(o.bucket.Name)
	if err != nil {
		return errors.Wrap(err, "osscli.Bucket")
	}
	err = bucket.SetObjectACL(o.Key, acl)
	if err != nil {
		return errors.Wrap(err, "bucket.SetObjectACL")
	}
	return nil
}

func (o *SObject) GetMeta() http.Header {
	if o.Meta != nil {
		return o.Meta
	}
	osscli, err := o.bucket.region.GetOssClient()
	if err != nil {
		log.Errorf("o.bucket.region.GetOssClient error %s", err)
		return nil
	}
	bucket, err := osscli.Bucket(o.bucket.Name)
	if err != nil {
		log.Errorf("osscli.Bucket error %s", err)
		return nil
	}
	result, err := bucket.GetObjectDetailedMeta(o.Key)
	if err != nil {
		log.Errorf("bucket.GetObjectACL error %s", err)
		return nil
	}
	o.Meta = cloudprovider.FetchMetaFromHttpHeader(OSS_META_HEADER, result)
	return o.Meta
}

func (o *SObject) SetMeta(ctx context.Context, meta http.Header) error {
	return cloudprovider.ObjectSetMeta(ctx, o.bucket, o, meta)
}
