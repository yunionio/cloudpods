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
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
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
	acl, err := o.bucket.client.GetObjectAcl(o.bucket.Name, o.Key)
	if err != nil {
		log.Errorf("o.bucket.client.GetObjectAcl error %s", err)
		return acl
	}
	return acl
}

func (o *SObject) SetAcl(aclStr cloudprovider.TBucketACLType) error {
	err := o.bucket.client.SetObjectAcl(o.bucket.Name, o.Key, aclStr)
	if err != nil {
		if strings.Contains(err.Error(), "not implemented") {
			return cloudprovider.ErrNotImplemented
		} else {
			return errors.Wrap(err, "o.bucket.client.SetObjectAcl")
		}
	}
	return nil
}
