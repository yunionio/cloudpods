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
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
)

func initStorageClass() {
	scCmd := initK8sClusterResource("storageclass", k8s.Storageclass)
	scCmd.Perform("set-default", new(o.ClusterResourceBaseOptions))

	rbdCmd := NewK8sResourceCmd(k8s.Storageclass).SetKeyword("storageclass-ceph-csi-rbd")
	rbdCmd.Create(new(o.StorageClassCephCSIRBDCreateOptions))
	rbdCmd.PerformClass("connection-test", new(o.StorageClassCephCSIRBDTestOptions))
}
