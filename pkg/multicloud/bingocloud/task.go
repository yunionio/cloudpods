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

package bingocloud

type MetaRequest struct {
	MethodName string `json:"method_name"`
}

type EntityList struct {
	EntityID   string      `json:"entity_id"`
	EntityType string      `json:"entity_type"`
	EntityName interface{} `json:"entity_name"`
}

type MetaResponse struct {
	ErrorCode   int    `json:"error_code"`
	ErrorDetail string `json:"error_detail"`
}

type STask struct {
	UUID                 string       `json:"uuid"`
	MetaRequest          MetaRequest  `json:"meta_request"`
	MetaResponse         MetaResponse `json:"meta_response,omitempty"`
	CreateTimeUsecs      int64        `json:"create_time_usecs"`
	StartTimeUsecs       int64        `json:"start_time_usecs"`
	CompleteTimeUsecs    int64        `json:"complete_time_usecs"`
	LastUpdatedTimeUsecs int64        `json:"last_updated_time_usecs"`
	EntityList           []EntityList `json:"entity_list,omitempty"`
	OperationType        string       `json:"operation_type"`
	Message              string       `json:"message"`
	PercentageComplete   int          `json:"percentage_complete"`
	ProgressStatus       string       `json:"progress_status"`
	ClusterUUID          string       `json:"cluster_uuid"`
	SubtaskUUIDList      []string     `json:"subtask_uuid_list,omitempty"`
}

func (self *SRegion) GetTasks() ([]STask, error) {
	tasks := []STask{}
	return tasks, self.post("tasks/list", nil, &tasks)
}

func (self *SRegion) GetTask(id string) (*STask, error) {
	return self.client.getTask(id)
}
