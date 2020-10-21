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
	models.RegisterAlertResourceDriverFactory(new(cloudaccountDriverF))
}

const (
	CLOUDACCOUNT_TAG_ID_KEY   = "cloudaccount_id"
	CLOUDACCOUNT_TAG_NAME_KEY = "cloudaccount_name"
)

type cloudaccountDriverF struct{}

func (drvF cloudaccountDriverF) GetType() monitor.AlertResourceType {
	return monitor.AlertResourceTypeCloudaccount
}

func (drvF cloudaccountDriverF) IsEvalMatched(input monitor.EvalMatch) bool {
	tags := input.Tags
	//_, hasId := tags[CLOUDACCOUNT_TAG_ID_KEY]
	//if !hasId {
	//	return false
	//}
	_, hasName := tags[CLOUDACCOUNT_TAG_NAME_KEY]
	if !hasName {
		return false
	}
	return true
}

func (drvF *cloudaccountDriverF) GetDriver(input monitor.EvalMatch) models.IAlertResourceDriver {
	tags := input.Tags
	return &cloudaccountDriver{
		cloudaccountDriverF: drvF,
		match:               input,
		id:                  tags[CLOUDACCOUNT_TAG_ID_KEY],
		name:                tags[CLOUDACCOUNT_TAG_NAME_KEY],
	}
}

type cloudaccountDriver struct {
	*cloudaccountDriverF
	id    string
	name  string
	match monitor.EvalMatch
}

func (drv *cloudaccountDriver) GetUniqCond() *models.AlertResourceUniqCond {
	return &models.AlertResourceUniqCond{
		Type: drv.GetType(),
		Name: drv.name,
	}
}
