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

package tasks

import (
	"context"
	"path/filepath"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/redfish"
)

type SBaremetalCdromTask struct {
	SBaremetalTaskBase
}

func NewBaremetalCdromTask(
	userCred mcclient.TokenCredential,
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) ITask {
	task := &SBaremetalCdromTask{
		SBaremetalTaskBase: newBaremetalTaskBase(userCred, baremetal, taskId, data),
	}
	task.SetVirtualObject(task)

	log.Debugf("NewBaremetalCdromTask: %s", data)

	action, _ := data.GetString("action")
	if action == api.BAREMETAL_CDROM_ACTION_INSERT {
		task.SetStage(task.DoInsertISO)
	} else {
		task.SetStage(task.DoEjectISO)
	}
	return task
}

func (self *SBaremetalCdromTask) GetName() string {
	return "SBaremetalCdromTask"
}

func (self *SBaremetalCdromTask) getRedfishApi(ctx context.Context) (redfish.IRedfishDriver, error) {
	ipmiInfo := self.Baremetal.GetRawIPMIConfig()
	if ipmiInfo == nil {
		ipmiInfo = &types.SIPMIInfo{}
	}
	if ipmiInfo.IpAddr == "" {
		return nil, errors.Error("empty IPMI ip_addr")
	}
	if ipmiInfo.Username == "" {
		return nil, errors.Error("empty IPMI username")
	}
	if ipmiInfo.Password == "" {
		return nil, errors.Error("empty IPMI password")
	}
	if !ipmiInfo.CdromBoot {
		return nil, errors.Error("mount Virtual Cdrom not supported")
	}
	redfishCli := redfish.NewRedfishDriver(ctx, "https://"+ipmiInfo.IpAddr, ipmiInfo.Username, ipmiInfo.Password, false)
	if redfishCli != nil {
		return redfishCli, nil
	} else {
		return nil, errors.Error("invalid redfish Api client")
	}
}

func (self *SBaremetalCdromTask) DoInsertISO(ctx context.Context, args interface{}) error {
	redfishCli, err := self.getRedfishApi(ctx)
	if err != nil {
		return errors.Wrap(err, "getRedfishApi")
	}
	imageId, _ := self.GetData().GetString("image_id")
	boot := jsonutils.QueryBoolean(self.GetData(), "boot", false)
	localImagePath := filepath.Join(o.Options.CachePath, imageId)
	if !fileutils2.Exists(localImagePath) {
		return errors.Error("image not cached")
	}
	imageBaseUrl := self.Baremetal.GetImageCacheUrl()
	if len(imageBaseUrl) == 0 {
		return errors.Error("empty image base url")
	}
	cdromPath := httputils.JoinPath(imageBaseUrl, "/images/"+imageId)
	err = redfish.MountVirtualCdrom(ctx, redfishCli, cdromPath, boot)
	if err != nil {
		return errors.Wrap(err, "MountVirtualCdrom")
	}
	self.Baremetal.AutoSyncStatus()
	SetTaskComplete(self, nil)
	return nil
}

func (self *SBaremetalCdromTask) DoEjectISO(ctx context.Context, args interface{}) error {
	redfishCli, err := self.getRedfishApi(ctx)
	if err != nil {
		return errors.Wrap(err, "getRedfishApi")
	}
	err = redfish.UmountVirtualCdrom(ctx, redfishCli)
	if err != nil {
		return errors.Wrap(err, "UmountVirtualCdrom")
	}
	self.Baremetal.AutoSyncStatus()
	SetTaskComplete(self, nil)
	return nil
}
