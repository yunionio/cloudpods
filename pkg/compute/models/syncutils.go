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

package models

import (
	"context"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type IMetadataSetter interface {
	SetCloudMetadataAll(ctx context.Context, meta map[string]interface{}, userCred mcclient.TokenCredential) error
	Keyword() string
	GetName() string
	GetCloudproviderId() string
}

func syncMetadata(ctx context.Context, userCred mcclient.TokenCredential, model IMetadataSetter, remote cloudprovider.ICloudResource) error {
	metaData := remote.GetMetadata()
	if metaData != nil {
		meta := make(map[string]interface{}, 0)
		err := metaData.Unmarshal(meta)
		if err != nil {
			log.Errorf("Get VM Metadata error: %v", err)
			return err
		}
		store := make(map[string]interface{}, 0)
		for key, value := range meta {
			store[db.CLOUD_TAG_PREFIX+key] = value
		}
		// model.SetMetadata(ctx, "ext:"+key, value, userCred)
		// replace all ext keys
		model.SetCloudMetadataAll(ctx, store, userCred)
	}
	return nil
}

func syncVirtualResourceMetadata(ctx context.Context, userCred mcclient.TokenCredential, model IMetadataSetter, remote cloudprovider.IVirtualResource) error {
	metaData := remote.GetMetadata()
	store := make(map[string]interface{}, 0)
	if metaData != nil {
		meta := make(map[string]interface{}, 0)
		err := metaData.Unmarshal(meta)
		if err != nil {
			log.Errorf("Get VM Metadata error: %v", err)
			return err
		}
		for key, value := range meta {
			store[db.CLOUD_TAG_PREFIX+key] = value
		}

	}

	extProjectId := remote.GetProjectId()
	if len(extProjectId) > 0 {
		extProject, err := ExternalProjectManager.GetProject(extProjectId, model.GetCloudproviderId())
		if err != nil {
			log.Errorf("sync project metadata for %s %s error: %v", model.Keyword(), model.GetName(), err)
		} else {
			store[db.CLOUD_TAG_PREFIX+"project"] = extProject.Name
		}
	}

	if len(store) > 0 {
		model.SetCloudMetadataAll(ctx, store, userCred)
	}
	return nil
}

func SyncMetadata(ctx context.Context, userCred mcclient.TokenCredential, model IMetadataSetter, remote cloudprovider.ICloudResource) error {
	return syncMetadata(ctx, userCred, model, remote)
}

func SyncVirtualResourceMetadata(ctx context.Context, userCred mcclient.TokenCredential, model IMetadataSetter, remote cloudprovider.IVirtualResource) error {
	return syncVirtualResourceMetadata(ctx, userCred, model, remote)
}
