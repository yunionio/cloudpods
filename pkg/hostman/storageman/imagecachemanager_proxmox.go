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

package storageman

import (
	"context"
	"os"
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/esxi/vcenter"
	"yunion.io/x/cloudmux/pkg/multicloud/proxmox"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
)

type SAgentProxmoxCacheManager struct {
	imageCacheManger IImageCacheManger
}

func NewAgentProxmoxCacheManager(manger IImageCacheManger) *SAgentProxmoxCacheManager {
	return &SAgentProxmoxCacheManager{manger}
}

func (c *SAgentProxmoxCacheManager) PrefetchProxmoxImageCache(ctx context.Context, data interface{}) (jsonutils.JSONObject, error) {
	input, ok := data.(*vcenter.ImageCacheInput)
	if !ok {
		return nil, errors.Wrap(hostutils.ParamsError, "PrefetchImageCache data format error")
	}
	lockman.LockRawObject(ctx, input.HostId, input.ImageId)
	defer lockman.ReleaseRawObject(ctx, input.HostId, input.ImageId)
	if cloudprovider.TImageType(input.ImageType) == cloudprovider.ImageTypeSystem {
		return c.prefetchProxmoxTemplateVMImageCache(ctx, input)
	}
	return c.prefetchProxmoxImageCacheByUpload(ctx, input)
}

func (c *SAgentProxmoxCacheManager) prefetchProxmoxTemplateVMImageCache(ctx context.Context, data *vcenter.ImageCacheInput) (jsonutils.JSONObject, error) {
	client, err := proxmox.NewProxmoxClientFromAccessInfo(&data.Datastore)
	if err != nil {
		return nil, errors.Wrap(err, "esxi.NewESXiClientFromJson")
	}
	images, err := client.GetTemplateImages("")
	if err != nil {
		return nil, errors.Wrapf(err, "GetClusterVmResources")
	}
	for i := range images {
		if images[i].GetGlobalId() == data.ImageExternalId {
			ret := jsonutils.NewDict()
			ret.Set("image_id", jsonutils.NewString(images[i].GetGlobalId()))
			return ret, nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "no template image found for %q", data.ImageExternalId)
}

func (c *SAgentProxmoxCacheManager) prefetchProxmoxImageCacheByUpload(ctx context.Context, input *vcenter.ImageCacheInput) (jsonutils.JSONObject, error) {
	data := api.CacheImageInput{}
	err := jsonutils.Update(&data, input)
	if err != nil {
		return nil, errors.Wrap(err, "jsonutils.Update")
	}
	localImage, err := c.imageCacheManger.PrefetchImageCache(ctx, data)
	if err != nil {
		return nil, errors.Wrapf(err, "PrefetchImageCache %s", input.ImageId)
	}

	localImgPath, _ := localImage.GetString("path")

	client, err := proxmox.NewProxmoxClientFromAccessInfo(&input.Datastore)
	if err != nil {
		return nil, errors.Wrap(err, "esxi.NewESXiClientFromJson")
	}
	host, err := client.GetHost(input.HostId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetHost %s", input.HostId)
	}

	storages, err := client.GetStoragesByHost(host.Node)
	if err != nil {
		return nil, errors.Wrapf(err, "GetStorage %s", input.StoragecacheId)
	}

	if len(input.Format) == 0 {
		input.Format = "qcow2"
	}

	content := "import"
	if input.Format == "iso" {
		content = "iso"
	}

	ret := jsonutils.NewDict()

	var storage *proxmox.SStorage = nil
	maxSize := int64(0)
	for i := range storages {
		if strings.Contains(storages[i].Content, content) {
			image, err := storages[i].SearchImage(content, input.ImageId, input.ImageName, input.Format)
			if err != nil {
				if errors.Cause(err) != cloudprovider.ErrNotFound {
					return nil, errors.Wrapf(err, "SearchImage %s", input.ImageId)
				}
				if storages[i].MaxDisk-storages[i].Disk > maxSize {
					maxSize = storages[i].MaxDisk - storages[i].Disk
					storage = &storages[i]
				}
				continue
			}
			ret.Set("path", jsonutils.NewString(image.Volid))
			ret.Set("name", jsonutils.NewString(image.GetName()))
			ret.Set("image_id", jsonutils.NewString(image.GetGlobalId()))
			ret.Set("size", jsonutils.NewInt(image.Size))
			return ret, nil
		}
	}
	if gotypes.IsNil(storage) {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "no valid storage found")
	}

	file, err := os.Open(localImgPath)
	if err != nil {
		return nil, errors.Wrapf(err, "Open %s", localImgPath)
	}
	defer file.Close()

	err = client.ImportImage(host.Node, storage.Storage, input.ImageId, input.Format, file)
	if err != nil {
		return nil, errors.Wrapf(err, "ImportImage %s", input.ImageId)
	}

	var image *proxmox.SImage = nil
	for i := 0; i < 3; i++ {
		image, err = storage.SearchImage(content, input.ImageId, input.ImageName, input.Format)
		if err != nil {
			if errors.Cause(err) != cloudprovider.ErrNotFound {
				return nil, errors.Wrapf(err, "SearchImage %s", input.ImageId)
			}
			time.Sleep(time.Second * 10)
			continue
		}
		break
	}

	if gotypes.IsNil(image) {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "search image %s %s %s %s after upload", content, input.ImageId, input.ImageName, input.Format)
	}

	ret.Set("name", jsonutils.NewString(image.GetName()))
	ret.Set("image_id", jsonutils.NewString(image.GetGlobalId()))
	ret.Set("size", jsonutils.NewInt(image.Size))
	return ret, nil
}
