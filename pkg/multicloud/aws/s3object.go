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
	acl := cloudprovider.ACLDefault
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
