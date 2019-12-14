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
	"net/http"

	"github.com/Azure/azure-sdk-for-go/storage"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
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
	return nil
}

func (o *SObject) getBlobName() string {
	if len(o.Key) <= len(o.container.Name)+1 {
		return ""
	} else {
		return o.Key[len(o.container.Name)+1:]
	}
}

func (o *SObject) getBlobRef() (*storage.Blob, error) {
	blobName := o.getBlobName()
	if len(blobName) == 0 {
		return nil, nil
	}
	contRef, err := o.container.getContainerRef()
	if err != nil {
		return nil, errors.Wrap(err, "src getContainerRef")
	}
	blobRef := contRef.GetBlobReference(blobName)
	return blobRef, nil
}

func (o *SObject) GetMeta() http.Header {
	if o.Meta != nil {
		return o.Meta
	}
	blobRef, err := o.getBlobRef()
	if err != nil {
		log.Errorf("o.getBlobRef fail %s", err)
		return nil
	}
	if blobRef == nil {
		return nil
	}
	err = blobRef.GetMetadata(nil)
	if err != nil {
		log.Errorf("blobRef.GetMetadata fail %s", err)
	}
	err = blobRef.GetProperties(nil)
	if err != nil {
		log.Errorf("blobRef.GetProperties fail %s", err)
	}
	meta := getBlobRefMeta(blobRef)
	o.Meta = meta
	return o.Meta
}

func (o *SObject) SetMeta(ctx context.Context, meta http.Header) error {
	blobRef, err := o.getBlobRef()
	if err != nil {
		return errors.Wrap(err, "o.getBlobRef")
	}
	if blobRef == nil {
		return cloudprovider.ErrNotSupported
	}
	propChanged, metaChanged := setBlobRefMeta(blobRef, meta)
	if propChanged {
		propOpts := storage.SetBlobPropertiesOptions{}
		err := blobRef.SetProperties(&propOpts)
		if err != nil {
			return errors.Wrap(err, "blob.SetProperties")
		}
	}
	if metaChanged {
		metaOpts := storage.SetBlobMetadataOptions{}
		err := blobRef.SetMetadata(&metaOpts)
		if err != nil {
			return errors.Wrap(err, "blob.SetMetadata")
		}
	}
	return nil
}
