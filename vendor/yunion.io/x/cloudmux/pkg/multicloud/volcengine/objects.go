// Copyright 2023 Yunion
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

package volcengine

import (
	"context"
	"net/http"

	"github.com/volcengine/ve-tos-golang-sdk/v2/tos"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos/enum"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

const (
	OSS_META_HEADER = "x-oss-meta-"
)

type SObject struct {
	bucket *SBucket

	cloudprovider.SBaseCloudObject
}

func (obj *SObject) GetIBucket() cloudprovider.ICloudBucket {
	return obj.bucket
}

func (obj *SObject) GetAcl() cloudprovider.TBucketACLType {
	acl := cloudprovider.ACLPrivate
	toscli, err := obj.bucket.region.GetTosClient()
	if err != nil {
		log.Errorf("Get Client %s", err)
		return acl
	}
	result, err := toscli.GetObjectACL(context.Background(), &tos.GetObjectACLInput{Bucket: obj.bucket.Name, Key: obj.Key})
	if err != nil {
		log.Errorf("GetObjectAcl %s", err)
	}
	grants := result.Grants
	return grantToCannedAcl(grants)
}

func (obj *SObject) SetAcl(aclStr cloudprovider.TBucketACLType) error {
	toscli, err := obj.bucket.region.GetTosClient()
	if err != nil {
		return errors.Wrap(err, "GetTosClient")
	}
	input := &tos.PutObjectACLInput{Bucket: obj.bucket.Name, Key: obj.Key, ACL: enum.ACLType(aclStr)}
	_, err = toscli.PutObjectACL(context.Background(), input)
	if err != nil {
		return errors.Wrapf(err, "PutObjectACL")
	}
	return nil
}

func (obj *SObject) GetMeta() http.Header {
	if obj.Meta != nil {
		return obj.Meta
	}
	toscli, err := obj.bucket.region.GetTosClient()
	if err != nil {
		log.Errorf("Get Client %s", err)
		return nil
	}
	result, err := toscli.GetObjectV2(context.Background(), &tos.GetObjectV2Input{Bucket: obj.bucket.Name, Key: obj.Key})
	if err != nil {
		log.Errorf("Get Object error %s", err)
		return nil
	}
	newHeader := http.Header{}
	meta := result.GetObjectBasicOutput.ObjectMetaV2.Meta
	for _, key := range meta.AllKeys() {
		value, exist := meta.Get(key)
		if !exist {
			log.Errorf("Key missing in meta data %s", key)
		} else {
			newHeader.Add(key, value)
		}
	}
	obj.Meta = cloudprovider.FetchMetaFromHttpHeader(
		OSS_META_HEADER,
		newHeader,
	)
	return obj.Meta
}

func (obj *SObject) SetMeta(ctx context.Context, meta http.Header) error {
	return cloudprovider.ObjectSetMeta(ctx, obj.bucket, obj, meta)
}
