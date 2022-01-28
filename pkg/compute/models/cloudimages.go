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
	"database/sql"

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
	if isStart {
		cnt, err := CloudimageManager.Query().CountWithError()
		if err != nil {
			return
		}
		if cnt > 0 {
			log.Infof("Public cloud image has already synced, skip syncing")
			return
		}
	}
	regions := []SCloudregion{}
	q := CloudregionManager.Query().In("provider", CloudproviderManager.GetPublicProviderProvidersQuery())
	err := db.FetchModelObjects(CloudregionManager, q, &regions)
	if err != nil {
		return
	}

	meta, err := FetchSkuResourcesMeta()
	if err != nil {
		log.Errorf("SyncServerSkus.FetchSkuResourcesMeta %s", err)
		return
	}

	index, err := meta.getServerSkuIndex()
	if err != nil {
		log.Errorf("getServerSkuIndex error: %v", err)
		return
	}

	for i := range regions {
		region := &regions[i]
		oldMd5, _ := imageIndex[region.ExternalId]
		newMd5, ok := index[region.ExternalId]
		if ok {
			imageIndex[region.ExternalId] = newMd5
		}

		if newMd5 == EMPTY_MD5 {
			log.Infof("%s images is empty skip syncing", region.Name)
			continue
		}

		if len(oldMd5) > 0 && newMd5 == oldMd5 {
			log.Infof("%s cloud images not changed skip syncing", region.Name)
			continue
		}

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
				log.Errorf("SyncSystemImages for region %s(%s) storagecache %s error: %v", regions[i].Name, regions[i].Id, storagecaches[j].Name, err)
			}
		}
	}
	return
}

func (self *SCloudimage) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	_image, err := db.FetchByExternalId(CachedimageManager, self.ExternalId)
	if err != nil {
		return errors.Wrapf(err, "db.FetchByExternalId")
	}
	image := _image.(*SCachedimage)

	err = image.ValidateDeleteCondition(ctx, nil)
	if err == nil {
		image.Delete(ctx, userCred)
	}

	return self.Delete(ctx, userCred)
}

func (self *SCloudimage) syncWithImage(ctx context.Context, userCred mcclient.TokenCredential, image SCachedimage) error {
	_cachedImage, err := db.FetchByExternalId(CachedimageManager, image.GetGlobalId())
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			return errors.Wrapf(err, "db.FetchByExternalId(%s)", image.GetGlobalId())
		}
		image := &image
		image.Id = ""
		image.IsPublic = true
		image.ProjectId = "system"
		image.SetModelManager(CachedimageManager, image)
		err = CachedimageManager.TableSpec().Insert(ctx, image)
		if err != nil {
			return errors.Wrapf(err, "Insert cachedimage")
		}
		return nil
	}
	cachedImage := _cachedImage.(*SCachedimage)
	_, err = db.Update(cachedImage, func() error {
		cachedImage.Info = image.Info
		cachedImage.Size = image.Size
		cachedImage.UEFI = image.UEFI
		return nil
	})
	return err
}
