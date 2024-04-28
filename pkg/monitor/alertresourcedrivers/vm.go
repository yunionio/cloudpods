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

package alertresourcedrivers

import (
	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/monitor/models"
)

func init() {
	models.RegisterAlertResourceDriverFactory(new(vmDriverF))
}

type vmDriverF struct{}

func (v vmDriverF) GetType() monitor.AlertResourceType {
	return monitor.AlertResourceTypeVM
}

const (
	VM_TAG_VM_ID   = "vm_id"
	VM_TAG_VM_NAME = "vm_name"
)

func (v vmDriverF) IsEvalMatched(input monitor.EvalMatch) bool {
	tags := input.Tags
	/*resType, hasResType := tags[hostconsts.TELEGRAF_TAG_KEY_RES_TYPE]
	if !hasResType {
		return false
	}
	if !sets.NewString("guest", "agent").Has(resType) {
		return false
	}*/
	_, hasVmTag := tags[VM_TAG_VM_ID]
	_, hasVmName := tags[VM_TAG_VM_NAME]
	return hasVmTag && hasVmName
}

func (v vmDriverF) GetDriver(input monitor.EvalMatch) models.IAlertResourceDriver {
	return &guestDriver{
		vmDriverF: &v,
		match:     input,
	}
}

type guestDriver struct {
	*vmDriverF

	match monitor.EvalMatch
}

func (g guestDriver) GetUniqCond() *models.AlertResourceUniqCond {
	tags := g.match.Tags
	name := tags[VM_TAG_VM_NAME]
	return &models.AlertResourceUniqCond{
		Type: g.GetType(),
		Name: name,
	}
}
