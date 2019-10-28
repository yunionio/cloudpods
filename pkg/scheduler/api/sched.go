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

package api

import (
	"net/http" //"yunion.io/x/jsonutils"

	"yunion.io/x/log"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/cmdline"
	"yunion.io/x/onecloud/pkg/compute/models"
	o "yunion.io/x/onecloud/pkg/scheduler/options"
)

type SchedInfo struct {
	*api.ScheduleInput

	Tag         string   `json:"tag"`
	Type        string   `json:"type"`
	IsContainer bool     `json:"is_container"`
	Candidates  []string `json:"candidates"`

	IgnoreFilters         map[string]bool `json:"ignore_filters"`
	IsSuggestion          bool            `json:"suggestion"`
	ShowSuggestionDetails bool            `json:"suggestion_details"`
	Raw                   string
}

func FetchSchedInfo(req *http.Request) (*SchedInfo, error) {
	body, err := appsrv.FetchJSON(req)
	if err != nil {
		return nil, err
	}

	input, err := cmdline.FetchScheduleInputByJSON(body)
	if err != nil {
		return nil, err
	}

	input = models.ApplySchedPolicies(input)

	data := NewSchedInfo(input)

	domainId := data.Domain
	for _, net := range data.Networks {
		if net.Domain == "" {
			net.Domain = domainId
		}
	}

	return data, nil
}

func NewSchedInfo(input *api.ScheduleInput) *SchedInfo {
	data := new(SchedInfo)
	data.ScheduleInput = input

	if data.Hypervisor == "" || data.Hypervisor == SchedTypeKvm {
		data.Hypervisor = HostHypervisorForKvm
	}

	candidates := make([]string, 0)
	if !data.Backup && data.PreferHost != "" {
		candidates = append(candidates, data.PreferHost)
	}

	if data.ResourceType == "" {
		data.ResourceType = computeapi.HostResourceTypeShared
	}

	if len(data.BaremetalDiskConfigs) == 0 {
		defaultConfs := []*computeapi.BaremetalDiskConfig{&computeapi.BaremetalDefaultDiskConfig}
		data.BaremetalDiskConfigs = defaultConfs
		log.V(4).Warningf("No baremetal_disk_config info found in json, use default baremetal disk config: %#v", defaultConfs)
	}

	data.Candidates = candidates

	data.reviseData()

	return data
}

func (data *SchedInfo) reviseData() {
	for _, d := range data.Disks {
		if len(d.Backend) == 0 {
			d.Backend = computeapi.STORAGE_LOCAL
		}
	}

	input := data.ScheduleInput

	if input.Count <= 0 {
		data.Count = 1
	}

	ignoreFilters := make(map[string]bool, len(input.IgnoreFilters))
	for _, filter := range input.IgnoreFilters {
		ignoreFilters[filter] = true
	}
	data.IgnoreFilters = ignoreFilters

	if data.SuggestionLimit == 0 {
		data.SuggestionLimit = int64(o.GetOptions().SchedulerTestLimit)
	}

	data.Raw = input.JSON(input).String()
}

func (d *SchedInfo) SkipDirtyMarkHost() bool {
	skipByHypervisor := d.IsContainer || d.Hypervisor == SchedTypeContainer
	skipByBackup := d.Backup
	return skipByHypervisor || skipByBackup
}

func (d *SchedInfo) GetCandidateHostTypes() []string {
	switch d.Hypervisor {
	case SchedTypeContainer:
		return []string{HostTypeKubelet, HostHypervisorForKvm}
	default:
		return []string{d.Hypervisor}
	}
}

func (d *SchedInfo) getDiskSize(backend string) int64 {
	total := int64(0)
	for _, disk := range d.Disks {
		if disk.Backend == backend {
			total += int64(disk.SizeMb)
		}
	}

	return total
}

func (d *SchedInfo) AllDiskBackendSize() map[string]int64 {
	backendSizeMap := make(map[string]int64, len(d.Disks))
	for _, disk := range d.Disks {
		newSize := int64(disk.SizeMb)
		if size, ok := backendSizeMap[disk.Backend]; ok {
			newSize += size
		}

		backendSizeMap[disk.Backend] = newSize
	}

	return backendSizeMap
}

type SchedResultItem interface{}

type SchedResult struct {
	Items []SchedResultItem `json:"scheduler"`
}

type SchedSuccItem struct {
	Candidate interface{} `json:"candidate"`
}

type SchedErrItem struct {
	Error string `json:"error"`
}

type SchedNormalResultItem struct {
	ID   string                 `json:"id"`
	Name string                 `json:"name"`
	Data map[string]interface{} `json:"data"`
}

type SchedBackupResultItem struct {
	MasterID string `json:"master_id"`
	SlaveID  string `json:"slave_id"`
}

type SchedTestResult struct {
	Data   interface{} `json:"data"`
	Total  int64       `json:"total"`
	Limit  int64       `json:"limit"`
	Offset int64       `json:"offset"`
}

type ForecastFilter struct {
	Filter   string   `json:"filter"`
	Messages []string `json:"messages"`
	Count    int64    `json:"count"`
}

type ForecastResult struct {
	Candidate string `json:"candidate"`
	Count     int64  `json:"count"`
	Capacity  int64  `json:"capacity"`
}

type SchedForecastResult struct {
	CanCreate bool                     `json:"can_create"`
	Filters   []*ForecastFilter        `json:"filters"`
	Results   []*api.CandidateResource `json:"results"`
}
