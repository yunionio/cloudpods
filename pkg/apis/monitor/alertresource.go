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

package monitor

import (
	"yunion.io/x/onecloud/pkg/apis"
)

type AlertResourceType string

const (
	// AlertResourceTypeNode means onecloud system infrastructure controller or host node
	AlertResourceTypeNode AlertResourceType = "host"
	// AlertResourceTypeCloudaccount means cloudaccount resource
	AlertResourceTypeCloudaccount AlertResourceType = "cloudaccount"
	// AlertResourceTypeVM means virtual machine guest resource
	AlertResourceTypeVM AlertResourceType = "vm"
)

type AlertResourceCreateInput struct {
	apis.StandaloneResourceCreateInput

	Type AlertResourceType `json:"type"`
}

type AlertResourceListInput struct {
	apis.StandaloneResourceListInput
	Type string `json:"type"`
}

type AlertResourceAttachInput struct {
	apis.Meta

	AlertResourceId string    `json:"alert_resource_id"`
	AlertId         string    `json:"alert_id"`
	AlertRecordId   string    `json:"alert_record_id"`
	Data            EvalMatch `json:"data"`
}

type AlertResourceDetails struct {
	apis.StandaloneResourceDetails
	Count int               `json:"count"`
	Tags  map[string]string `json:"tags"`
}

type AlertResourceAlertListInput struct {
	apis.JointResourceBaseListInput
	AlertResourceId string `json:"alert_resource_id"`
	AlertId         string `json:"alert_id"`
}

type AlertResourceJointBaseDetails struct {
	apis.JointResourceBaseDetails

	AlertResource string            `json:"alert_resource"`
	Type          AlertResourceType `json:"type"`
}

type AlertResourceAlertDetails struct {
	AlertResourceJointBaseDetails
	Alert                    string                      `json:"alert"`
	AlertType                string                      `json:"alert_type"`
	Level                    string                      `json:"level"`
	CommonAlertMetricDetails []*CommonAlertMetricDetails `json:"common_alert_metric_details"`
}
