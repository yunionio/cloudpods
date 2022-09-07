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
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"yunion.io/x/onecloud/pkg/apis/compute"
	apisdu "yunion.io/x/onecloud/pkg/apis/scheduler"
	_ "yunion.io/x/onecloud/pkg/compute/guestdrivers"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

func TestGenericSchedulerSchedule(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	commonInfo := &api.SchedInfo{
		ScheduleInput: &apisdu.ScheduleInput{
			ServerConfig: apisdu.ServerConfig{
				ServerConfigs: &compute.ServerConfigs{
					PreferRegion: GlobalCloudregion.GetId(),
					PreferZone:   GlobalZone.GetId(),
					PreferHost:   "host01",
					Hypervisor:   "hypervisor",
					ResourceType: "shared",
					InstanceType: "ecs.g1.c1m1",
					Sku:          "ecs.g1.c1m1",
					Backup:       false,
					Count:        1,
					Disks: []*compute.DiskConfig{
						{
							Backend:  "local",
							DiskType: "sys",
							ImageId:  "CentOS7.6",
							Index:    0,
							SizeMb:   30720,
						},
						{
							Backend:  "local",
							DiskType: "data",
							Index:    1,
							SizeMb:   10240,
						},
					},
					Networks: []*compute.NetworkConfig{
						{
							Index:   0,
							Network: "network01",
							Domain:  GlobalDoamin,
						},
					},
					BaremetalDiskConfigs: []*compute.BaremetalDiskConfig{
						{
							Type:  "hybrid",
							Conf:  "none",
							Count: 0,
						},
					},
					InstanceGroupIds: []string{
						"instancegroup01",
					},
				},
				Memory:  1024,
				Ncpu:    1,
				Project: GlobalProject,
				Domain:  GlobalDoamin,
			},
		},
		PreferCandidates: []string{
			"host01",
		},
		RequiredCandidates: 1,
		InstanceGroupsDetail: map[string]*models.SGroup{
			"instancegroup01": buildInstanceGroup("instancegroup01", 1, true),
		},
	}

	t.Run("Base test", func(t *testing.T) {
		info := deepCopy(commonInfo)
		getterParam := sGetterParams{
			HostId:                 "host01",
			HostName:               "host01name",
			Domain:                 "default",
			PublicScope:            "system",
			Zone:                   buildZone("zone01", ""),
			CloudRegion:            buildCloudregion("default", "", ""),
			HostType:               api.HostHypervisorForKvm,
			Storages:               []*api.CandidateStorage{buildStorage("storage01", "", 201330)},
			Networks:               []*api.CandidateNetwork{buildNetwork("network01", "", "192.168.1.0/24")},
			TotalCPUCount:          8,
			FreeCPUCount:           8,
			TotalMemorySize:        10240,
			FreeMemorySize:         10240,
			FreeStorageSizeAnyType: 201330,
			FreeGroupCount:         1,
			Skus:                   []string{"ecs.g1.c1m1"},
		}
		netowrkNicCount := map[string]int{"nework01": 10}
		scheduler, err := core.NewGenericScheduler(buildScheduler(ctrl, netowrkNicCount, basePredicateNames...))
		if err != nil {
			t.Errorf("NewGenericScheduler: %s", err.Error())
			return
		}
		candidate := buildCandidate(ctrl, getterParam)
		_, err = scheduler.Schedule(preSchedule(info, []core.Candidater{candidate}, false))
		if err != nil {
			t.Errorf("genericScheduler.Schedule error: %s", err.Error())
		}
	})
	t.Run("Forcast schedule: no match specified network", func(t *testing.T) {
		info := deepCopy(commonInfo)
		info.PreferHost = ""
		info.PreferCandidates = []string{}
		info.InstanceGroupIds = []string{}
		info.InstanceGroupsDetail = make(map[string]*models.SGroup)
		info.Count = 2
		getterParam1 := sGetterParams{
			HostId:      "host01",
			HostName:    "host01name",
			Domain:      "default",
			PublicScope: "system",
			Zone:        buildZone("zone01", ""),
			CloudRegion: buildCloudregion("default", "", ""),
			HostType:    api.HostHypervisorForKvm,
			Storages:    []*api.CandidateStorage{buildStorage("storage01", "", 201330)},
			Networks: []*api.CandidateNetwork{
				buildNetwork("network01", "nework01name", "192.168.1.0/24"),
				buildNetwork("network02", "nework02name", "192.168.2.0/24"),
			},
			TotalCPUCount:          8,
			FreeCPUCount:           8,
			TotalMemorySize:        10240,
			FreeMemorySize:         10240,
			FreeStorageSizeAnyType: 201330,
			Skus:                   []string{"ecs.g1.c1m1"},
		}
		getterParam2 := getterParam1
		getterParam2.HostId = "host02"
		getterParam2.HostName = "host02name"
		getterParam2.Networks = []*api.CandidateNetwork{
			buildNetwork("network03", "nework03name", "192.168.3.0/24"),
			buildNetwork("network04", "nework04name", "192.168.4.0/24"),
		}
		candidates := []core.Candidater{
			buildCandidate(ctrl, getterParam1),
			buildCandidate(ctrl, getterParam2),
		}
		netowrkNicCount := map[string]int{
			"network01": 255,
			"network02": 256,
			"network03": 1,
			"network04": 1,
		}
		scheduler, err := core.NewGenericScheduler(buildScheduler(ctrl, netowrkNicCount, basePredicateNames...))
		if err != nil {
			t.Errorf("NewGenericScheduler: %s", err.Error())
			return
		}
		res, err := scheduler.Schedule(preSchedule(info, candidates, true))
		if err != nil {
			t.Errorf("genericScheduler.Schedule error: %s", err.Error())
		}
		forcastResult := &api.SchedForecastResult{
			CanCreate:       false,
			AllowCount:      1,
			ReqCount:        2,
			NotAllowReasons: []string{"Out of resource"},
			FilteredCandidates: []api.FilteredCandidate{
				{
					FilterName: "host_network",
					ID:         "host02",
					Name:       "host02name",
					Reasons: []string{
						"nework03name(network03): id/name not matched",
						"nework04name(network04): id/name not matched",
					},
				},
			},
		}
		res.ForecastResult.Candidates = nil
		assert := assert.New(t)
		assert.Equal(forcastResult, res.ForecastResult, "ForecastResult should equal")
	})
}
