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
	"yunion.io/x/pkg/gotypes"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/yunionmeta"
)

// +onecloud:swagger-gen-ignore
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

// +onecloud:swagger-gen-ignore
type SCloudimage struct {
	db.SStandaloneResourceBase
	db.SExternalizedResourceBase

	SCloudregionResourceBase `width:"36" charset:"ascii" nullable:"false" list:"user" default:"default" create:"optional" json:"cloudregion_id" index:"true"`
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

	if len(regions) == 0 {
		return
	}

	meta, err := yunionmeta.FetchYunionmeta(ctx)
	if err != nil {
		log.Errorf("FetchYunionmeta %v", err)
		return
	}

	index, err := meta.Index(CloudimageManager.Keyword())
	if err != nil {
		log.Errorf("getServerSkuIndex error: %v", err)
		return
	}

	for i := range regions {
		region := &regions[i]

		skuMeta := &SCloudimage{}
		skuMeta.SetModelManager(CloudimageManager, skuMeta)
		skuMeta.Id = region.ExternalId

		oldMd5 := db.Metadata.GetStringValue(ctx, skuMeta, db.SKU_METADAT_KEY, userCred)
		newMd5, ok := index[region.ExternalId]
		if !ok || newMd5 == yunionmeta.EMPTY_MD5 || len(oldMd5) > 0 && newMd5 == oldMd5 {
			continue
		}

		db.Metadata.SetValue(ctx, skuMeta, db.SKU_METADAT_KEY, newMd5, userCred)

		err = regions[i].SyncCloudImages(ctx, userCred, !isStart, false)
		if err != nil {
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
		if errors.Cause(err) == sql.ErrNoRows {
			return self.Delete(ctx, userCred)
		}
		return errors.Wrapf(err, "db.FetchByExternalId(%s)", self.ExternalId)
	}
	image := _image.(*SCachedimage)

	err = image.ValidateDeleteCondition(ctx, nil)
	if err == nil {
		image.Delete(ctx, userCred)
	}

	return self.Delete(ctx, userCred)
}

func (self *SCloudimage) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.RealDeleteModel(ctx, userCred, self)
}

func (self *SCloudimage) syncWithImage(ctx context.Context, userCred mcclient.TokenCredential, image SCachedimage, region *SCloudregion) error {
	meta, err := yunionmeta.FetchYunionmeta(ctx)
	if err != nil {
		return err
	}

	skuUrl := region.getMetaUrl(meta.ImageBase, image.GetGlobalId())

	obj, err := db.FetchByExternalId(CachedimageManager, image.GetGlobalId())
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			return errors.Wrapf(err, "db.FetchByExternalId(%s)", image.GetGlobalId())
		}

		cachedImage := &SCachedimage{}
		cachedImage.SetModelManager(CachedimageManager, cachedImage)

		err = meta.Get(skuUrl, cachedImage)
		if err != nil {
			return errors.Wrapf(err, "Get")
		}

		cachedImage.IsPublic = true
		cachedImage.ProjectId = "system"
		err = CachedimageManager.TableSpec().Insert(ctx, cachedImage)
		if err != nil {
			return errors.Wrapf(err, "Insert cachedimage")
		}
		return nil
	}
	cachedImage := obj.(*SCachedimage)
	if gotypes.IsNil(cachedImage.Info) {
		err = meta.Get(skuUrl, &image)
		if err != nil {
			return errors.Wrapf(err, "Get")
		}
		_, err := db.Update(cachedImage, func() error {
			cachedImage.Info = image.Info
			cachedImage.Size = image.Size
			cachedImage.UEFI = image.UEFI
			return nil
		})
		return err
	}
	return nil
}
