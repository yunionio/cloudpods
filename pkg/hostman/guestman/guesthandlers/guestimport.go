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

package guesthandlers

import (
	"context"
	"net/http"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/hostman/guestman"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

const libvirtMountPath = "/opt/cloud/libvirt"

func guestPrepareImportFormLibvirt(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, _, body := appsrv.FetchEnv(ctx, w, r)
	config := &compute.SLibvirtHostConfig{}
	err := body.Unmarshal(config)
	if err != nil {
		hostutils.Response(ctx, w, httperrors.NewInputParameterError("Parse params to libvirt config error %s", err))
		return
	}
	if len(config.XmlFilePath) == 0 {
		hostutils.Response(ctx, w, httperrors.NewMissingParameterError("xml_file_path"))
		return
	}
	err = procutils.NewRemoteCommandAsFarAsPossible("test", "-d", config.XmlFilePath).Run()
	if err != nil {
		hostutils.Response(ctx, w,
			httperrors.NewBadRequestError("check xml_file_path %s failed: %s", config.XmlFilePath, err))
		return
	}

	if len(config.Servers) == 0 {
		hostutils.Response(ctx, w, httperrors.NewMissingParameterError("servers"))
		return
	}

	if len(config.MonitorPath) > 0 {
		err = procutils.NewRemoteCommandAsFarAsPossible("test", "-d", config.MonitorPath).Run()
		if err != nil {
			hostutils.Response(ctx, w, httperrors.NewBadRequestError(
				"check monitor_path %s failed: %s", config.MonitorPath, err,
			))
			return
		}

		// monitor path may not exist in container
		err = procutils.NewRemoteCommandAsFarAsPossible("mkdir", "-p", libvirtMountPath).Run()
		if err != nil {
			hostutils.Response(ctx, w, httperrors.NewBadRequestError(
				"mkdir libvirt cloud path %s failed: %s", libvirtMountPath, err,
			))
			return
		}

		if err = procutils.NewRemoteCommandAsFarAsPossible("mountpoint", libvirtMountPath).Run(); err != nil {
			out, err := procutils.NewRemoteCommandAsFarAsPossible(
				"mount", "--bind", config.MonitorPath, libvirtMountPath,
			).Output()
			if err != nil {
				hostutils.Response(ctx, w, httperrors.NewBadRequestError(
					"monitor path create symbolic link failed: %s", out,
				))
				return
			}
		}
		config.MonitorPath = libvirtMountPath
	}

	hostutils.DelayTask(ctx, guestman.GetGuestManager().PrepareImportFromLibvirt, config)
	hostutils.ResponseOk(ctx, w)
}

func guestCreateFromLibvirt(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
	err := guestman.GetGuestManager().PrepareCreate(sid)
	if err != nil {
		return nil, err
	}

	var guestDesc = new(desc.SGuestDesc)
	err = body.Unmarshal(guestDesc, "desc")
	if err != nil {
		return nil, httperrors.NewBadRequestError("Guest desc unmarshal failed %s", err)
	}

	iDisksPath, err := body.Get("disks_path")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("disks_path")
	}
	disksPath, ok := iDisksPath.(*jsonutils.JSONDict)
	if !ok {
		return nil, httperrors.NewInputParameterError("disks_path is not dict")
	}

	monitorPath, _ := body.GetString("monitor_path")
	if len(monitorPath) > 0 && !fileutils2.Exists(monitorPath) {
		return nil, httperrors.NewBadRequestError("Monitor path %s not found", monitorPath)
	}

	hostutils.DelayTask(ctx, guestman.GetGuestManager().GuestCreateFromLibvirt,
		&guestman.SGuestCreateFromLibvirt{
			Sid:         sid,
			MonitorPath: monitorPath,
			GuestDesc:   guestDesc,
			DisksPath:   disksPath,
		})
	return nil, nil
}

func guestCreateFromEsxi(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
	err := guestman.GetGuestManager().PrepareCreate(sid)
	if err != nil {
		return nil, err
	}

	var guestDesc = new(desc.SGuestDesc)
	err = body.Unmarshal(guestDesc, "desc")
	if err != nil {
		return nil, httperrors.NewBadRequestError("Guest desc unmarshal failed %s", err)
	}
	var disksAccessInfo = guestman.SEsxiAccessInfo{}
	err = body.Unmarshal(&disksAccessInfo, "esxi_access_info")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("esxi_access_info")
	}
	hostutils.DelayTask(ctx, guestman.GetGuestManager().GuestCreateFromEsxi,
		&guestman.SGuestCreateFromEsxi{
			Sid:            sid,
			GuestDesc:      guestDesc,
			EsxiAccessInfo: disksAccessInfo,
		})
	return nil, nil
}

func guestCreateFromCloudpods(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
	err := guestman.GetGuestManager().PrepareCreate(sid)
	if err != nil {
		return nil, err
	}

	var guestDesc = new(desc.SGuestDesc)
	err = body.Unmarshal(guestDesc, "desc")
	if err != nil {
		return nil, httperrors.NewBadRequestError("Guest desc unmarshal failed %s", err)
	}
	var disksAccessInfo = guestman.SCloudpodsAccessInfo{}
	err = body.Unmarshal(&disksAccessInfo, "cloudpods_access_info")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("cloudpods_access_info")
	}
	hostutils.DelayTask(ctx, guestman.GetGuestManager().GuestCreateFromCloudpods,
		&guestman.SGuestCreateFromCloudpods{
			Sid:                 sid,
			GuestDesc:           guestDesc,
			CloudpodsAccessInfo: disksAccessInfo,
		})
	return nil, nil
}
