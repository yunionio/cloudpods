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

package modules

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type HiProcInstManager struct {
	modulebase.ResourceManager
}

func (this *HiProcInstManager) GetStatistics(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	userId, _ := params.GetString("user_id")

	hiProcInstPath := fmt.Sprintf("/historic-process-instances?status=unfinished&user_id=%s", userId)
	procTaskPath := fmt.Sprintf("/process-tasks?user_id=%s", userId)

	hiProcInstObj, err := modulebase.List(this.ResourceManager, s, hiProcInstPath, "historic-process-instances")
	if err != nil {
		return nil, err
	}

	procTaskObj, err := modulebase.List(this.ResourceManager, s, procTaskPath, "process-tasks")
	if err != nil {
		return nil, err
	}

	nrHiProcInst := hiProcInstObj.Total
	nrProcTask := procTaskObj.Total

	hiProcInstCusPath := fmt.Sprintf("%s&process_definition_key=customer-service", hiProcInstPath)
	procTaskCusPath := fmt.Sprintf("%s&process_definition_key=customer-service", procTaskPath)

	hiProcInstCusObj, err := modulebase.List(this.ResourceManager, s, hiProcInstCusPath, "historic-process-instances")
	if err != nil {
		return nil, err
	}

	procTaskCusObj, err := modulebase.List(this.ResourceManager, s, procTaskCusPath, "process-tasks")
	if err != nil {
		return nil, err
	}

	hiProcInstCusTotal := hiProcInstCusObj.Total
	procTaskCusTotal := procTaskCusObj.Total

	rst := jsonutils.NewDict()
	rst.Add(jsonutils.NewInt(int64(nrHiProcInst-hiProcInstCusTotal)), "nr-historic-process-instance")
	rst.Add(jsonutils.NewInt(int64(nrProcTask-procTaskCusTotal)), "nr-process-task")
	rst.Add(jsonutils.NewInt(int64(hiProcInstCusTotal)), "nr-historic-process-instance-cus")
	rst.Add(jsonutils.NewInt(int64(procTaskCusTotal)), "nr-process-task-cus")

	return rst, nil
}

var (
	HistoricProcessInstance HiProcInstManager
)

func init() {
	HistoricProcessInstance = HiProcInstManager{NewITSMManager("historic-process-instance", "historic-process-instances",
		[]string{"id", "process_definition_key", "start_activity_id", "end_time", "duration_in_millis", "removal_time", "business_key", "end_activity_id", "process_definition_version", "delete_reason", "process_definition_id", "start_time", "start_user_id", "case_instance_id", "root_process_instance_id", "super_case_instance_id", "state", "process_definition_name", "super_process_instance_id", "tenant_id"},
		[]string{})}
	register(&HistoricProcessInstance)
}
