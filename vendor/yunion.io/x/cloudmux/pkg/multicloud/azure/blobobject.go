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

package azure

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SObject struct {
	container *SContainer

	cloudprovider.SBaseCloudObject
}

func (o *SObject) GetIBucket() cloudprovider.ICloudBucket {
	return o.container.storageaccount
}

func (o *SObject) GetAcl() cloudprovider.TBucketACLType {
	return o.container.getAcl()
}

func (o *SObject) SetAcl(aclStr cloudprovider.TBucketACLType) error {
	return cloudprovider.ErrNotSupported
}

func (o *SObject) getBlobName() string {
	if len(o.Key) <= len(o.container.Name)+1 {
		return ""
	} else {
		return o.Key[len(o.container.Name)+1:]
	}
}

func (sa *SStorageAccount) GetObjectMeta(object string) (http.Header, error) {
	accessKey, err := sa.GetAccountKey()
	if err != nil {
		return nil, errors.Wrap(err, "GetAccountKey")
	}

	ret := http.Header{}

	params := url.Values{}
	header, err := sa.region.header_storage_v2(accessKey, sa.Name, object, params)
	if err != nil {
		return nil, errors.Wrap(err, "header_storage_v2")
	}
	for k := range header {
		if strings.HasPrefix(strings.ToLower(k), "x-ms-meta-") {
			ret.Add(strings.TrimPrefix(strings.ToLower(k), "x-ms-meta-"), header.Get(k))
		}
		if utils.IsInStringArray(k, []string{
			http.CanonicalHeaderKey(cloudprovider.META_HEADER_CACHE_CONTROL),
			http.CanonicalHeaderKey(cloudprovider.META_HEADER_CONTENT_TYPE),
			http.CanonicalHeaderKey(cloudprovider.META_HEADER_CONTENT_DISPOSITION),
			http.CanonicalHeaderKey(cloudprovider.META_HEADER_CONTENT_ENCODING),
			http.CanonicalHeaderKey(cloudprovider.META_HEADER_CONTENT_LANGUAGE),
			http.CanonicalHeaderKey(cloudprovider.META_HEADER_CONTENT_MD5),
		}) {
			ret.Set(k, header.Get(k))
		}
	}
	return ret, nil
}

func (o *SObject) GetMeta() http.Header {
	if o.Meta != nil {
		return o.Meta
	}
	objectName := fmt.Sprintf("%s/%s", o.container.Name, o.getBlobName())
	var err error
	meta, err := o.container.storageaccount.GetObjectMeta(objectName)
	if err != nil {
		return nil
	}
	o.Meta = meta
	return meta
}

func (sa *SStorageAccount) SetObjectMeta(ctx context.Context, object string, meta http.Header) error {
	accessKey, err := sa.GetAccountKey()
	if err != nil {
		return errors.Wrap(err, "GetAccountKey")
	}

	properties := http.Header{}
	metadata := http.Header{}
	for k := range meta {
		if utils.IsInStringArray(k, []string{
			cloudprovider.META_HEADER_CACHE_CONTROL,
			cloudprovider.META_HEADER_CONTENT_TYPE,
			cloudprovider.META_HEADER_CONTENT_DISPOSITION,
			cloudprovider.META_HEADER_CONTENT_ENCODING,
			cloudprovider.META_HEADER_CONTENT_LANGUAGE,
			cloudprovider.META_HEADER_CONTENT_MD5,
		}) {
			properties.Set(fmt.Sprintf("x-ms-blob-%s", strings.ToLower(k)), meta.Get(k))
		} else {
			metadata.Set(fmt.Sprintf("x-ms-meta-%s", k), meta.Get(k))
		}
	}
	if len(properties) > 0 {
		params := url.Values{}
		params.Set("comp", "properties")
		err = sa.region.client.put_storage_v2(accessKey, sa.Name, object, properties, params, nil, nil)
		if err != nil {
			return errors.Wrap(err, "put_storage_v2 set properties")
		}
	}
	if len(metadata) > 0 {
		params := url.Values{}
		params.Set("comp", "metadata")
		err = sa.region.client.put_storage_v2(accessKey, sa.Name, object, metadata, params, nil, nil)
		if err != nil {
			return errors.Wrap(err, "put_storage_v2 set metadata")
		}
	}
	return nil
}

func (o *SObject) SetMeta(ctx context.Context, meta http.Header) error {
	fileName := fmt.Sprintf("%s/%s", o.container.Name, o.getBlobName())

	return o.container.storageaccount.SetObjectMeta(ctx, fileName, meta)
}
