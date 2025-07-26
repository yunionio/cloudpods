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

package guestman

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

func (m *SGuestManager) GetImageDeps(storageType string) []string {
	if len(storageType) == 0 {
		storageType = api.STORAGE_LOCAL
	}

	images := stringutils2.NewSortedStrings(nil)

	m.Servers.Range(func(k, v interface{}) bool {
		inst := v.(GuestRuntimeInstance)
		imgs := inst.GetDependsImageIds(storageType)
		images = images.Append(imgs...)
		return true
	})

	return images
}

func (kvm *SKVMGuestInstance) GetDependsImageIds(storageType string) []string {
	images := stringutils2.NewSortedStrings(nil)
	for i := range kvm.Desc.Disks {
		disk := kvm.Desc.Disks[i]
		if len(disk.StorageType) == 0 {
			disk.StorageType = api.STORAGE_LOCAL
		}
		if disk.StorageType != storageType {
			continue
		}
		if disk.TemplateId != "" {
			images = images.Append(disk.TemplateId)
		}
	}
	return images
}

func (pod *sPodGuestInstance) GetDependsImageIds(storageType string) []string {
	images := stringutils2.NewSortedStrings(nil)

	for i := range pod.Desc.Containers {
		container := pod.Desc.Containers[i]
		for j := range container.Spec.VolumeMounts {
			volumeMount := container.Spec.VolumeMounts[j]
			if volumeMount.Disk != nil {
				if volumeMount.Disk.TemplateId != "" {
					images = images.Append(volumeMount.Disk.TemplateId)
				}
				for k := range volumeMount.Disk.PostOverlay {
					postOverlay := volumeMount.Disk.PostOverlay[k]
					if postOverlay.Image != nil && postOverlay.Image.Id != "" {
						images = images.Append(postOverlay.Image.Id)
					}
				}
			}
		}
	}
	return images
}
