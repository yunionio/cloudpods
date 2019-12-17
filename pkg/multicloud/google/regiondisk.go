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

package google

import (
	"fmt"
	"time"
)

type SRegionDisk struct {
	storage *SStorage

	Id                     string
	CreationTimestamp      time.Time
	Name                   string
	SizeGB                 int
	Zone                   string
	Status                 string
	SelfLink               string
	Type                   string
	LastAttachTimestamp    time.Time
	LastDetachTimestamp    time.Time
	LabelFingerprint       string
	PhysicalBlockSizeBytes string
	Kind                   string
}

func (region *SRegion) GetRegionDisks(storageType string, maxResults int, pageToken string) ([]SRegionDisk, error) {
	disks := []SRegionDisk{}
	params := map[string]string{}
	if len(storageType) > 0 {
		params["filter"] = fmt.Sprintf(`type="%s/%s/projects/%s/regions/%s/diskTypes/%s"`, GOOGLE_COMPUTE_DOMAIN, GOOGLE_API_VERSION, region.GetProjectId(), region.Name, storageType)
	}
	resource := fmt.Sprintf("regions/%s/disks", region.Name)
	return disks, region.List(resource, params, maxResults, pageToken, &disks)
}

func (region *SRegion) GetRegionDisk(id string) (*SRegionDisk, error) {
	disk := &SRegionDisk{}
	return disk, region.Get(id, disk)
}
