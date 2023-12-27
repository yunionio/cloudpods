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

package jdcloud

import (
	"context"
	"fmt"
	"time"

	commodels "github.com/jdcloud-api/jdcloud-sdk-go/services/common/models"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/vm/apis"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/vm/client"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/vm/models"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/sets"

	napis "yunion.io/x/cloudmux/pkg/apis"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SInstance struct {
	multicloud.SInstanceBase
	multicloud.SBillingBase
	JdcloudTags

	host  *SHost
	image *SImage

	models.Instance

	instanceType *SInstanceType
}

func (i *SInstance) GetBillingType() string {
	return billingType(&i.Charge)
}

func (i *SInstance) GetExpiredAt() time.Time {
	return expireAt(&i.Charge)
}

func (i *SInstance) GetId() string {
	return i.InstanceId
}

func (i *SInstance) GetName() string {
	return i.InstanceName
}

func (i *SInstance) GetHostname() string {
	return i.Hostname
}

func (i *SInstance) GetGlobalId() string {
	return i.GetId()
}

func (i *SInstance) GetStatus() string {
	switch i.Status {
	case "pending":
		return api.VM_DEPLOYING
	case "starting":
		return api.VM_STARTING
	case "running":
		return api.VM_RUNNING
	case "stopping":
		return api.VM_STOPPING
	case "stopped":
		return api.VM_READY
	case "reboot":
		return api.VM_STARTING
	case "rebuilding":
		return api.VM_REBUILD_ROOT
	case "resizing":
		return api.VM_CHANGE_FLAVOR
	case "deleting":
		return api.VM_DELETING
	default:
		return api.VM_UNKNOWN
	}
}

func (i *SInstance) Refresh() error {
	return nil
}

func (i *SInstance) IsEmulated() bool {
	return false
}

func (i *SInstance) GetBootOrder() string {
	return "dcn"
}

func (i *SInstance) GetVga() string {
	return "std"
}

func (i *SInstance) GetVdi() string {
	return "vnc"
}

func (i *SInstance) GetImage() (*SImage, error) {
	if i.image != nil {
		return i.image, nil
	}
	image, err := i.host.zone.region.GetImage(i.ImageId)
	if err != nil {
		return nil, err
	}
	i.image = image
	return i.image, nil
}

func (i *SInstance) GetOsType() cloudprovider.TOsType {
	image, err := i.GetImage()
	if err != nil {
		return cloudprovider.OsTypeLinux
	}
	return image.GetOsType()
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
		return napis.OS_ARCH_X86_64
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
	return i.InstanceType
}

func (in *SInstance) GetProjectId() string {
	return ""
}

func (in *SInstance) GetIHost() cloudprovider.ICloudHost {
	return in.host
}

func (in *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks := make([]cloudprovider.ICloudDisk, 0, len(in.DataDisks)+1)
	if in.SystemDisk.DiskCategory != "local" {
		disk := &SDisk{
			Disk:         in.SystemDisk.CloudDisk,
			ImageId:      in.ImageId,
			IsSystemDisk: true,
		}
		stroage, err := in.host.zone.getStorageByType(disk.DiskType)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to find storage with type %s", disk.DiskType)
		}
		disk.storage = stroage
		disks = append(disks, disk)
	}
	for i := range in.DataDisks {
		disk := &SDisk{
			Disk: in.DataDisks[i].CloudDisk,
		}
		stroage, err := in.host.zone.getStorageByType(disk.DiskType)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to find storage with type %s", disk.DiskType)
		}
		disk.storage = stroage
		disks = append(disks, disk)
	}
	return disks, nil
}

func (in *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	nis := make([]cloudprovider.ICloudNic, 0, len(in.SecondaryNetworkInterfaces)+1)
	nis = append(nis, &SInstanceNic{
		instance:                 in,
		InstanceNetworkInterface: in.PrimaryNetworkInterface.NetworkInterface,
	})
	for i := range in.SecondaryNetworkInterfaces {
		nis = append(nis, &SInstanceNic{
			instance:                 in,
			InstanceNetworkInterface: in.SecondaryNetworkInterfaces[i].NetworkInterface,
		})
	}
	return nis, nil
}

func (in *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	eip, err := in.host.zone.region.GetEIPById(in.ElasticIpId)
	if err != nil {
		return nil, err
	}
	return eip, nil
}

func (in *SInstance) GetSecurityGroupIds() ([]string, error) {
	ids := sets.NewString()
	for _, sg := range in.PrimaryNetworkInterface.NetworkInterface.SecurityGroups {
		ids.Insert(sg.GroupId)
	}
	for i := range in.SecondaryNetworkInterfaces {
		for _, sg := range in.SecondaryNetworkInterfaces[i].NetworkInterface.SecurityGroups {
			ids.Insert(sg.GroupId)
		}
	}
	return ids.UnsortedList(), nil
}

func (in *SInstance) fetchInstanceType() error {
	its, err := in.host.zone.region.InstanceTypes(in.InstanceType)
	if err != nil {
		return err
	}
	if len(its) == 0 {
		return cloudprovider.ErrNotFound
	}
	for i := range its {
		if its[i].InstanceType.InstanceType == in.InstanceType {
			in.instanceType = &its[i]
			return nil
		}
	}
	return cloudprovider.ErrNotFound
}

func (in *SInstance) GetVcpuCount() int {
	if in.instanceType == nil {
		err := in.fetchInstanceType()
		if err != nil {
			log.Errorf("unable to get instance type %s: %v", in.InstanceType, err)
			return 0
		}
	}
	return in.instanceType.GetCpu()
}

func (in *SInstance) GetVmemSizeMB() int {
	if in.instanceType == nil {
		err := in.fetchInstanceType()
		if err != nil {
			log.Errorf("unable to get instance type %s", in.InstanceType)
			return 0
		}
	}
	return in.instanceType.GetMemoryMB()
}

func (in *SInstance) SetSecurityGroups(ids []string) error {
	return cloudprovider.ErrNotImplemented
}

func (in *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_JDCLOUD
}

func (in *SInstance) StartVM(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (in *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	return cloudprovider.ErrNotImplemented
}

func (in *SInstance) DeleteVM(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (in *SInstance) UpdateVM(ctx context.Context, input cloudprovider.SInstanceUpdateOptions) error {
	return cloudprovider.ErrNotSupported
}

func (in *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) RebuildRoot(ctx context.Context, config *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SInstance) DeployVM(ctx context.Context, opts *cloudprovider.SInstanceDeployOptions) error {
	return cloudprovider.ErrNotImplemented
}

func (in *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (in *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	region := in.host.zone.region
	req := apis.NewDescribeInstanceVncUrlRequest(region.ID, in.InstanceId)
	client := client.NewVmClient(region.getCredential())
	client.Logger = Logger{debug: region.client.debug}
	resp, err := client.DescribeInstanceVncUrl(req)
	if err != nil {
		return nil, err
	}

	ret := &cloudprovider.ServerVncOutput{
		Url:        resp.Result.VncUrl,
		Protocol:   "jdcloud",
		InstanceId: in.GetId(),
		Hypervisor: api.HYPERVISOR_JDCLOUD,
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

func (r *SRegion) GetInstances(zoneId string, ids []string, pangeNumber, pageSize int) ([]SInstance, int, error) {
	filters := []commodels.Filter{}
	if zoneId != "" {
		filters = append(filters, commodels.Filter{
			Name:   "az",
			Values: []string{zoneId},
		})
	}
	if len(ids) > 0 {
		filters = append(filters, commodels.Filter{
			Name:   "instanceId",
			Values: ids,
		})
	}
	req := apis.NewDescribeInstancesRequestWithAllParams(r.ID, &pangeNumber, &pageSize, filters)
	client := client.NewVmClient(r.getCredential())
	client.Logger = Logger{debug: r.client.debug}
	resp, err := client.DescribeInstances(req)
	if err != nil {
		return nil, 0, err
	}
	if resp.Error.Code >= 400 {
		return nil, 0, fmt.Errorf(resp.Error.Message)
	}
	ins := make([]SInstance, len(resp.Result.Instances))
	for i := range ins {
		ins[i] = SInstance{
			Instance: resp.Result.Instances[i],
		}
	}
	return ins, resp.Result.TotalCount, nil
}

func (r *SRegion) GetInstanceById(id string) (*SInstance, error) {
	req := apis.NewDescribeInstanceRequest(r.ID, id)
	client := client.NewVmClient(r.getCredential())
	client.Logger = Logger{debug: r.client.debug}
	resp, err := client.DescribeInstance(req)
	if err != nil {
		return nil, err
	}
	return &SInstance{
		Instance: resp.Result.Instance,
	}, nil
}

func (r *SRegion) FillHost(instance *SInstance) {
	izone, err := r.getIZoneByRealId(instance.Az)
	if err != nil {
		log.Errorf("unable to find zone %s: %v", instance.Az, err)
		return
	}
	zone := izone.(*SZone)
	instance.host = &SHost{
		zone: zone,
	}
}
