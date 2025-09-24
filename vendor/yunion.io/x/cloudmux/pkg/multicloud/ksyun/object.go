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
	"net/http"

	"github.com/ks3sdklib/aws-sdk-go/service/s3"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"
)

type SObject struct {
	bucket *SBucket

	cloudprovider.SBaseCloudObject
}

func (o *SObject) GetIBucket() cloudprovider.ICloudBucket {
	return o.bucket
}

func (o *SObject) GetAcl() cloudprovider.TBucketACLType {
	svc := o.bucket.region.getS3Client()
	input := &s3.GetObjectACLInput{
		Bucket: &o.bucket.Name,
		Key:    &o.Key,
	}
	output, err := svc.GetObjectACL(input)
	if err != nil {
		return cloudprovider.ACLPrivate
	}
	return cloudprovider.TBucketACLType(s3.GetCannedACL(output.Grants))
}

func (o *SObject) SetAcl(acl cloudprovider.TBucketACLType) error {
	svc := o.bucket.region.getS3Client()
	aclStr := string(acl)
	input := &s3.PutObjectACLInput{
		Bucket: &o.bucket.Name,
		Key:    &o.Key,
		ACL:    &aclStr,
	}
	_, err := svc.PutObjectACL(input)
	if err != nil {
		return errors.Wrap(err, "PutObjectACL")
	}
	return nil
}

func (o *SObject) GetMeta() http.Header {
	svc := o.bucket.region.getS3Client()
	input := &s3.GetObjectInput{
		Bucket: &o.bucket.Name,
		Key:    &o.Key,
	}
	output, err := svc.GetObject(input)
	if err != nil {
		return nil
	}
	meta := http.Header{}
	for k, v := range output.Metadata {
		meta.Add(k, *v)
	}
	return meta
}

func (o *SObject) SetMeta(ctx context.Context, meta http.Header) error {
	return cloudprovider.ObjectSetMeta(ctx, o.bucket, o, meta)
}
