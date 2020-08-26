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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var PersistentVolumes *PersistentVolumeManager

type PersistentVolumeManager struct {
	*ClusterResourceManager
}

func init() {
	PersistentVolumes = &PersistentVolumeManager{
		ClusterResourceManager: NewClusterResourceManager("persistentvolume", "persistentvolumes",
			NewColumns("StorageClass", "Claim", "AccessModes"),
			NewColumns()),
	}

	modules.Register(PersistentVolumes)
}

func (m PersistentVolumeManager) Get_StorageClass(obj jsonutils.JSONObject) interface{} {
	sc, _ := obj.GetString("storageClass")
	return sc
}

func (m PersistentVolumeManager) Get_Claim(obj jsonutils.JSONObject) interface{} {
	claim, _ := obj.GetString("claim")
	return claim
}

func (m PersistentVolumeManager) Get_AccessModes(obj jsonutils.JSONObject) interface{} {
	modes, _ := obj.(*jsonutils.JSONDict).GetArray("accessModes")
	return modes
}
