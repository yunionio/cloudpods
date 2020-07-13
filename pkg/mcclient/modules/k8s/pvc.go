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

package k8s

import (
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var PersistentVolumeClaims *PersistentVolumeClaimManager

type PersistentVolumeClaimManager struct {
	*NamespaceResourceManager
	statusGetter
}

func init() {
	PersistentVolumeClaims = &PersistentVolumeClaimManager{
		NamespaceResourceManager: NewNamespaceResourceManager(
			"persistentvolumeclaim", "persistentvolumeclaims",
			NewColumns("Status", "Volume", "StorageClass", "MountedBy"), NewColumns()),
		statusGetter: getStatus,
	}
	modules.Register(PersistentVolumeClaims)
}

func (m PersistentVolumeClaimManager) Get_Volume(obj jsonutils.JSONObject) interface{} {
	volume, _ := obj.GetString("volume")
	return volume
}

func (m PersistentVolumeClaimManager) Get_StorageClass(obj jsonutils.JSONObject) interface{} {
	sc, _ := obj.GetString("storageClass")
	return sc
}

func (m PersistentVolumeClaimManager) Get_MountedBy(obj jsonutils.JSONObject) interface{} {
	pods, _ := obj.GetArray("mountedBy")
	var ret []string
	for _, pod := range pods {
		podName, _ := pod.GetString()
		ret = append(ret, podName)
	}
	return strings.Join(ret, ",")
}
