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
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostconsts"
	"yunion.io/x/onecloud/pkg/monitor/models"
)

const (
	NODE_TAG_HOST_KEY = "host"
)

func init() {
	models.RegisterAlertResourceDriverFactory(new(nodeDriverF))
}

type nodeDriverF struct{}

func (drvF *nodeDriverF) GetType() monitor.AlertResourceType {
	return monitor.AlertResourceTypeNode
}

func (drvF *nodeDriverF) IsEvalMatched(input monitor.EvalMatch) bool {
	tags := input.Tags
	_, hasResType := tags[hostconsts.TELEGRAF_TAG_KEY_RES_TYPE]
	if !hasResType {
		return false
	}
	_, hasHostType := tags[hostconsts.TELEGRAF_TAG_KEY_HOST_TYPE]
	if !hasHostType {
		return false
	}
	_, hasHost := tags[NODE_TAG_HOST_KEY]
	if !hasHost {
		return false
	}
	return true
}

func (drvF *nodeDriverF) GetDriver(input monitor.EvalMatch) models.IAlertResourceDriver {
	tags := input.Tags
	return &nodeDriver{
		nodeDriverF: drvF,
		match:       input,
		host:        tags[NODE_TAG_HOST_KEY],
		resType:     tags[hostconsts.TELEGRAF_TAG_KEY_RES_TYPE],
	}
}

type nodeDriver struct {
	*nodeDriverF

	host    string
	resType string
	match   monitor.EvalMatch
}

func (drv *nodeDriver) GetUniqCond() *models.AlertResourceUniqCond {
	return &models.AlertResourceUniqCond{
		Type: drv.GetType(),
		Name: drv.host,
	}
}
