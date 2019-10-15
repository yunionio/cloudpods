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
	"fmt"
	"path/filepath"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/multicloud/esxi"
)

type SAgentImageCacheManager struct {
	imageCacheManger IImageCacheManger
}

func NewAgentImageCacheManager(manger IImageCacheManger) *SAgentImageCacheManager {
	return &SAgentImageCacheManager{manger}
}

func (c *SAgentImageCacheManager) PrefetchImageCache(ctx context.Context, data interface{}) (jsonutils.JSONObject, error) {
	dataDict, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, hostutils.ParamsError
	}
	var (
		imageID, _ = dataDict.GetString("image_id")
		HostID, _  = dataDict.GetString("host_id")
	)
	lockman.LockRawObject(ctx, HostID, imageID)
	defer lockman.LockRawObject(ctx, HostID, imageID)
	if dataDict.Contains("src_host_ip") {
		return c.prefetchImageCacheByCopy(ctx, dataDict)
	}
	return c.prefetchImageCacheByUpload(ctx, dataDict)
}

func (c *SAgentImageCacheManager) prefetchImageCacheByCopy(ctx context.Context, data *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	var (
		imageID, _        = data.GetString("image_id")
		dstHostIp, _      = data.GetString("host_ip")
		srcHostIp, _      = data.GetString("src_host_ip")
		dstDataStore, _   = data.Get("datastore")
		srcDataStore, _   = data.Get("src_datastore")
		storagecacheID, _ = data.GetString("storagecache_id")
		srcPath, _        = data.GetString("src_path")
		isForce           = jsonutils.QueryBoolean(data, "is_force", false)
	)
	client, _, err := esxi.NewESXiClientFromJson(ctx, dstDataStore)
	if err != nil {
		return nil, err
	}
	dstHost, err := client.FindHostByIp(dstHostIp)
	if err != nil {
		return nil, err
	}
	dstDsId, _ := dstDataStore.GetString("private_id")
	dstDs, err := dstHost.FindDataStoreById(dstDsId)
	if err != nil {
		return nil, err
	}
	srcHost, err := client.FindHostByIp(srcHostIp)
	if err != nil {
		return nil, err
	}
	srcDsId, _ := srcDataStore.GetString("private_id")
	srcDs, err := srcHost.FindDataStoreById(srcDsId)
	if err != nil {
		return nil, err
	}
	srcPath = srcPath[len(srcDs.GetUrl()):]
	dstPath := fmt.Sprintf("image_cache/%s.vmdk", imageID)

	// check if dst vmdk has been existed
	exists := false
	log.Infof("check file: src=%s, dst=%s", srcPath, dstPath)
	dstVmdkInfo, err := dstDs.GetVmdkInfo(ctx, dstPath)
	if err != nil {
		return nil, err
	}
	srcVmdkInfo, err := srcDs.GetVmdkInfo(ctx, srcPath)
	if err != nil {
		return nil, err
	}
	if dstVmdkInfo == srcVmdkInfo {
		exists = true
	}

	dstUrl := dstDs.GetPathUrl(dstPath)
	if !exists || isForce {
		_, err = dstDs.MakeDir(ctx, dstPath)
		if err != nil {
			return nil, err
		}
		srcUrl := srcDs.GetPathUrl(srcPath)
		log.Infof("Copy %s => %s", srcUrl, dstUrl)
		err = client.CopyDisk(ctx, srcUrl, dstUrl, isForce)
		if err != nil {
			return nil, err
		}
		dstVmdkInfo, err = dstDs.GetVmdkInfo(ctx, dstPath)
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewInt(dstVmdkInfo.Size()), "size")
	ret.Add(jsonutils.NewString(dstUrl), "path")
	ret.Add(jsonutils.NewString(imageID), "image_id")
	_, err = hostutils.RemoteStoragecacheCacheImage(ctx, storagecacheID, imageID, "ready", dstUrl)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (c *SAgentImageCacheManager) prefetchImageCacheByUpload(ctx context.Context, data *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	var (
		imageID, _        = data.GetString("image_id")
		hostIP, _         = data.GetString("host_ip")
		dsInfo, _         = data.Get("datastore")
		storagecacheID, _ = data.GetString("storagecache_id")
		isForce           = jsonutils.QueryBoolean(data, "is_force", false)
		format, _         = data.GetString("format")
	)

	data.Add(jsonutils.NewString("vmdk"), "format")
	localImage, err := c.imageCacheManger.PrefetchImageCache(ctx, data)
	if err != nil {
		return nil, err
	}
	localImgPath, _ := localImage.GetString("path")
	localImgSize, _ := localImage.Int("size")

	client, info, err := esxi.NewESXiClientFromJson(ctx, dsInfo)
	host, err := client.FindHostByIp(hostIP)
	if err != nil {
		return nil, err
	}
	ds, err := host.FindDataStoreById(info.PrivateId)
	remotePath := fmt.Sprintf("image_cache/%s.%s", imageID, format)

	// check if dst vmdk is exist
	exists := false
	if format == "vmdk" {
		err = ds.CheckVmdk(ctx, remotePath)
		if err != nil {
			return nil, err
		}
		exists = true
	} else {
		ret, err := ds.CheckFile(ctx, remotePath)
		if err != nil {
			return nil, err
		}
		if int64(ret.Size) == localImgSize {
			// exist and same size
			exists = true
		}
	}
	if !exists || isForce {
		err := ds.ImportTemplate(ctx, localImgPath, remotePath, host)
		if err != nil {
			return nil, errors.Wrap(err, "SDatastore.ImportTemplate")
		}
	}
	remotePath = filepath.Join(ds.GetUrl(), remotePath)
	remoteImg := localImage.(*jsonutils.JSONDict)
	remoteImg.Add(jsonutils.NewString(remotePath), "path")

	_, err = hostutils.RemoteStoragecacheCacheImage(ctx, storagecacheID, imageID, "ready", remotePath)
	if err != nil {
		return nil, err
	}
	return remoteImg, nil
}

func (c *SAgentImageCacheManager) DeleteImageCache(ctx context.Context, data interface{}) (jsonutils.JSONObject, error) {
	dataDict, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, hostutils.ParamsError
	}
	var (
		imageID, _ = dataDict.GetString("image_id")
		hostIP, _  = dataDict.GetString("host_ip")
		dsInfo, _  = dataDict.Get("ds_info")
	)

	client, _, err := esxi.NewESXiClientFromJson(ctx, dsInfo)
	if err != nil {
		return nil, err
	}
	host, err := client.FindHostByIp(hostIP)
	if err != nil {
		return nil, err
	}
	dsID, _ := dsInfo.GetString("private_id")
	ds, err := host.FindDataStoreById(dsID)
	if err != nil {
		return nil, err
	}
	remotePath := fmt.Sprintf("image_cache/%s.vmdk", imageID)
	return nil, ds.Delete(ctx, remotePath)
}
