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

type SRegionStorage struct {
	region *SRegion

	CreationTimestamp time.Time
	Name              string
	Description       string
	ValidDiskSize     string
	Zone              string
	SelfLink          string
	DefaultDiskSizeGb string
	Kind              string
}

func (region *SRegion) GetRegionStorages(maxResults int, pageToken string) ([]SRegionStorage, error) {
	storages := []SRegionStorage{}
	resource := fmt.Sprintf("regions/%s/diskTypes", region.Name)
	params := map[string]string{}
	return storages, region.List(resource, params, maxResults, pageToken, &storages)
}

func (region *SRegion) GetRegionStorage(id string) (*SRegionStorage, error) {
	storage := &SRegionStorage{region: region}
	return storage, region.Get(id, storage)
}
