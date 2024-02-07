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
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type IMetadataSetter interface {
	SetCloudMetadataAll(ctx context.Context, meta map[string]string, userCred mcclient.TokenCredential, readOnly bool) error
	SetSysCloudMetadataAll(ctx context.Context, meta map[string]string, userCred mcclient.TokenCredential, readOnly bool) error
	Keyword() string
	GetName() string
	GetCloudproviderId() string
}

type IVirtualResourceMetadataSetter interface {
	IMetadataSetter
	SetSystemInfo(isSystem bool) error
}

func syncMetadata(ctx context.Context, userCred mcclient.TokenCredential, model IMetadataSetter, remote cloudprovider.ICloudResource, readOnly bool) error {
	sysTags := remote.GetSysTags()
	sysStore := make(map[string]string, 0)
	for key, value := range sysTags {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		sysStore[db.SYS_CLOUD_TAG_PREFIX+key] = value
	}
	if options.Options.KeepTagLocalization {
		readOnly = true
	}
	model.SetSysCloudMetadataAll(ctx, sysStore, userCred, readOnly)

	tags, err := remote.GetTags()
	if err == nil {
		store := make(map[string]string, 0)
		for key, value := range tags {
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			store[db.CLOUD_TAG_PREFIX+key] = value
		}
		model.SetCloudMetadataAll(ctx, store, userCred, readOnly)
	}
	return nil
}

func syncVirtualResourceMetadata(ctx context.Context, userCred mcclient.TokenCredential, model IVirtualResourceMetadataSetter, remote cloudprovider.IVirtualResource, readOnly bool) error {
	sysTags := remote.GetSysTags()
	sysStore := make(map[string]string, 0)
	for key, value := range sysTags {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == apis.IS_SYSTEM && value == "true" {
			model.SetSystemInfo(true)
		}
		sysStore[db.SYS_CLOUD_TAG_PREFIX+key] = value
	}
	extProjectId := remote.GetProjectId()
	if len(extProjectId) > 0 {
		extProject, err := ExternalProjectManager.GetProject(extProjectId, model.GetCloudproviderId())
		if err != nil {
			log.Errorf("sync project metadata for %s %s error: %v", model.Keyword(), model.GetName(), err)
		} else {
			sysStore[db.SYS_CLOUD_TAG_PREFIX+"project"] = extProject.Name
		}
	}
	if options.Options.KeepTagLocalization {
		readOnly = true
	}

	model.SetSysCloudMetadataAll(ctx, sysStore, userCred, readOnly)

	tags, err := remote.GetTags()
	if err == nil {
		store := make(map[string]string, 0)
		for key, value := range tags {
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			store[db.CLOUD_TAG_PREFIX+key] = value
		}
		model.SetCloudMetadataAll(ctx, store, userCred, readOnly)
	}
	return nil
}

func SyncMetadata(ctx context.Context, userCred mcclient.TokenCredential, model IMetadataSetter, remote cloudprovider.ICloudResource, readOnly bool) error {
	return syncMetadata(ctx, userCred, model, remote, readOnly)
}

func SyncVirtualResourceMetadata(ctx context.Context, userCred mcclient.TokenCredential, model IVirtualResourceMetadataSetter, remote cloudprovider.IVirtualResource, readOnly bool) error {
	return syncVirtualResourceMetadata(ctx, userCred, model, remote, readOnly)
}
