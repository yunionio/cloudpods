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
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cloud.google.com/go/storage"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SBucketObjects struct {
	Prefixes      []string
	Items         []SObject
	NextPageToken string
}

type SObject struct {
	bucket *SBucket

	Id                      string
	Name                    string
	SelfLink                string
	MediaLink               string
	Bucket                  string
	Generation              string
	Metageneration          string
	ContentType             string
	ContentEncoding         string
	ContentDisposition      string
	ContentLanguage         string
	CacheControl            string
	StorageClass            string
	Size                    int64
	Md5Hash                 string
	Metadata                map[string]string
	Crc32c                  string
	Etag                    string
	TimeCreated             time.Time
	Updated                 time.Time
	TimeStorageClassUpdated time.Time
}

func (region *SRegion) GetObjects(bucket string, prefix string, nextPageToken string, delimiter string, maxCount int) (*SBucketObjects, error) {
	if maxCount <= 0 {
		maxCount = 20
	}
	resource := fmt.Sprintf("b/%s/o", bucket)
	params := map[string]string{"maxResults": fmt.Sprintf("%d", maxCount)}
	if len(prefix) > 0 {
		params["prefix"] = prefix
	}
	if len(delimiter) > 0 {
		params["delimiter"] = delimiter
	}
	if len(nextPageToken) > 0 {
		params["pageToken"] = nextPageToken
	}
	objs := SBucketObjects{}
	resp, err := region.client.storageList(resource, params)
	if err != nil {
		return nil, errors.Wrap(err, "storageList")
	}
	err = resp.Unmarshal(&objs)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	return &objs, nil
}

func (region *SRegion) ConvertAcl(acls []GCSAcl) cloudprovider.TBucketACLType {
	ret := cloudprovider.ACLPrivate
	for _, acl := range acls {
		if acl.Entity == string(storage.AllUsers) {
			if acl.Role == string(storage.RoleOwner) || acl.Role == string(storage.RoleWriter) {
				ret = cloudprovider.ACLPublicReadWrite
			}
			if acl.Role == string(storage.RoleReader) && ret != cloudprovider.ACLPublicReadWrite {
				ret = cloudprovider.ACLPublicRead
			}
		}
		if !(ret == cloudprovider.ACLPublicRead || ret == cloudprovider.ACLPublicReadWrite) && acl.Entity == string(storage.AllAuthenticatedUsers) {
			ret = cloudprovider.ACLAuthRead
		}
	}
	return ret
}

func (o *SObject) GetAcl() cloudprovider.TBucketACLType {
	if strings.HasSuffix(o.Name, "/") {
		return cloudprovider.ACLPrivate
	}
	acls, err := o.bucket.region.GetObjectAcl(o.bucket.Name, o.Name)
	if err != nil {
		log.Errorf("failed to get object %s acls error: %v", o.Name, err)
		return cloudprovider.ACLUnknown
	}
	return o.bucket.region.ConvertAcl(acls)
}

func (o *SObject) SetAcl(acl cloudprovider.TBucketACLType) error {
	return o.bucket.region.SetObjectAcl(o.bucket.Name, o.Name, acl)
}

func (o *SObject) GetIBucket() cloudprovider.ICloudBucket {
	return o.bucket
}

func (o *SObject) GetKey() string {
	return o.Name
}

func (o *SObject) GetSizeBytes() int64 {
	return o.Size
}

func (o *SObject) GetLastModified() time.Time {
	return o.Updated
}

func (o *SObject) GetStorageClass() string {
	return o.StorageClass
}

func (o *SObject) GetETag() string {
	return o.Etag
}

func (o *SObject) GetMeta() http.Header {
	meta := http.Header{}
	for k, v := range o.Metadata {
		meta.Set(k, v)
	}
	for k, v := range map[string]string{
		cloudprovider.META_HEADER_CONTENT_TYPE:        o.ContentType,
		cloudprovider.META_HEADER_CONTENT_ENCODING:    o.ContentEncoding,
		cloudprovider.META_HEADER_CONTENT_DISPOSITION: o.ContentDisposition,
		cloudprovider.META_HEADER_CONTENT_LANGUAGE:    o.ContentLanguage,
		cloudprovider.META_HEADER_CACHE_CONTROL:       o.CacheControl,
	} {
		meta.Set(k, v)
	}
	return meta
}

func (region *SRegion) SetObjectMeta(bucket, object string, meta http.Header) error {
	body := map[string]string{}
	for k := range meta {
		switch k {
		case cloudprovider.META_HEADER_CONTENT_TYPE:
			body["contentType"] = meta.Get(k)
		case cloudprovider.META_HEADER_CONTENT_ENCODING:
			body["contentEncoding"] = meta.Get(k)
		case cloudprovider.META_HEADER_CONTENT_DISPOSITION:
			body["contentDisposition"] = meta.Get(k)
		case cloudprovider.META_HEADER_CONTENT_LANGUAGE:
			body["contentLanguage"] = meta.Get(k)
		case cloudprovider.META_HEADER_CACHE_CONTROL:
			body["cacheControl"] = meta.Get(k)
		default:
			body[fmt.Sprintf("metadata.%s", k)] = meta.Get(k)
		}
	}
	resource := fmt.Sprintf("b/%s/o/%s", bucket, url.PathEscape(object))
	return region.StoragePut(resource, jsonutils.Marshal(body), nil)
}

func (o *SObject) SetMeta(ctx context.Context, meta http.Header) error {
	return o.bucket.region.SetObjectMeta(o.bucket.Name, o.Name, meta)
}
