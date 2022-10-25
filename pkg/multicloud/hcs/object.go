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

package hcs

import (
	"context"
	"net/http"

	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
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
	obscli, err := o.bucket.region.getOBSClient()
	if err != nil {
		log.Errorf("o.bucket.region.GetOssClient error %s", err)
		return acl
	}
	input := &obs.GetObjectAclInput{}
	input.Bucket = o.bucket.Name
	input.Key = o.Key
	output, err := obscli.GetObjectAcl(input)
	if err != nil {
		log.Errorf("GetObjectAcl error: %v", err)
		return acl
	}
	acl = obsAcl2CannedAcl(output.Grants)
	return acl
}

func (o *SObject) SetAcl(aclStr cloudprovider.TBucketACLType) error {
	obscli, err := o.bucket.region.getOBSClient()
	if err != nil {
		return errors.Wrap(err, "o.bucket.region.getOBSClient")
	}
	input := &obs.SetObjectAclInput{}
	input.Bucket = o.bucket.Name
	input.Key = o.Key
	input.ACL = obs.AclType(string(aclStr))
	_, err = obscli.SetObjectAcl(input)
	if err != nil {
		return errors.Wrap(err, "obscli.SetObjectAcl")
	}
	return nil
}

func (o *SObject) GetMeta() http.Header {
	if o.Meta != nil {
		return o.Meta
	}
	obscli, err := o.bucket.region.getOBSClient()
	if err != nil {
		log.Errorf("getOBSClient fail %s", err)
		return nil
	}
	input := &obs.GetObjectMetadataInput{}
	input.Bucket = o.bucket.Name
	input.Key = o.Key
	output, err := obscli.GetObjectMetadata(input)
	if err != nil {
		log.Errorf("obscli.GetObjectMetadata fail %s", err)
		return nil
	}
	meta := http.Header{}
	for k, v := range output.Metadata {
		meta.Add(k, v)
	}
	if len(output.ContentType) > 0 {
		meta.Add(cloudprovider.META_HEADER_CONTENT_TYPE, output.ContentType)
	}
	o.Meta = meta
	return meta
}

func (o *SObject) SetMeta(ctx context.Context, meta http.Header) error {
	return cloudprovider.ObjectSetMeta(ctx, o.bucket, o, meta)
}
