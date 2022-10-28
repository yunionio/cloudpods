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

package google

import (
	"fmt"
	"net/url"

	"cloud.google.com/go/storage"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type GCSAcl struct {
	Kind        string
	Id          string
	SelfLink    string
	Bucket      string
	Entity      string
	Role        string
	Etag        string
	ProjectTeam map[string]string
}

func (region *SRegion) GetBucketAcl(bucket string) ([]GCSAcl, error) {
	resource := fmt.Sprintf("b/%s/acl", bucket)
	acls := []GCSAcl{}
	err := region.StorageListAll(resource, map[string]string{}, &acls)
	if err != nil {
		return nil, errors.Wrapf(err, "StorageListAll(%s)", resource)
	}
	return acls, nil
}

func (region *SRegion) SetObjectAcl(bucket, object string, cannedAcl cloudprovider.TBucketACLType) error {
	resource := fmt.Sprintf("b/%s/o/%s", bucket, url.PathEscape(object))
	acl := map[string]string{}
	switch cannedAcl {
	case cloudprovider.ACLPrivate:
		acls, err := region.GetObjectAcl(bucket, object)
		if err != nil {
			return errors.Wrap(err, "GetObjectAcl")
		}
		for _, _acl := range acls {
			if _acl.Entity == string(storage.AllUsers) || _acl.Entity == string(storage.AllAuthenticatedUsers) {
				resource := fmt.Sprintf("b/%s/o/%s/acl/%s", bucket, url.PathEscape(object), _acl.Entity)
				err = region.StorageDelete(resource)
				if err != nil {
					return errors.Wrapf(err, "StorageDelete(%s)", resource)
				}
			}
		}
		return nil
	case cloudprovider.ACLAuthRead:
		acl["entity"] = "allAuthenticatedUsers"
		acl["role"] = "READER"
	case cloudprovider.ACLPublicRead:
		acl["entity"] = "allUsers"
		acl["role"] = "READER"
	case cloudprovider.ACLPublicReadWrite:
		acl["entity"] = "allUsers"
		acl["role"] = "OWNER"
	}
	body := jsonutils.Marshal(acl)
	return region.StorageDo(resource, "acl", nil, body)
}

type BindingCondition struct {
	Title       string
	Description string
	Expression  string
}

type SBucketBinding struct {
	Role      string
	Members   []string
	Condition BindingCondition
}

type SBucketIam struct {
	Version    int
	Kind       string
	ResourceId string
	Bindings   []SBucketBinding
	Etag       string
}

func (region *SRegion) GetBucketIam(bucket string) (*SBucketIam, error) {
	resource := fmt.Sprintf("b/%s/iam", bucket)
	iam := SBucketIam{}
	err := region.StorageGet(resource, &iam)
	if err != nil {
		return nil, errors.Wrapf(err, "StorageListAll(%s)", resource)
	}
	return &iam, nil
}

func (region *SRegion) SetBucketIam(bucket string, iam *SBucketIam) (*SBucketIam, error) {
	resource := fmt.Sprintf("b/%s/iam", bucket)
	ret := SBucketIam{}
	err := region.StoragePut(resource, jsonutils.Marshal(iam), &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "StoragePut(%s)", resource)
	}
	return &ret, nil
}

func (region *SRegion) GetObjectAcl(bucket string, object string) ([]GCSAcl, error) {
	resource := fmt.Sprintf("b/%s/o/%s/acl", bucket, url.PathEscape(object))
	acls := []GCSAcl{}
	err := region.StorageListAll(resource, map[string]string{}, &acls)
	if err != nil {
		return nil, errors.Wrapf(err, "StorageListAll(%s)", resource)
	}
	return acls, nil
}
