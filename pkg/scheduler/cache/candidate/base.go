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

package candidate

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	computedb "yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	computemodels "yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
	"yunion.io/x/onecloud/pkg/scheduler/data_manager/sku"
	schedmodels "yunion.io/x/onecloud/pkg/scheduler/models"
)

type BaseHostDesc struct {
	*computemodels.SHost
	Region        *computemodels.SCloudregion              `json:"region"`
	Zone          *computemodels.SZone                     `json:"zone"`
	Cloudprovider *computemodels.SCloudprovider            `json:"cloudprovider"`
	Cloudaccount  *computemodels.SCloudaccount             `json:"cloudaccount"`
	Networks      []*api.CandidateNetwork                  `json:"networks"`
	NetInterfaces map[string][]computemodels.SNetInterface `json:"net_interfaces"`
	Storages      []*api.CandidateStorage                  `json:"storages"`

	IsolatedDevices []*core.IsolatedDeviceDesc `json:"isolated_devices"`

	Tenants       map[string]int64          `json:"tenants"`
	HostSchedtags []computemodels.SSchedtag `json:"schedtags"`

	InstanceGroups map[string]*api.CandidateGroup `json:"instance_groups"`
	IpmiInfo       types.SIPMIInfo                `json:"ipmi_info"`

	Nics []*types.SNic `json:"nics"`

	SharedDomains []string               `json:"shared_domains"`
	PendingUsage  map[string]interface{} `json:"pending_usage"`

	ClassMetadata map[string]string `json:"class_metadata"`
}

type baseHostGetter struct {
	h *BaseHostDesc
}

func newBaseHostGetter(h *BaseHostDesc) *baseHostGetter {
	return &baseHostGetter{h}
}

func (b baseHostGetter) Id() string {
	return b.h.GetId()
}

func (b baseHostGetter) Name() string {
	return b.h.GetName()
}

func (b baseHostGetter) Zone() *computemodels.SZone {
	return b.h.Zone
}

func (b baseHostGetter) Host() *computemodels.SHost {
	return b.h.SHost
}

func (b baseHostGetter) IsArmHost() bool {
	return b.h.IsArmHost()
}

func (b baseHostGetter) CPUArch() string {
	return b.h.CpuArchitecture
}

func (b baseHostGetter) Cloudprovider() *computemodels.SCloudprovider {
	return b.h.Cloudprovider
}

func (b baseHostGetter) IsPublic() bool {
	return b.h.IsPublic
	/*provider := b.Cloudprovider()
	if provider == nil {
		return false
	}
	account := provider.GetCloudaccount()
	if account == nil {
		return false
	}
	return account.ShareMode == computeapi.CLOUD_ACCOUNT_SHARE_MODE_SYSTEM*/
}

func (b baseHostGetter) KeywordPlural() string {
	return b.h.KeywordPlural()
}

func (b baseHostGetter) DomainId() string {
	return b.h.DomainId
}

func (b baseHostGetter) PublicScope() string {
	return b.h.PublicScope
}

func (b baseHostGetter) SharedDomains() []string {
	return b.h.SharedDomains
}

func (b baseHostGetter) Region() *computemodels.SCloudregion {
	return b.h.Region
}

func (b baseHostGetter) HostType() string {
	return b.h.HostType
}

func (b baseHostGetter) HostSchedtags() []computemodels.SSchedtag {
	return b.h.HostSchedtags
}

func (b baseHostGetter) Sku(instanceType string) *sku.ServerSku {
	zone := b.Zone()
	return sku.GetByZone(instanceType, zone.GetId())
}

func (b baseHostGetter) Storages() []*api.CandidateStorage {
	return b.h.Storages
}

func (b baseHostGetter) InstanceGroups() map[string]*api.CandidateGroup {
	return b.h.InstanceGroups
}

func (b baseHostGetter) GetAllClassMetadata() (map[string]string, error) {
	return b.h.ClassMetadata, nil
}

func (b baseHostGetter) GetFreeGroupCount(groupId string) (int, error) {
	// Must Be
	scg, ok := b.h.InstanceGroups[groupId]
	if !ok {
		return 0, errors.Wrap(core.ErrInstanceGroupNotFound, groupId)
	}
	free := scg.Granularity - scg.ReferCount
	if free < 1 {
		return 0, nil
	}
	pendingScg, ok := b.h.GetPendingUsage().InstanceGroupUsage[groupId]
	if ok {
		free -= pendingScg.ReferCount
	}
	return free, nil
}

func (b baseHostGetter) Networks() []*api.CandidateNetwork {
	return b.h.Networks
}

func (b baseHostGetter) OvnCapable() bool {
	return false
}

func (b baseHostGetter) ResourceType() string {
	return reviseResourceType(b.h.ResourceType)
}

func (b baseHostGetter) NetInterfaces() map[string][]computemodels.SNetInterface {
	return b.h.NetInterfaces
}

func (b baseHostGetter) Status() string {
	return b.h.Status
}

func (b baseHostGetter) HostStatus() string {
	return b.h.HostStatus
}

func (b baseHostGetter) Enabled() bool {
	return b.h.GetEnabled()
}

func (b baseHostGetter) ProjectGuests() map[string]int64 {
	return b.h.Tenants
}

func (b baseHostGetter) CreatingGuestCount() int {
	return 0
}

func (b baseHostGetter) RunningCPUCount() int64 {
	return 0
}

func (b baseHostGetter) TotalCPUCount(_ bool) int64 {
	return int64(b.h.CpuCount)
}

func (b baseHostGetter) RunningMemorySize() int64 {
	return 0
}

func (b baseHostGetter) TotalMemorySize(_ bool) int64 {
	return int64(b.h.MemSize)
}

func (b baseHostGetter) GetFreeStorageSizeOfType(storageType string, useRsvd bool) (int64, int64) {
	var size int64
	var actualSize int64
	for _, s := range b.Storages() {
		if s.StorageType == storageType {
			size += int64(float32(s.Capacity) * s.Cmtbound)
			actualSize += s.Capacity - s.ActualCapacityUsed
		}
	}
	return size, actualSize
}

func (b baseHostGetter) GetFreePort(netId string) int {
	return b.h.GetFreePort(netId)
}

func (b baseHostGetter) GetIpmiInfo() types.SIPMIInfo {
	return b.h.IpmiInfo
}

func (b baseHostGetter) GetNics() []*types.SNic {
	return b.h.Nics
}

func (b baseHostGetter) GetQuotaKeys(s *api.SchedInfo) computemodels.SComputeResourceKeys {
	return b.h.getQuotaKeys(s)
}

func (b baseHostGetter) GetPendingUsage() *schedmodels.SPendingUsage {
	return b.h.GetPendingUsage()
}

func (b baseHostGetter) UnusedIsolatedDevices() []*core.IsolatedDeviceDesc {
	return b.h.UnusedIsolatedDevices()
}

func (b baseHostGetter) UnusedIsolatedDevicesByType(devType string) []*core.IsolatedDeviceDesc {
	return b.h.UnusedIsolatedDevicesByType(devType)
}

func (b baseHostGetter) UnusedIsolatedDevicesByVendorModel(vendorModel string) []*core.IsolatedDeviceDesc {
	return b.h.UnusedIsolatedDevicesByVendorModel(vendorModel)
}

func (b baseHostGetter) UnusedIsolatedDevicesByModel(model string) []*core.IsolatedDeviceDesc {
	return b.h.UnusedIsolatedDevicesByModel(model)
}

func (b baseHostGetter) GetIsolatedDevice(devID string) *core.IsolatedDeviceDesc {
	return b.h.GetIsolatedDevice(devID)
}

func (b baseHostGetter) UnusedGpuDevices() []*core.IsolatedDeviceDesc {
	return b.h.UnusedGpuDevices()
}

func (b baseHostGetter) GetIsolatedDevices() []*core.IsolatedDeviceDesc {
	return b.h.GetIsolatedDevices()
}

func reviseResourceType(resType string) string {
	if resType == "" {
		return computeapi.HostResourceTypeDefault
	}
	return resType
}

func newBaseHostDesc(b *baseBuilder, host *computemodels.SHost) (*BaseHostDesc, error) {
	host.ResourceType = reviseResourceType(host.ResourceType)
	desc := &BaseHostDesc{
		SHost: host,
	}

	if err := desc.fillCloudProvider(host); err != nil {
		return nil, fmt.Errorf("Fill cloudprovider info error: %v", err)
	}

	if err := desc.fillNetworks(host); err != nil {
		return nil, fmt.Errorf("Fill networks error: %v", err)
	}
	// only onecloud host should fill onecloud vpc networks
	if host.HostType == computeapi.HOST_TYPE_HYPERVISOR {
		if err := desc.fillOnecloudVpcNetworks(); err != nil {
			return nil, fmt.Errorf("Fill onecloud vpc networks error: %v", err)
		}
	}
	if host.HostType == computeapi.HOST_TYPE_CLOUDPODS {
		if err := desc.fillCloudpodsVpcNetworks(); err != nil {
			return nil, fmt.Errorf("Fill cloudpods vpc networks error: %v", err)
		}
	}

	if err := desc.fillZone(host); err != nil {
		return nil, fmt.Errorf("Fill zone error: %v", err)
	}

	if err := desc.fillRegion(host); err != nil {
		return nil, fmt.Errorf("Fill region error: %v", err)
	}

	if err := desc.fillResidentTenants(host); err != nil {
		return nil, fmt.Errorf("Fill resident tenants error: %v", err)
	}

	if err := desc.fillStorages(host); err != nil {
		return nil, fmt.Errorf("Fill storage error: %v", err)
	}

	if err := desc.fillSchedtags(); err != nil {
		return nil, fmt.Errorf("Fill schedtag error: %v", err)
	}

	if err := desc.fillInstanceGroups(host); err != nil {
		return nil, fmt.Errorf("Fill instance group error: %v", err)
	}

	if err := desc.fillClassMetadata(host); err != nil {
		return nil, fmt.Errorf("Fill class metadata error: %v", err)
	}

	if err := desc.fillIpmiInfo(host); err != nil {
		return nil, fmt.Errorf("Fill ipmi info error: %v", err)
	}

	if err := desc.fillNics(host); err != nil {
		return nil, fmt.Errorf("Fill nics info error: %v", err)
	}

	if err := desc.fillIsolatedDevices(b, host); err != nil {
		return nil, fmt.Errorf("Fill isolated devices error: %v", err)
	}

	desc.fillSharedDomains()
	desc.PendingUsage = desc.GetPendingUsage().ToMap()

	return desc, nil
}

func (b BaseHostDesc) GetSchedDesc() *jsonutils.JSONDict {
	desc := jsonutils.Marshal(b.SHost).(*jsonutils.JSONDict)

	if b.Cloudprovider != nil {
		p := b.Cloudprovider
		cloudproviderDesc := jsonutils.NewDict()
		cloudproviderDesc.Add(jsonutils.NewString(p.ProjectId), "tenant_id")
		cloudproviderDesc.Add(jsonutils.NewString(p.Provider), "provider")
		desc.Add(cloudproviderDesc, "cloudprovider")
	}

	return desc
}

func (b *BaseHostDesc) GetPendingUsage() *schedmodels.SPendingUsage {
	usage, err := schedmodels.HostPendingUsageManager.GetPendingUsage(b.GetId())
	if err != nil {
		return schedmodels.NewPendingUsageBySchedInfo(b.GetId(), nil)
	}
	return usage
}

func (b *BaseHostDesc) GetFreePort(netId string) int {
	var selNet *api.CandidateNetwork = nil
	for _, n := range b.Networks {
		if n.GetId() == netId {
			selNet = n
			break
		}
	}
	if selNet == nil {
		return 0
	}
	// freeCount, _ := selNet.GetFreeAddressCount()
	return selNet.FreePort
}

func (b BaseHostDesc) GetResourceType() string {
	return b.ResourceType
}

func (h *BaseHostDesc) UnusedIsolatedDevices() []*core.IsolatedDeviceDesc {
	ret := make([]*core.IsolatedDeviceDesc, 0)
	for _, dev := range h.IsolatedDevices {
		if len(dev.GuestID) == 0 {
			ret = append(ret, dev)
		}
	}
	return ret
}

func (h *BaseHostDesc) UnusedIsolatedDevicesByType(devType string) []*core.IsolatedDeviceDesc {
	ret := make([]*core.IsolatedDeviceDesc, 0)
	for _, dev := range h.UnusedIsolatedDevices() {
		if dev.DevType == devType {
			ret = append(ret, dev)
		}
	}
	return ret
}

func (h *BaseHostDesc) UnusedIsolatedDevicesByVendorModel(vendorModel string) []*core.IsolatedDeviceDesc {
	ret := make([]*core.IsolatedDeviceDesc, 0)
	vm := core.NewVendorModelByStr(vendorModel)
	for _, dev := range h.UnusedIsolatedDevices() {
		if dev.GetVendorModel().IsMatch(vm) {
			ret = append(ret, dev)
		}
	}
	return ret
}

func (h *BaseHostDesc) UnusedIsolatedDevicesByModel(model string) []*core.IsolatedDeviceDesc {
	ret := make([]*core.IsolatedDeviceDesc, 0)
	for _, dev := range h.UnusedIsolatedDevices() {
		if strings.Contains(dev.Model, model) {
			ret = append(ret, dev)
		}
	}
	return ret
}

func (h *BaseHostDesc) GetIsolatedDevice(devID string) *core.IsolatedDeviceDesc {
	for _, dev := range h.IsolatedDevices {
		if dev.ID == devID {
			return dev
		}
	}
	return nil
}

func (h *BaseHostDesc) GetIsolatedDevices() []*core.IsolatedDeviceDesc {
	return h.IsolatedDevices
}

func (h *BaseHostDesc) UnusedGpuDevices() []*core.IsolatedDeviceDesc {
	ret := make([]*core.IsolatedDeviceDesc, 0)
	for _, dev := range h.UnusedIsolatedDevices() {
		if strings.HasPrefix(dev.DevType, "GPU") {
			ret = append(ret, dev)
		}
	}
	return ret
}

func (h *BaseHostDesc) fillIsolatedDevices(b *baseBuilder, host *computemodels.SHost) error {
	allDevs := b.getIsolatedDevices(host.Id)
	if len(allDevs) == 0 {
		return nil
	}

	devs := make([]*core.IsolatedDeviceDesc, len(allDevs))
	for index, devModel := range allDevs {
		dev := &core.IsolatedDeviceDesc{
			ID:             devModel.Id,
			GuestID:        devModel.GuestId,
			HostID:         devModel.HostId,
			DevType:        devModel.DevType,
			Model:          devModel.Model,
			Addr:           devModel.Addr,
			VendorDeviceID: devModel.VendorDeviceId,
		}
		devs[index] = dev
	}
	h.IsolatedDevices = devs

	return nil
}

func (b *BaseHostDesc) fillCloudProvider(host *computemodels.SHost) error {
	b.Cloudprovider = host.GetCloudprovider()
	if b.Cloudprovider != nil {
		var err error
		b.Cloudaccount, err = b.Cloudprovider.GetCloudaccount()
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *BaseHostDesc) fillRegion(host *computemodels.SHost) error {
	b.Region, _ = host.GetRegion()
	return nil
}

func (b *BaseHostDesc) fillZone(host *computemodels.SHost) error {
	zone, _ := host.GetZone()
	b.Zone = zone
	b.ZoneId = host.ZoneId
	return nil
}

func (b *BaseHostDesc) fillResidentTenants(host *computemodels.SHost) error {
	rets, err := HostResidentTenantCount(host.Id)
	if err != nil {
		return err
	}

	b.Tenants = rets

	return nil
}

func (b *BaseHostDesc) fillSchedtags() error {
	b.HostSchedtags = b.SHost.GetSchedtags()
	return nil
}

func (b *BaseHostDesc) fillSharedDomains() error {
	b.SharedDomains = b.SHost.GetSharedDomains()
	return nil
}

func (b *BaseHostDesc) fillNetworks(host *computemodels.SHost) error {
	hostId := host.Id
	hostwires := computemodels.HostwireManager.Query().SubQuery()
	sq := hostwires.Query(sqlchemy.DISTINCT("wire_id", hostwires.Field("wire_id"))).Equals("host_id", hostId)
	networks := computemodels.NetworkManager.Query().SubQuery()
	q := networks.Query().In("wire_id", sq)

	nets := make([]computemodels.SNetwork, 0)
	err := computedb.FetchModelObjects(computemodels.NetworkManager, q, &nets)
	if err != nil {
		return err
	}
	b.Networks = make([]*api.CandidateNetwork, len(nets))
	for idx, n := range nets {
		freePort, err := n.GetFreeAddressCount()
		if err != nil {
			return errors.Wrapf(err, "GetFreeAddressCount for network %s(%s)", n.GetName(), n.GetId())
		}
		b.Networks[idx] = &api.CandidateNetwork{
			SNetwork:  &nets[idx],
			Schedtags: n.GetSchedtags(),
			FreePort:  freePort,
		}
	}

	netifs := host.GetNetInterfaces()
	netifIndexs := make(map[string][]computemodels.SNetInterface, 0)
	for _, netif := range netifs {
		if !netif.IsUsableServernic() {
			continue
		}
		wire := netif.GetWire()
		if wire == nil {
			continue
		}
		if _, exist := netifIndexs[wire.Id]; !exist {
			netifIndexs[wire.Id] = make([]computemodels.SNetInterface, 0)
		}
		netifIndexs[wire.Id] = append(netifIndexs[wire.Id], netif)
	}
	b.NetInterfaces = netifIndexs

	return nil
}

func (b *BaseHostDesc) fillCloudpodsVpcNetworks() error {
	nets := computemodels.NetworkManager.Query()
	wires := computemodels.WireManager.Query().SubQuery()
	vpcs := computemodels.VpcManager.Query().SubQuery()
	regions := computemodels.CloudregionManager.Query().SubQuery()
	q := nets.AppendField(nets.QueryFields()...)
	q = q.AppendField(
		vpcs.Field("id", "vpc_id"),
		regions.Field("provider"),
	)
	q = q.Join(wires, sqlchemy.Equals(wires.Field("id"), nets.Field("wire_id")))
	q = q.Join(vpcs, sqlchemy.Equals(vpcs.Field("id"), wires.Field("vpc_id")))
	q = q.Join(regions, sqlchemy.Equals(regions.Field("id"), vpcs.Field("cloudregion_id")))
	q = q.Filter(sqlchemy.AND(
		sqlchemy.Equals(regions.Field("provider"), computeapi.CLOUD_PROVIDER_CLOUDPODS),
		sqlchemy.NOT(sqlchemy.Equals(vpcs.Field("external_id"), computeapi.DEFAULT_VPC_ID)),
	))

	type Row struct {
		computemodels.SNetwork
		VpcId    string
		Provider string
	}
	rows := []Row{}
	if err := q.All(&rows); err != nil {
		return errors.Wrap(err, "query cloudpods vpc networks")
	}
	for i := range rows {
		row := &rows[i]
		net := &row.SNetwork
		net.SetModelManager(computemodels.NetworkManager, net)
		freePort, err := net.GetFreeAddressCount()
		if err != nil {
			return errors.Wrapf(err, "GetFreeAddressCount for network %s(%s)", net.GetName(), net.GetId())
		}
		candidateNet := &api.CandidateNetwork{
			SNetwork: net,
			FreePort: freePort,
			VpcId:    row.VpcId,
			Provider: row.Provider,
		}
		b.Networks = append(b.Networks, candidateNet)
	}
	return nil
}

func (b *BaseHostDesc) fillOnecloudVpcNetworks() error {
	nets := computemodels.NetworkManager.Query()
	wires := computemodels.WireManager.Query().SubQuery()
	vpcs := computemodels.VpcManager.Query().SubQuery()
	regions := computemodels.CloudregionManager.Query().SubQuery()
	q := nets.AppendField(nets.QueryFields()...)
	q = q.AppendField(
		vpcs.Field("id", "vpc_id"),
		regions.Field("provider"),
	)
	q = q.Join(wires, sqlchemy.Equals(wires.Field("id"), nets.Field("wire_id")))
	q = q.Join(vpcs, sqlchemy.Equals(vpcs.Field("id"), wires.Field("vpc_id")))
	q = q.Join(regions, sqlchemy.Equals(regions.Field("id"), vpcs.Field("cloudregion_id")))
	q = q.Filter(sqlchemy.AND(
		sqlchemy.Equals(regions.Field("provider"), computeapi.CLOUD_PROVIDER_ONECLOUD),
		sqlchemy.NOT(sqlchemy.Equals(vpcs.Field("id"), computeapi.DEFAULT_VPC_ID)),
	))

	type Row struct {
		computemodels.SNetwork
		VpcId    string
		Provider string
	}
	rows := []Row{}
	if err := q.All(&rows); err != nil {
		return errors.Wrap(err, "query onecloud vpc networks")
	}
	for i := range rows {
		row := &rows[i]
		net := &row.SNetwork
		net.SetModelManager(computemodels.NetworkManager, net)
		freePort, err := net.GetFreeAddressCount()
		if err != nil {
			return errors.Wrapf(err, "GetFreeAddressCount for network %s(%s)", net.GetName(), net.GetId())
		}
		candidateNet := &api.CandidateNetwork{
			SNetwork: net,
			FreePort: freePort,
			VpcId:    row.VpcId,
			Provider: row.Provider,
		}
		b.Networks = append(b.Networks, candidateNet)
	}
	return nil
}

func (b *BaseHostDesc) fillStorages(host *computemodels.SHost) error {
	ss := make([]*api.CandidateStorage, 0)
	for _, s := range host.GetHoststorages() {
		storage := s.GetStorage()
		ss = append(ss, &api.CandidateStorage{
			SStorage:     storage,
			FreeCapacity: storage.GetFreeCapacity(),
			Schedtags:    storage.GetSchedtags(),
		})
	}
	b.Storages = ss
	return nil
}

func (b *BaseHostDesc) fillInstanceGroups(host *computemodels.SHost) error {
	candidateSet := make(map[string]*api.CandidateGroup)
	groups, groupSet, err := host.InstanceGroups()
	if err != nil {
		b.InstanceGroups = candidateSet
		return err
	}
	for i := range groups {
		id := groups[i].GetId()
		candidateSet[id] = &api.CandidateGroup{
			SGroup:     &groups[i],
			ReferCount: groupSet[id],
		}
	}
	b.InstanceGroups = candidateSet
	return nil
}

func (b *BaseHostDesc) fillClassMetadata(host *computemodels.SHost) error {
	cm, err := host.GetAllClassMetadata()
	if err != nil {
		return err
	}
	b.ClassMetadata = cm
	return nil
}

func (b *BaseHostDesc) fillIpmiInfo(host *computemodels.SHost) error {
	info, err := host.GetIpmiInfo()
	if err != nil {
		return err
	}
	b.IpmiInfo = info
	return nil
}

func (b *BaseHostDesc) fillNics(host *computemodels.SHost) error {
	b.Nics = host.GetNics()
	return nil
}

func (h *BaseHostDesc) GetEnableStatus() string {
	if h.GetEnabled() {
		return "enable"
	}
	return "disable"
}

func (h *BaseHostDesc) GetHostType() string {
	if h.HostType == api.HostTypeBaremetal && h.IsBaremetal {
		return api.HostTypeBaremetal
	}
	return h.HostType
}

func (h *BaseHostDesc) getQuotaKeys(s *api.SchedInfo) computemodels.SComputeResourceKeys {
	computeKeys := computemodels.SComputeResourceKeys{}
	computeKeys.DomainId = s.Domain
	computeKeys.ProjectId = s.Project
	if h.Cloudprovider != nil {
		computeKeys.Provider = h.Cloudaccount.Provider
		computeKeys.Brand = h.Cloudaccount.Brand
		computeKeys.CloudEnv = h.Cloudaccount.GetCloudEnv()
		computeKeys.AccountId = h.Cloudaccount.Id
		computeKeys.ManagerId = h.Cloudprovider.Id
	} else {
		computeKeys.Provider = computeapi.CLOUD_PROVIDER_ONECLOUD
		computeKeys.Brand = computeapi.ONECLOUD_BRAND_ONECLOUD
		computeKeys.CloudEnv = computeapi.CLOUD_ENV_ON_PREMISE
		computeKeys.AccountId = ""
		computeKeys.ManagerId = ""
	}
	computeKeys.RegionId = h.Region.Id
	computeKeys.ZoneId = h.Zone.Id
	computeKeys.Hypervisor = computeapi.HOSTTYPE_HYPERVISOR[h.HostType]
	return computeKeys
}

func HostsResidentTenantStats(hostIDs []string) (map[string]map[string]interface{}, error) {
	residentTenantStats, err := FetchHostsResidentTenants(hostIDs)
	if err != nil {
		return nil, err
	}
	stat3 := make([]utils.StatItem3, len(residentTenantStats))
	for i, item := range residentTenantStats {
		stat3[i] = item
	}
	return utils.ToStatDict3(stat3)
}

func HostResidentTenantCount(id string) (map[string]int64, error) {
	residentTenantDict, err := HostsResidentTenantStats([]string{id})
	if err != nil {
		return nil, err
	}
	tenantMap, ok := residentTenantDict[id]
	if !ok {
		log.V(10).Infof("Not found host ID: %s when fill resident tenants, may be no guests on it.", id)
		return nil, nil
	}
	rets := make(map[string]int64, len(tenantMap))
	for tenantID, countObj := range tenantMap {
		rets[tenantID] = countObj.(int64)
	}
	return rets, nil
}

type DescBuilder struct {
	actor BuildActor

	isolatedDevicesDict map[string][]interface{}
}

func NewDescBuilder(act BuildActor) *DescBuilder {
	return &DescBuilder{
		actor: act,
	}
}

func (d *DescBuilder) Build(ids []string) ([]interface{}, error) {
	return d.actor.Do(ids)
}
