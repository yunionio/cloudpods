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
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SCloudimageManager struct {
	db.SStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
}

var CloudimageManager *SCloudimageManager

func init() {
	CloudimageManager = &SCloudimageManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SCloudimage{},
			"cloudimages_tbl",
			"cloudimage",
			"cloudimages",
		),
	}
	CloudimageManager.SetVirtualObject(CloudimageManager)
}

type SCloudimage struct {
	db.SStandaloneResourceBase
	db.SExternalizedResourceBase

	SCloudregionResourceBase
}

func SyncPublicCloudImages(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	regions := []SCloudregion{}
	q := CloudregionManager.Query().In("provider", CloudproviderManager.GetPublicProviderProvidersQuery())
	err := db.FetchModelObjects(CloudregionManager, q, &regions)
	if err != nil {
		return
	}
	for i := range regions {
		err = regions[i].SyncCloudImages(ctx, userCred, !isStart)
		if err != nil {
			log.Errorf("SyncCloudImages for region %s(%s) error: %v", regions[i].Name, regions[i].Id, err)
			continue
		}
		storagecaches, err := regions[i].GetStoragecaches()
		if err != nil {
			log.Errorf("GetStoragecaches for region %s(%s) error: %v", regions[i].Name, regions[i].Id, err)
		}
		for j := range storagecaches {
			err = storagecaches[j].CheckCloudimages(ctx, userCred, regions[i].Name, regions[i].Id)
			if err != nil {
				log.Errorf("CheckCloudimages for region %s(%s) storagecache %s error: %v", regions[i].Name, regions[i].Id, storagecaches[j].Name, err)
			}
		}
	}
}

func (self *SCloudimage) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	_image, err := db.FetchByExternalId(CachedimageManager, self.ExternalId)
	if err != nil {
		return errors.Wrapf(err, "db.FetchByExternalId")
	}
	image := _image.(*SCachedimage)

	err = image.ValidateDeleteCondition(ctx)
	if err == nil {
		image.Delete(ctx, userCred)
	}

	return self.Delete(ctx, userCred)
}
