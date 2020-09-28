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

func initService() {
	cmd := NewK8sResourceCmd(k8s.Services)
	cmd.SetKeyword("service")
	cmd.Show(new(o.NamespaceResourceGetOptions))
	cmd.BatchDeleteWithParam(new(o.NamespaceResourceDeleteOptions))
	cmd.List(new(o.ServiceListOptions))
	cmd.Create(new(o.ServiceCreateOptions))
	cmd.ShowEvent()
}
