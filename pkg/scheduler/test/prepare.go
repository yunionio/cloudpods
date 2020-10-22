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

package test

import (
	"fmt"

	"github.com/golang/mock/gomock"

	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/apis/compute"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/models"
	_ "yunion.io/x/onecloud/pkg/scheduler/algorithmprovider"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
	"yunion.io/x/onecloud/pkg/scheduler/data_manager/sku"
	"yunion.io/x/onecloud/pkg/scheduler/factory"
	"yunion.io/x/onecloud/pkg/scheduler/test/mock"
)

type sPredicateName string

var (
	HostStatus    sPredicateName = "a-GuestHostStatusFilter"
	Hypervisor    sPredicateName = "b-GuestHypervisorFilter"
	Migrate       sPredicateName = "d-GuestMigrateFilter"
	Domain        sPredicateName = "e-GuestDomainFilter"
	Image         sPredicateName = "e-GuestImageFilter"
	CPU           sPredicateName = "g-GuestCPUFilter"
	Memory        sPredicateName = "h-GuestMemoryFilter"
	Storage       sPredicateName = "i-GuestStorageFilter"
	Network       sPredicateName = "j-GuestNetworkFilter"
	IsolateDevice sPredicateName = "k-GuestIsolatedDeviceFilter"
	ResourceType  sPredicateName = "l-GuestResourceTypeFilter"
	ServerSku     sPredicateName = "n-ServerSkuFilter"

	// scheudle tag predicate, need db operator for now
	HostSchedtag    sPredicateName = "c-GuestAggregateFilter"
	DiskSchedtag    sPredicateName = "m-GuestDiskschedtagFilter"
	NetworkSchedtag sPredicateName = "o-GuestNetschedtagFilter"

	// quota
	Quota sPredicateName = "z-QuotaFilter"

	basePredicateNames = []sPredicateName{
		HostStatus, Hypervisor, Migrate, Domain, Image, CPU, Memory, Storage, Network,
		IsolateDevice, ResourceType, ServerSku,
	}
)

var (
	GlobalDoamin      = "default"
	GlobalProject     = "default"
	GlobalZone        = buildZone("default", "Default")
	GlobalCloudregion = buildCloudregion("default", "Default", "")
	GlobalWire        = "default"
	GlobaleVPC        = "default"
)

func buildCandidate(ctrl *gomock.Controller, param sGetterParams) *mock.MockCandidater {
	cn := mock.NewMockCandidater(ctrl)
	getter := buildGetter(ctrl, param)
	cn.EXPECT().Getter().AnyTimes().Return(getter)
	cn.EXPECT().IndexKey().AnyTimes().Return(getter.Id())
	cn.EXPECT().GetResourceType().AnyTimes().Return(getter.ResourceType())
	return cn
}

type sGetterParams struct {
	HostId                 string
	HostName               string
	IsPublic               *bool
	Domain                 string
	PublicScope            string
	Zone                   *models.SZone
	CloudRegion            *models.SCloudregion
	CloudProvider          *models.SCloudprovider
	HostType               string
	Storages               []*api.CandidateStorage
	Networks               []*api.CandidateNetwork
	OvnCapable             *bool
	Status                 string
	HostStatus             string
	Enabled                *bool
	ResourceType           string
	TotalCPUCount          int64
	FreeCPUCount           int64
	TotalMemorySize        int64
	FreeMemorySize         int64
	FreeStorageSizeAnyType int64
	FreePort               int
	QuotaKeys              *models.SComputeResourceKeys
	FreeGroupCount         int
	Skus                   []string
}

func buildGetter(ctrl *gomock.Controller, param sGetterParams) *mock.MockCandidatePropertyGetter {
	cg := mock.NewMockCandidatePropertyGetter(ctrl)
	cg.EXPECT().Id().AnyTimes().Return(param.HostId)
	cg.EXPECT().Name().AnyTimes().Return(param.HostName)
	cg.EXPECT().Zone().AnyTimes().Return(param.Zone)
	if param.IsPublic == nil {
		cg.EXPECT().IsPublic().AnyTimes().Return(true)
	} else {
		cg.EXPECT().IsPublic().AnyTimes().Return(*param.IsPublic)
	}
	if param.Enabled == nil {
		cg.EXPECT().Enabled().AnyTimes().Return(true)
	} else {
		cg.EXPECT().Enabled().AnyTimes().Return(*param.Enabled)
	}
	cg.EXPECT().DomainId().AnyTimes().Return(param.Domain)
	cg.EXPECT().PublicScope().AnyTimes().Return(param.PublicScope)
	cg.EXPECT().Region().AnyTimes().Return(param.CloudRegion)
	cg.EXPECT().Cloudprovider().AnyTimes().Return(param.CloudProvider)
	cg.EXPECT().HostType().AnyTimes().Return(param.HostType)
	cg.EXPECT().Storages().AnyTimes().Return(param.Storages)
	cg.EXPECT().Networks().AnyTimes().Return(param.Networks)
	cg.EXPECT().Sku(gomock.Any()).AnyTimes().DoAndReturn(func(instanceType string) *sku.ServerSku {
		for _, t := range param.Skus {
			if t != instanceType {
				continue
			}
			return &sku.ServerSku{
				Id:     fmt.Sprintf("%s-%s", param.Zone.Id, instanceType),
				ZoneId: param.Zone.Id,
			}
		}
		return nil
	})
	if param.OvnCapable == nil {
		cg.EXPECT().OvnCapable().AnyTimes().Return(false)
	} else {
		cg.EXPECT().OvnCapable().AnyTimes().Return(*param.OvnCapable)
	}
	if len(param.Status) == 0 {
		cg.EXPECT().Status().AnyTimes().Return(computeapi.HOST_HEALTH_STATUS_RUNNING)
	} else {
		cg.EXPECT().Status().AnyTimes().Return(param.Status)
	}
	if len(param.HostStatus) == 0 {
		cg.EXPECT().HostStatus().AnyTimes().Return(computeapi.HOST_ONLINE)
	} else {
		cg.EXPECT().HostStatus().AnyTimes().Return(param.HostStatus)
	}
	if len(param.ResourceType) == 0 {
		cg.EXPECT().ResourceType().AnyTimes().Return(computeapi.HostResourceTypeDefault)
	} else {
		cg.EXPECT().ResourceType().AnyTimes().Return(param.ResourceType)
	}
	cg.EXPECT().TotalCPUCount(gomock.Any()).AnyTimes().Return(param.TotalCPUCount)
	cg.EXPECT().FreeCPUCount(gomock.Any()).AnyTimes().Return(param.FreeCPUCount)
	cg.EXPECT().TotalMemorySize(gomock.Any()).AnyTimes().Return(param.TotalMemorySize)
	cg.EXPECT().FreeMemorySize(gomock.Any()).AnyTimes().Return(param.FreeMemorySize)
	cg.EXPECT().GetFreeStorageSizeOfType(gomock.Any(), gomock.Any()).AnyTimes().Return(param.FreeStorageSizeAnyType)
	cg.EXPECT().GetFreePort(gomock.Any()).AnyTimes().Return(param.FreePort)
	if param.QuotaKeys != nil {
		cg.EXPECT().GetQuotaKeys(gomock.Any()).AnyTimes().Return(param.QuotaKeys)
	}
	cg.EXPECT().GetFreeGroupCount(gomock.Any()).AnyTimes().Return(param.FreeGroupCount, nil)
	return cg
}

func buildScheduler(ctrl *gomock.Controller, predicates ...sPredicateName) core.Scheduler {
	pres := sets.NewString()
	for _, pre := range predicates {
		pres.Insert(string(pre))
	}
	algorithmProvider, _ := factory.GetAlgorithmProvider(factory.DefaultProvider)
	mockScheduler := mock.NewMockScheduler(ctrl)
	mockScheduler.EXPECT().BeforePredicate().AnyTimes().Return(nil)
	mockScheduler.EXPECT().Predicates().AnyTimes().DoAndReturn(func() (map[string]core.FitPredicate, error) {
		return factory.GetPredicates(pres)
	})
	mockScheduler.EXPECT().PriorityConfigs().AnyTimes().DoAndReturn(func() ([]core.PriorityConfig, error) {
		return factory.GetPriorityConfigs(algorithmProvider.PriorityKeys)
	})
	return mockScheduler
}

func buildZone(id, name string) *models.SZone {
	zone := &models.SZone{}
	zone.Id = id
	zone.Name = name
	zone.Status = "enable"
	return zone
}

func buildCloudregion(id, name, provider string) *models.SCloudregion {
	if len(provider) == 0 {
		provider = computeapi.CLOUD_PROVIDER_ONECLOUD
	}
	return &models.SCloudregion{
		SEnabledStatusStandaloneResourceBase: db.SEnabledStatusStandaloneResourceBase{
			SStatusStandaloneResourceBase: db.SStatusStandaloneResourceBase{
				SStandaloneResourceBase: db.SStandaloneResourceBase{
					SStandaloneAnonResourceBase: db.SStandaloneAnonResourceBase{
						Id: id,
					},
					Name: name,
				},
				SStatusResourceBase: db.SStatusResourceBase{
					Status: "inservice",
				},
			},
			SEnabledResourceBase: db.SEnabledResourceBase{
				Enabled: "true",
			},
		},
		Provider: provider,
	}
}

func buildStorage(id, name string, capacity int64) *api.CandidateStorage {
	storage := &models.SStorage{}
	storage.Id, storage.Name = id, name
	storage.DomainId = GlobalDoamin
	storage.IsPublic = true
	storage.PublicScope = "system"
	storage.Status = computeapi.STORAGE_ONLINE
	storage.Enabled = "true"
	storage.ZoneId = GlobalZone.GetId()
	storage.Capacity = capacity
	storage.StorageType = computeapi.STORAGE_LOCAL
	storage.MediumType = computeapi.DISK_TYPE_ROTATE
	storage.Cmtbound = 1.0
	storage.IsSysDiskStore = "true"
	return &api.CandidateStorage{
		SStorage: storage,
	}
}

func buildNetwork(id, name string, cidr string) *api.CandidateNetwork {
	network := &models.SNetwork{}
	network.Id, network.Name = id, name
	network.Status = computeapi.NETWORK_STATUS_AVAILABLE
	network.DomainId = GlobalDoamin
	network.ProjectId = GlobalProject
	network.WireId = GlobalWire
	network.ServerType = computeapi.NETWORK_TYPE_GUEST
	prefix, err := netutils.NewIPV4Prefix(cidr)
	if err == nil {
		network.GuestIpMask = prefix.MaskLen
		ipRange := prefix.ToIPRange()
		network.GuestIpStart = ipRange.StartIp().String()
		network.GuestIpEnd = ipRange.EndIp().String()
	}
	return &api.CandidateNetwork{
		SNetwork: network,
		VpcId:    GlobaleVPC,
	}
}

func buildInstanceGroup(id string, granularity int, force bool) *models.SGroup {
	group := &models.SGroup{}
	group.Id = id
	group.Status = "ready"
	group.DomainId = GlobalDoamin
	group.ProjectId = GlobalProject
	group.Granularity = granularity
	group.ForceDispersion = tristate.NewFromBool(force)
	return nil
}

func preSchedule(info *api.SchedInfo, candidates []core.Candidater, isForcast bool) (*core.Unit, []core.Candidater, core.IResultHelper) {
	resultHelper := core.SResultHelperFunc(core.ResultHelp)
	if isForcast {
		resultHelper = core.SResultHelperFunc(core.ResultHelpForForcast)
		info.Suggestion = true
		info.IsSuggestion = true
		info.SuggestionAll = true
		info.ShowSuggestionDetails = true
		info.SuggestionLimit = 100
	}
	return core.NewScheduleUnit(info, nil), candidates, resultHelper
}

func deepCopy(info *api.SchedInfo) *api.SchedInfo {
	serverConfigs := *info.ServerConfigs
	// disks
	disks := make([]*compute.DiskConfig, len(info.Disks))
	for i := range disks {
		disk := *info.Disks[i]
		disks[i] = &disk
	}
	// network
	networks := make([]*compute.NetworkConfig, len(info.Networks))
	for i := range networks {
		network := *info.Networks[i]
		networks[i] = &network
	}
	// baremetal_disk_config
	bareConfigs := make([]*compute.BaremetalDiskConfig, len(info.BaremetalDiskConfigs))
	for i := range bareConfigs {
		bareConfig := *info.BaremetalDiskConfigs[i]
		bareConfigs[i] = &bareConfig
	}
	// instancegroup
	instancegroupIds := make([]string, len(info.InstanceGroupIds))
	for i := range instancegroupIds {
		instancegroupIds[i] = info.InstanceGroupIds[i]
	}
	serverConfigs.Disks = disks
	serverConfigs.Networks = networks
	serverConfigs.BaremetalDiskConfigs = bareConfigs

	serverConfig := info.ServerConfig
	serverConfig.ServerConfigs = &serverConfigs

	scheduInput := *info.ScheduleInput
	scheduInput.ServerConfig = serverConfig

	copyInfo := *info
	copyInfo.ScheduleInput = &scheduInput
	preferCandidates := make([]string, len(info.PreferCandidates))
	for i := range preferCandidates {
		preferCandidates[i] = info.PreferCandidates[i]
	}
	instanceGroupDetail := make(map[string]*models.SGroup, len(info.InstanceGroupsDetail))
	for k, v := range info.InstanceGroupsDetail {
		// v only read
		instanceGroupDetail[k] = v
	}
	copyInfo.PreferCandidates = preferCandidates
	copyInfo.InstanceGroupsDetail = instanceGroupDetail

	return &copyInfo
}
