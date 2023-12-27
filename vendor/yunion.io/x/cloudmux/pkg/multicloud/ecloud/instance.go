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

package ecloud

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/sets"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SNovaRequest struct {
	SApiRequest
}

func NewNovaRequest(ar *SApiRequest) *SNovaRequest {
	return &SNovaRequest{
		SApiRequest: *ar,
	}
}

func (nr *SNovaRequest) GetPort() string {
	if nr.RegionId == "guangzhou-2" {
		return ""
	}
	return nr.SApiRequest.GetPort()
}

type SInstance struct {
	multicloud.SInstanceBase
	EcloudTags
	multicloud.SBillingBase
	SZoneRegionBase
	SCreateTime

	nicComplete bool
	host        *SHost
	image       *SImage

	sysDisk   cloudprovider.ICloudDisk
	dataDisks []cloudprovider.ICloudDisk

	Id               string
	Name             string
	Vcpu             int
	Vmemory          int
	KeyName          string
	ImageRef         string
	ImageName        string
	ImageOsType      string
	FlavorRef        string
	SystemDiskSizeGB int `json:"vdisk"`
	SystemDiskId     string
	ServerType       string
	ServerVmType     string
	EcStatus         string
	BootVolumeType   string
	Deleted          int
	Visible          bool
	Region           string
	PortDetail       []SInstanceNic
}

func (i *SInstance) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (i *SInstance) GetExpiredAt() time.Time {
	return time.Time{}
}

func (i *SInstance) GetId() string {
	return i.Id
}

func (i *SInstance) GetName() string {
	return i.Name
}

func (i *SInstance) GetHostname() string {
	return i.Name
}

func (i *SInstance) GetGlobalId() string {
	return i.GetId()
}

func (i *SInstance) GetStatus() string {
	switch i.EcStatus {
	case "active":
		return api.VM_RUNNING
	case "suspended", "paused":
		return api.VM_SUSPEND
	case "build", "rebuild", "resize", "verify_resize", "revert_resize", "password":
		return api.VM_STARTING
	case "reboot", "hard_reboot":
		return api.VM_STOPPING
	case "stopped", "shutoff":
		return api.VM_READY
	case "migrating":
		return api.VM_MIGRATING
	case "backuping":
		return api.VM_BACKUP_CREATING
	default:
		return api.VM_UNKNOWN
	}
}

func (i *SInstance) Refresh() error {
	// TODO
	return nil
}

func (i *SInstance) IsEmulated() bool {
	return false
}

func (self *SInstance) GetBootOrder() string {
	return "dcn"
}

func (self *SInstance) GetVga() string {
	return "std"
}

func (self *SInstance) GetVdi() string {
	return "vnc"
}

func (i *SInstance) GetImage() (*SImage, error) {
	if i.image != nil {
		return i.image, nil
	}
	image, err := i.host.zone.region.GetImage(i.ImageRef)
	if err != nil {
		return nil, err
	}
	i.image = image
	return i.image, nil
}

func (i *SInstance) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(i.ImageOsType)
}

func (i *SInstance) GetFullOsName() string {
	image, err := i.GetImage()
	if err != nil {
		return ""
	}
	return image.GetFullOsName()
}

func (i *SInstance) GetBios() cloudprovider.TBiosType {
	image, err := i.GetImage()
	if err != nil {
		return cloudprovider.BIOS
	}
	return image.GetBios()
}

func (i *SInstance) GetOsArch() string {
	image, err := i.GetImage()
	if err != nil {
		return ""
	}
	return image.GetOsArch()
}

func (i *SInstance) GetOsDist() string {
	image, err := i.GetImage()
	if err != nil {
		return ""
	}
	return image.GetOsDist()
}

func (i *SInstance) GetOsVersion() string {
	image, err := i.GetImage()
	if err != nil {
		return ""
	}
	return image.GetOsVersion()
}

func (i *SInstance) GetOsLang() string {
	image, err := i.GetImage()
	if err != nil {
		return ""
	}
	return image.GetOsLang()
}

func (i *SInstance) GetMachine() string {
	return "pc"
}

func (i *SInstance) GetInstanceType() string {
	return i.FlavorRef
}

func (in *SInstance) GetProjectId() string {
	return ""
}

func (in *SInstance) GetIHost() cloudprovider.ICloudHost {
	return in.host
}

func (in *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	if in.sysDisk == nil {
		in.fetchSysDisk()
	}
	if in.dataDisks == nil {
		err := in.fetchDataDisks()
		if err != nil {
			return nil, err
		}
	}
	return append([]cloudprovider.ICloudDisk{in.sysDisk}, in.dataDisks...), nil
}

func (in *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	if !in.nicComplete {
		err := in.makeNicComplete()
		if err != nil {
			return nil, errors.Wrap(err, "unable to make nics complete")
		}
		in.nicComplete = true
	}
	inics := make([]cloudprovider.ICloudNic, len(in.PortDetail))
	for i := range in.PortDetail {
		in.PortDetail[i].instance = in
		inics[i] = &in.PortDetail[i]
	}
	return inics, nil
}

func (in *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	if !in.nicComplete {
		err := in.makeNicComplete()
		if err != nil {
			return nil, errors.Wrap(err, "unable to make nics complete")
		}
		in.nicComplete = true
	}
	var eipId string
	for i := range in.PortDetail {
		if len(in.PortDetail[i].IpId) > 0 {
			eipId = in.PortDetail[i].IpId
			break
		}
	}
	if len(eipId) == 0 {
		return nil, nil
	}
	return in.host.zone.region.GetEipById(eipId)
}

func (in *SInstance) GetSecurityGroupIds() ([]string, error) {
	if !in.nicComplete {
		err := in.makeNicComplete()
		if err != nil {
			return nil, errors.Wrap(err, "unable to make nics complete")
		}
		in.nicComplete = true
	}
	ret := sets.NewString()
	for i := range in.PortDetail {
		for _, group := range in.PortDetail[i].SecurityGroups {
			ret.Insert(group.Id)
		}
	}
	return ret.UnsortedList(), nil
}

func (in *SInstance) GetVcpuCount() int {
	return in.Vcpu
}

func (in *SInstance) GetVmemSizeMB() int {
	return in.Vmemory
}

func (in *SInstance) SetSecurityGroups(ids []string) error {
	return cloudprovider.ErrNotImplemented
}

func (in *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_ECLOUD
}

func (in *SInstance) StartVM(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) UpdateVM(ctx context.Context, input cloudprovider.SInstanceUpdateOptions) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) RebuildRoot(ctx context.Context, config *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SInstance) DeployVM(ctx context.Context, opts *cloudprovider.SInstanceDeployOptions) error {
	return cloudprovider.ErrNotImplemented
}

func (in *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	return errors.ErrNotImplemented
}

func (in *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	url, err := in.host.zone.region.GetInstanceVNCUrl(in.GetId())
	if err != nil {
		return nil, err
	}
	ret := &cloudprovider.ServerVncOutput{
		Url:        url,
		Protocol:   "ecloud",
		InstanceId: in.GetId(),
		Hypervisor: api.HYPERVISOR_ECLOUD,
	}
	return ret, nil
}

func (in *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (in *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) Renew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) GetError() error {
	return nil
}

func (in *SInstance) fetchSysDisk() {
	storage, _ := in.host.zone.getStorageByType(api.STORAGE_ECLOUD_SYSTEM)
	disk := SDisk{
		storage: storage,
		ManualAttr: SDiskManualAttr{
			IsVirtual:  true,
			TempalteId: in.ImageRef,
			ServerId:   in.Id,
		},
		SCreateTime:     in.SCreateTime,
		SZoneRegionBase: in.SZoneRegionBase,
		ServerId:        []string{in.Id},
		IsShare:         false,
		IsDelete:        false,
		SizeGB:          in.SystemDiskSizeGB,
		ID:              in.SystemDiskId,
		Name:            fmt.Sprintf("%s-root", in.Name),
		Status:          "in-use",
		Type:            api.STORAGE_ECLOUD_SYSTEM,
	}
	in.sysDisk = &disk
	return
}

func (in *SInstance) fetchDataDisks() error {
	request := NewNovaRequest(NewApiRequest(in.host.zone.region.ID, "/api/v2/volume/volume/mount/list",
		map[string]string{"serverId": in.Id}, nil))
	disks := make([]SDisk, 0, 5)
	err := in.host.zone.region.client.doList(context.Background(), request, &disks)
	if err != nil {
		return err
	}
	idisks := make([]cloudprovider.ICloudDisk, len(disks))
	for i := range idisks {
		storageType := disks[i].Type
		storage, err := in.host.zone.getStorageByType(storageType)
		if err != nil {
			return errors.Wrapf(err, "unable to fetch storage with stoageType %s", storageType)
		}
		disks[i].storage = storage
		idisks[i] = &disks[i]
	}
	in.dataDisks = idisks
	return nil
}

func (in *SInstance) makeNicComplete() error {
	routerIds := sets.NewString()
	nics := make(map[string]*SInstanceNic, len(in.PortDetail))
	for i := range in.PortDetail {
		nic := &in.PortDetail[i]
		routerIds.Insert(nic.RouterId)
		nics[nic.PortId] = nic
	}
	for _, routerId := range routerIds.UnsortedList() {
		request := NewConsoleRequest(in.host.zone.region.ID, fmt.Sprintf("/api/vpc/%s/nic", routerId),
			map[string]string{
				"resourceId": in.Id,
			}, nil,
		)
		completeNics := make([]SInstanceNic, 0, len(nics)/2)
		err := in.host.zone.region.client.doList(context.Background(), request, &completeNics)
		if err != nil {
			return errors.Wrapf(err, "unable to get nics with instance %s in vpc %s", in.Id, routerId)
		}
		for i := range completeNics {
			id := completeNics[i].Id
			nic, ok := nics[id]
			if !ok {
				continue
			}
			nic.SInstanceNicDetail = completeNics[i].SInstanceNicDetail
		}
	}
	return nil
}

func (r *SRegion) findHost(zoneRegion string) (*SHost, error) {
	zone, err := r.FindZone(zoneRegion)
	if err != nil {
		return nil, err
	}
	return &SHost{
		zone: zone,
	}, nil
}

func (r *SRegion) GetInstancesWithHost(zoneRegion string) ([]SInstance, error) {
	instances, err := r.GetInstances(zoneRegion)
	if err != nil {
		return nil, err
	}
	for i := range instances {
		host, _ := r.findHost(instances[i].Region)
		instances[i].host = host
	}
	return instances, nil
}

func (r *SRegion) GetInstances(zoneRegion string) ([]SInstance, error) {
	return r.getInstances(zoneRegion, "")
}

func (r *SRegion) getInstances(zoneRegion string, serverId string) ([]SInstance, error) {
	query := map[string]string{
		"serverTypes":  "VM",
		"productTypes": "NORMAL,AUTOSCALING,VO,CDN,PAAS_MASTER,PAAS_SLAVE,VCPE,EMR,LOGAUDIT",
		//"productTypes": "NORMAL",
		"visible": "true",
	}
	if len(serverId) > 0 {
		query["serverId"] = serverId
	}
	if len(zoneRegion) > 0 {
		query["region"] = zoneRegion
	}
	request := NewNovaRequest(NewApiRequest(r.ID, "/api/v2/server/web/with/network", query, nil))
	var instances []SInstance
	err := r.client.doList(context.Background(), request, &instances)
	if err != nil {
		return nil, err
	}
	return instances, nil
}

func (r *SRegion) GetInstanceById(id string) (*SInstance, error) {
	instances, err := r.getInstances("", id)
	if err != nil {
		return nil, err
	}
	if len(instances) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	instance := &instances[0]
	host, err := r.findHost(instance.Region)
	if err == nil {
		instance.host = host
	}
	return instance, nil
}

func (r *SRegion) GetInstanceVNCUrl(instanceId string) (string, error) {
	request := NewNovaRequest(NewApiRequest(r.ID, fmt.Sprintf("/api/server/%s/vnc", instanceId), nil, nil))
	var url string
	err := r.client.doGet(context.Background(), request, &url)
	if err != nil {
		return "", err
	}
	return url, nil
}
