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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/scheduler"
)

func (self *SGuest) PerformConvert(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data *api.ConvertToKvmInput,
) (jsonutils.JSONObject, error) {
	if data.TargetHypervisor != api.HYPERVISOR_KVM {
		return nil, httperrors.NewBadRequestError("not support target hypervisor %s", data.TargetHypervisor)
	}

	return self.PerformConvertToKvm(ctx, userCred, query, data)
}

func (self *SGuest) PerformConvertToKvm(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data *api.ConvertToKvmInput,
) (jsonutils.JSONObject, error) {
	if len(self.GetMetadata(ctx, api.SERVER_META_CONVERTED_SERVER, userCred)) > 0 {
		return nil, httperrors.NewBadRequestError("guest has been converted")
	}
	drv, err := self.GetDriver()
	if err != nil {
		return nil, err
	}

	if drv.GetProvider() == api.CLOUD_PROVIDER_ONECLOUD && drv.GetHypervisor() == api.HYPERVISOR_ESXI {
		return self.ConvertEsxiToKvm(ctx, userCred, data)
	}
	if drv.GetProvider() == api.CLOUD_PROVIDER_CLOUDPODS && drv.GetHypervisor() == api.HYPERVISOR_DEFAULT {
		return self.ConvertCloudpodsToKvm(ctx, userCred, data)
	}
	return nil, httperrors.NewBadRequestError("not support %s", self.Hypervisor)
}

func (self *SGuest) ConvertCloudpodsToKvm(ctx context.Context, userCred mcclient.TokenCredential, data *api.ConvertToKvmInput) (jsonutils.JSONObject, error) {
	preferHost := data.PreferHost
	if len(preferHost) > 0 {
		iHost, err := HostManager.FetchByIdOrName(ctx, userCred, preferHost)
		if err != nil {
			return nil, err
		}
		host := iHost.(*SHost)
		if host.HostType != api.HOST_TYPE_HYPERVISOR {
			return nil, httperrors.NewBadRequestError("host %s is not kvm host", preferHost)
		}
		preferHost = host.GetId()
	}

	if self.Status != api.VM_READY {
		return nil, httperrors.NewBadRequestError("guest status must be ready")
	}
	newGuest, createInput, err := self.createConvertedServer(ctx, userCred, data)
	if err != nil {
		return nil, errors.Wrap(err, "create converted server")
	}

	return nil, self.StartConvertToKvmTask(ctx, userCred, "GuestConvertCloudpodsToKvmTask", preferHost, newGuest, createInput, data)
}

func (self *SGuest) ConvertEsxiToKvm(ctx context.Context, userCred mcclient.TokenCredential, data *api.ConvertToKvmInput) (jsonutils.JSONObject, error) {
	preferHost := data.PreferHost
	if len(preferHost) > 0 {
		iHost, err := HostManager.FetchByIdOrName(ctx, userCred, preferHost)
		if err != nil {
			return nil, err
		}
		host := iHost.(*SHost)
		if host.HostType != api.HOST_TYPE_HYPERVISOR {
			return nil, httperrors.NewBadRequestError("host %s is not kvm host", preferHost)
		}
		preferHost = host.GetId()
	}

	if self.Status != api.VM_READY {
		return nil, httperrors.NewBadRequestError("guest status must be ready")
	}

	nets, err := self.GetNetworks("")
	if err != nil {
		return nil, errors.Wrap(err, "GetNetworks")
	}
	if len(nets) == 0 {
		syncIps := self.GetMetadata(ctx, "sync_ips", userCred)
		if len(syncIps) > 0 {
			return nil, errors.Wrap(httperrors.ErrInvalidStatus, "VMware network not configured properly")
		}
	}

	newGuest, createInput, err := self.createConvertedServer(ctx, userCred, data)
	if err != nil {
		return nil, errors.Wrap(err, "create converted server")
	}

	if data.Networks != nil && len(data.Networks) != len(createInput.Networks) {
		return nil, httperrors.NewInputParameterError("input network configs length must equal guestnetworks length")
	}

	for i := 0; i < len(createInput.Networks); i++ {
		createInput.Networks[i].Network = ""
		createInput.Networks[i].Wire = ""
		if data.Networks != nil {
			createInput.Networks[i].Network = data.Networks[i].Network
			createInput.Networks[i].Address = data.Networks[i].Address
			if data.Networks[i].Schedtags != nil {
				createInput.Networks[i].Schedtags = data.Networks[i].Schedtags
			}
		}
	}

	return nil, self.StartConvertToKvmTask(ctx, userCred, "GuestConvertEsxiToKvmTask", preferHost, newGuest, createInput, data)
}

func (self *SGuest) StartConvertToKvmTask(
	ctx context.Context, userCred mcclient.TokenCredential, taskName, preferHostId string,
	newGuest *SGuest, createInput *api.ServerCreateInput, data *api.ConvertToKvmInput,
) error {
	params := jsonutils.NewDict()
	if len(preferHostId) > 0 {
		params.Set("prefer_host_id", jsonutils.NewString(preferHostId))
	}
	params.Set("target_guest_id", jsonutils.NewString(newGuest.Id))
	params.Set("input", jsonutils.Marshal(createInput))
	params.Set("deploy_telegraf", jsonutils.NewBool(data.DeployTelegraf))
	task, err := taskman.TaskManager.NewTask(ctx, taskName, self, userCred,
		params, "", "", nil)
	if err != nil {
		return err
	} else {
		self.SetStatus(ctx, userCred, api.VM_CONVERTING, "esxi guest convert to kvm")
		task.ScheduleRun(nil)
		return nil
	}
}

func (self *SGuest) createConvertedServer(ctx context.Context, userCred mcclient.TokenCredential, data *api.ConvertToKvmInput) (*SGuest, *api.ServerCreateInput, error) {
	// set guest pending usage
	pendingUsage, pendingRegionUsage, err := self.getGuestUsage(1)
	keys, err := self.GetQuotaKeys()
	if err != nil {
		return nil, nil, errors.Wrap(err, "GetQuotaKeys")
	}
	pendingUsage.SetKeys(keys)
	err = quotas.CheckSetPendingQuota(ctx, userCred, &pendingUsage)
	if err != nil {
		return nil, nil, httperrors.NewOutOfQuotaError("Check set pending quota error %s", err)
	}
	regionKeys, err := self.GetRegionalQuotaKeys()
	if err != nil {
		quotas.CancelPendingUsage(ctx, userCred, &pendingUsage, &pendingUsage, false)
		return nil, nil, errors.Wrap(err, "GetRegionalQuotaKeys")
	}
	pendingRegionUsage.SetKeys(regionKeys)
	err = quotas.CheckSetPendingQuota(ctx, userCred, &pendingRegionUsage)
	if err != nil {
		quotas.CancelPendingUsage(ctx, userCred, &pendingUsage, &pendingUsage, false)
		return nil, nil, errors.Wrap(err, "CheckSetPendingQuota")
	}
	// generate guest create params
	createInput := self.ToCreateInput(ctx, userCred)
	createInput.Hypervisor = api.HYPERVISOR_KVM
	createInput.PreferHost = data.PreferHost
	createInput.GenerateName = fmt.Sprintf("%s-%s", self.Name, api.HYPERVISOR_KVM)
	createInput.Hostname = self.Name
	if self.Hostname != "" {
		createInput.Hostname = self.Hostname
	}

	if self.Hypervisor == api.HYPERVISOR_ESXI {
		// change drivers so as to bootable in KVM
		for i := range createInput.Disks {
			if !utils.IsInStringArray(createInput.Disks[i].Driver, []string{api.DISK_DRIVER_VIRTIO, api.DISK_DRIVER_PVSCSI, api.DISK_DRIVER_IDE}) {
				createInput.Disks[i].Driver = api.DISK_DRIVER_IDE
			}
			createInput.Disks[i].Format = ""
			createInput.Disks[i].Backend = ""
			createInput.Disks[i].Medium = ""
		}
		gns, err := self.GetNetworks("")
		if err != nil {
			return nil, nil, errors.Wrap(err, "GetNetworks")
		}
		for i := range createInput.Networks {
			if createInput.Networks[i].Driver != "e1000" && createInput.Networks[i].Driver != "vmxnet3" {
				createInput.Networks[i].Driver = "e1000"
			}
			createInput.Networks[i].Network = ""
			createInput.Networks[i].Wire = ""
			createInput.Networks[i].Mac = gns[i].MacAddr
			createInput.Networks[i].Address = gns[i].IpAddr
		}
		createInput.Vdi = api.VM_VDI_PROTOCOL_VNC
	} else {
		createInput.Disks[0].ImageId = ""
	}

	if data.Networks != nil && len(data.Networks) != len(createInput.Networks) {
		return nil, nil, httperrors.NewInputParameterError("input network configs length  must equal guestnetworks length")
	}

	for i := 0; i < len(createInput.Networks); i++ {
		createInput.Networks[i].Network = ""
		createInput.Networks[i].Wire = ""
		if data.Networks != nil {
			if data.Networks[i].Schedtags != nil {
				createInput.Networks[i].Schedtags = data.Networks[i].Schedtags
			}
			createInput.Networks[i].Address = data.Networks[i].Address
			createInput.Networks[i].Network = data.Networks[i].Network
		}
	}

	schedDesc := self.ToSchedDesc()
	schedDesc.PreferHost = data.PreferHost
	for i := range schedDesc.Disks {
		schedDesc.Disks[i].Backend = ""
		schedDesc.Disks[i].Medium = ""
		schedDesc.Disks[i].Storage = ""
	}
	schedDesc.Networks = data.Networks
	schedDesc.Hypervisor = api.HYPERVISOR_KVM

	s := auth.GetAdminSession(ctx, options.Options.Region)
	succ, res, err := scheduler.SchedManager.DoScheduleForecast(s, schedDesc, 1)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Do schedule migrate forecast")
	}
	if !succ {
		return nil, nil, httperrors.NewInsufficientResourceError(res.String())
	}

	lockman.LockClass(ctx, GuestManager, userCred.GetProjectId())
	defer lockman.ReleaseClass(ctx, GuestManager, userCred.GetProjectId())
	newGuest, err := db.DoCreate(GuestManager, ctx, userCred, nil,
		jsonutils.Marshal(createInput), self.GetOwnerId())
	quotas.CancelPendingUsage(ctx, userCred, &pendingUsage, &pendingUsage, true)
	if err != nil {
		return nil, nil, errors.Wrap(err, "db.DoCreate")
	}
	return newGuest.(*SGuest), createInput, nil
}

func (manager *SGuestManager) PerformBatchConvertPrecheck(
	ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *api.BatchConvertToKvmCheckInput,
) (jsonutils.JSONObject, error) {
	if len(data.GuestIds) == 0 {
		return nil, httperrors.NewInputParameterError("missing guest ids")
	}
	guests := make([]SGuest, 0)
	q := GuestManager.Query().In("id", data.GuestIds)
	err := db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		return nil, httperrors.NewInternalServerError("%v", err)
	}
	if len(guests) != len(data.GuestIds) {
		return nil, httperrors.NewBadRequestError("Check input guests is exist")
	}
	res := jsonutils.NewDict()
	for i := 0; i < len(guests); i++ {
		gns, err := guests[i].GetNetworks("")
		if err != nil {
			return nil, errors.Wrapf(err, "Get guest networks %s", err)
		}
		for j := 0; j < len(gns); j++ {
			if gns[j].IpAddr != "" {
				cnt, err := NetworkManager.checkIpHasOneCloudNetworks(gns[j].IpAddr)
				if err != nil {
					return nil, err
				}
				if cnt <= 0 {
					reason := fmt.Sprintf("kvm networks has no addr %s for guest %s convert", gns[j].IpAddr, guests[i].GetName())
					res.Set("reason", jsonutils.NewString(reason))
					res.Set("network_failed", jsonutils.JSONTrue)
					return res, nil
				}
			}
		}
	}

	return res, nil
}

func (manager *SNetworkManager) checkIpHasOneCloudNetworks(ipAddr string) (int, error) {
	ip4Addr, err := netutils.NewIPV4Addr(ipAddr)
	if err != nil {
		return -1, err
	}
	q := manager.Query()
	// filter onecloud wire
	wireQ := WireManager.Query().IsNullOrEmpty("manager_id").SubQuery()
	ipStart := sqlchemy.INET_ATON(q.Field("guest_ip_start"))
	ipEnd := sqlchemy.INET_ATON(q.Field("guest_ip_end"))
	ipCondtion := sqlchemy.AND(
		sqlchemy.GE(ipEnd, uint32(ip4Addr)),
		sqlchemy.LE(ipStart, uint32(ip4Addr)),
	)
	q = q.Filter(ipCondtion)
	q = q.Join(wireQ, sqlchemy.Equals(q.Field("wire_id"), wireQ.Field("id")))
	return q.CountWithError()
}
