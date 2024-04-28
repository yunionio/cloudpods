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
	"reflect"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/esxi"
	"yunion.io/x/cloudmux/pkg/multicloud/esxi/vcenter"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	comapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
)

type SAgentImageCacheManager struct {
	imageCacheManger IImageCacheManger
}

func NewAgentImageCacheManager(manger IImageCacheManger) *SAgentImageCacheManager {
	return &SAgentImageCacheManager{manger}
}

type sImageCacheData struct {
	ImageId            string
	HostId             string
	HostIp             string
	SrcHostIp          string
	SrcPath            string
	SrcDatastore       vcenter.SVCenterAccessInfo
	Datastore          vcenter.SVCenterAccessInfo
	Format             string
	IsForce            bool
	StoragecacheId     string
	ImageType          string
	ImageExternalId    string
	StorageCacheHostIp string
}

func (c *SAgentImageCacheManager) PrefetchImageCache(ctx context.Context, data interface{}) (jsonutils.JSONObject, error) {
	dataDict, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, errors.Wrap(hostutils.ParamsError, "PrefetchImageCache data format error")
	}
	idata := new(sImageCacheData)
	err := dataDict.Unmarshal(idata)
	if err != nil {
		return nil, errors.Wrap(err, "%s: unmarshal to sImageCacheData error")
	}
	lockman.LockRawObject(ctx, idata.HostId, idata.ImageId)
	defer lockman.ReleaseRawObject(ctx, idata.HostId, idata.ImageId)
	if cloudprovider.TImageType(idata.ImageType) == cloudprovider.ImageTypeSystem {
		return c.perfetchTemplateVMImageCache(ctx, idata)
	}
	if len(idata.SrcHostIp) != 0 {
		return c.prefetchImageCacheByCopy(ctx, idata)
	}
	return c.prefetchImageCacheByUpload(ctx, idata, dataDict)
}

func (c *SAgentImageCacheManager) prefetchImageCacheByCopy(ctx context.Context, data *sImageCacheData) (jsonutils.JSONObject, error) {
	client, err := esxi.NewESXiClientFromAccessInfo(ctx, &data.Datastore)
	if err != nil {
		return nil, err
	}
	dstHost, err := client.FindHostByIp(data.HostIp)
	if err != nil {
		return nil, err
	}
	dstDs, err := dstHost.FindDataStoreById(data.Datastore.PrivateId)
	if err != nil {
		return nil, err
	}
	srcHost, err := client.FindHostByIp(data.SrcHostIp)
	if err != nil {
		return nil, err
	}
	srcDs, err := srcHost.FindDataStoreById(data.SrcDatastore.PrivateId)
	if err != nil {
		return nil, err
	}
	srcPath := data.SrcPath[len(srcDs.GetUrl()):]
	dstPath := fmt.Sprintf("image_cache/%s.vmdk", data.ImageId)

	// check if dst vmdk has been existed
	exists := false
	log.Infof("check file: src=%s, dst=%s", srcPath, dstPath)
	dstVmdkInfo, err := dstDs.GetVmdkInfo(ctx, dstPath)
	if err != nil && errors.Cause(err) != cloudprovider.ErrNotFound {
		return nil, err
	}
	srcVmdkInfo, err := srcDs.GetVmdkInfo(ctx, srcPath)
	if err != nil {
		return nil, err
	}
	if dstVmdkInfo != nil && reflect.DeepEqual(dstVmdkInfo, srcVmdkInfo) {
		exists = true
	}

	dstUrl := dstDs.GetPathUrl(dstPath)
	if !exists || data.IsForce {
		err = dstDs.CheckDirC(filepath.Dir(dstPath))
		if err != nil {
			return nil, errors.Wrap(err, "dstDs.MakeDir")
		}
		srcUrl := srcDs.GetPathUrl(srcPath)
		log.Infof("Copy %s => %s", srcUrl, dstUrl)
		err = client.CopyDisk(ctx, srcUrl, dstUrl, data.IsForce)
		if err != nil {
			return nil, errors.Wrap(err, "client.CopyDisk")
		}
		dstVmdkInfo, err = dstDs.GetVmdkInfo(ctx, dstPath)
		if err != nil {
			return nil, errors.Wrap(err, "dstDs.GetVmdkInfo")
		}
	}
	dstPath = dstDs.GetFullPath(dstPath)
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewInt(dstVmdkInfo.Size()), "size")
	ret.Add(jsonutils.NewString(dstPath), "path")
	ret.Add(jsonutils.NewString(data.ImageId), "image_id")
	_, err = hostutils.RemoteStoragecacheCacheImage(ctx, data.StoragecacheId, data.ImageId, comapi.CACHED_IMAGE_STATUS_ACTIVE, dstPath)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (c *SAgentImageCacheManager) prefetchImageCacheByUpload(ctx context.Context, data *sImageCacheData,
	origin *jsonutils.JSONDict) (jsonutils.JSONObject, error) {

	localImage, err := c.imageCacheManger.PrefetchImageCache(ctx, origin)
	if err != nil {
		return nil, err
	}
	localImgPath, _ := localImage.GetString("path")
	//localImgSize, _ := localImage.Int("size")

	client, err := esxi.NewESXiClientFromAccessInfo(ctx, &data.Datastore)
	if err != nil {
		return nil, errors.Wrap(err, "esxi.NewESXiClientFromJson")
	}
	host, err := client.FindHostByIp(data.HostIp)
	if err != nil {
		return nil, err
	}
	ds, err := host.FindDataStoreById(data.Datastore.PrivateId)
	if err != nil {
		return nil, errors.Wrap(err, "SHost.FindDataStoreById")
	}

	format, _ := origin.GetString("format")
	if format == "" {
		format = "vmdk"
	}
	remotePath := fmt.Sprintf("image_cache/%s.%s", data.ImageId, format)
	// check if dst image is exist
	exists := false
	if format == "vmdk" {
		err = ds.CheckVmdk(ctx, remotePath)
		if err != nil {
			log.Debugf("ds.CheckVmdk failed: %s", err)
		} else {
			exists = true
		}
	} else {
		_, err := ds.ListPath(ctx, remotePath)
		if err != nil {
			if errors.Cause(err) != errors.ErrNotFound {
				return nil, errors.Wrapf(err, "unable to check file with path %s", remotePath)
			}
		} else {
			exists = true
		}
	}
	log.Debugf("exist: %t, remotePath: %s", exists, remotePath)
	if !exists || data.IsForce {
		if format == "iso" {
			err := ds.ImportISO(ctx, localImgPath, remotePath)
			if err != nil {
				return nil, errors.Wrapf(err, "SDatastore.ImportISO %s -> %s", localImage, remotePath)
			}
		} else {
			err := ds.ImportVMDK(ctx, localImgPath, remotePath, host)
			if err != nil {
				return nil, errors.Wrap(err, "SDatastore.ImportTemplate")
			}
		}
	}
	remotePath = ds.GetFullPath(remotePath)
	remoteImg := localImage.(*jsonutils.JSONDict)
	remoteImg.Add(jsonutils.NewString(remotePath), "path")

	_, err = hostutils.RemoteStoragecacheCacheImage(ctx, data.StoragecacheId, data.ImageId, "active", remotePath)
	if err != nil {
		return nil, err
	}
	log.Debugf("prefetchImageCacheByUpload over")
	return remoteImg, nil
}

func (c *SAgentImageCacheManager) perfetchTemplateVMImageCache(ctx context.Context, data *sImageCacheData) (jsonutils.JSONObject, error) {
	client, err := esxi.NewESXiClientFromAccessInfo(ctx, &data.Datastore)
	if err != nil {
		return nil, errors.Wrap(err, "esxi.NewESXiClientFromJson")
	}
	_, err = client.SearchTemplateVM(data.ImageExternalId)
	if err != nil {
		return nil, errors.Wrapf(err, "SEsxiClient.SearchTemplateVM for image %q", data.ImageExternalId)
	}
	res := jsonutils.NewDict()
	res.Add(jsonutils.NewString(data.ImageExternalId), "image_id")
	return res, nil
}

func (c *SAgentImageCacheManager) DeleteImageCache(ctx context.Context, data interface{}) (jsonutils.JSONObject, error) {
	dataDict, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, errors.Wrap(hostutils.ParamsError, "DeleteImageCache data format error")
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
