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
	"net/http"

	"github.com/aws/aws-sdk-go/service/s3"

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
	s3cli, err := o.bucket.region.GetS3Client()
	if err != nil {
		log.Errorf("o.bucket.region.GetS3Client error %s", err)
		return acl
	}
	input := &s3.GetObjectAclInput{}
	input.SetBucket(o.bucket.Name)
	input.SetKey(o.Key)
	output, err := s3cli.GetObjectAcl(input)
	if err != nil {
		log.Errorf("s3cli.GetObjectAcl error %s", err)
		return acl
	}
	return s3ToCannedAcl(output.Grants)
}

func (o *SObject) SetAcl(aclStr cloudprovider.TBucketACLType) error {
	s3cli, err := o.bucket.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "o.bucket.region.GetS3Client")
	}
	input := &s3.PutObjectAclInput{}
	input.SetBucket(o.bucket.Name)
	input.SetKey(o.Key)
	input.SetACL(string(aclStr))
	_, err = s3cli.PutObjectAcl(input)
	if err != nil {
		return errors.Wrap(err, "s3cli.PutObjectAcl")
	}
	return nil
}

func (o *SObject) GetMeta() http.Header {
	if o.Meta != nil {
		return o.Meta
	}
	s3cli, err := o.bucket.region.GetS3Client()
	if err != nil {
		log.Errorf("o.bucket.region.GetS3Client fail %s", err)
		return nil
	}
	input := &s3.HeadObjectInput{}
	input.SetBucket(o.bucket.Name)
	input.SetKey(o.Key)
	output, err := s3cli.HeadObject(input)
	if err != nil {
		log.Errorf("s3cli.HeadObject fail %s", err)
		return nil
	}
	ret := http.Header{}
	for k, v := range output.Metadata {
		if v != nil && len(*v) > 0 {
			ret.Add(k, *v)
		}
	}
	if output.CacheControl != nil && len(*output.CacheControl) > 0 {
		ret.Set(cloudprovider.META_HEADER_CACHE_CONTROL, *output.CacheControl)
	}
	if output.ContentType != nil && len(*output.ContentType) > 0 {
		ret.Set(cloudprovider.META_HEADER_CONTENT_TYPE, *output.ContentType)
	}
	if output.ContentDisposition != nil && len(*output.ContentDisposition) > 0 {
		ret.Set(cloudprovider.META_HEADER_CONTENT_DISPOSITION, *output.ContentDisposition)
	}
	if output.ContentEncoding != nil && len(*output.ContentEncoding) > 0 {
		ret.Set(cloudprovider.META_HEADER_CONTENT_ENCODING, *output.ContentEncoding)
	}
	if output.ContentLanguage != nil && len(*output.ContentLanguage) > 0 {
		ret.Set(cloudprovider.META_HEADER_CONTENT_LANGUAGE, *output.ContentLanguage)
	}
	return ret
}

func (o *SObject) SetMeta(ctx context.Context, meta http.Header) error {
	return cloudprovider.ObjectSetMeta(ctx, o.bucket, o, meta)
}
