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
	"context"
	"net/http"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/s3cli"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	META_HEADER = "X-Amz-Meta-"
)

type SObject struct {
	bucket *SBucket

	cloudprovider.SBaseCloudObject
}

func (o *SObject) GetIBucket() cloudprovider.ICloudBucket {
	return o.bucket
}

func (o *SObject) GetAcl() cloudprovider.TBucketACLType {
	acl, err := o.bucket.client.GetObjectAcl(o.bucket.Name, o.Key)
	if err != nil {
		if e, ok := errors.Cause(err).(s3cli.ErrorResponse); ok {
			if e.Code == "NoSuchKey" || e.Message == "The specified key does not exist." {
				objects, _ := o.bucket.ListObjects(o.Key, "", "", 1)
				if len(objects.Objects) > 0 {
					return cloudprovider.ACLPrivate
				}
			}
		}
		log.Errorf("o.bucket.client.GetObjectAcl error %s", err)
		return cloudprovider.ACLPrivate
	}
	return acl
}

func (o *SObject) SetAcl(aclStr cloudprovider.TBucketACLType) error {
	err := o.bucket.client.SetObjectAcl(o.bucket.Name, o.Key, aclStr)
	if err != nil {
		if strings.Contains(err.Error(), "not implemented") || strings.Contains(err.Error(), "Please use AWS4-HMAC-SHA256") {
			// ignore not implemented error
			return nil // cloudprovider.ErrNotImplemented
		} else {
			return errors.Wrap(err, "o.bucket.client.SetObjectAcl")
		}
	}
	return nil
}

func (o *SObject) GetMeta() http.Header {
	if o.Meta != nil {
		return o.Meta
	}
	cli := o.bucket.client.S3Client()
	objInfo, err := cli.StatObject(o.bucket.Name, o.Key, s3cli.StatObjectOptions{})
	if err != nil {
		log.Errorf("cli.statObject fail %s", err)
		return nil
	}
	if len(objInfo.ContentType) > 0 {
		objInfo.Metadata.Set(cloudprovider.META_HEADER_CONTENT_TYPE, objInfo.ContentType)
	}
	o.Meta = cloudprovider.FetchMetaFromHttpHeader(META_HEADER, objInfo.Metadata)
	return o.Meta
}

func (o *SObject) SetMeta(ctx context.Context, meta http.Header) error {
	return cloudprovider.ObjectSetMeta(ctx, o.bucket, o, meta)
}
