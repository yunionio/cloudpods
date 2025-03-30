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

package apis

import (
	"time"

	"yunion.io/x/jsonutils"
)

type TaskBaseListInput struct {
	ProjectizedResourceListInput
	StatusResourceBaseListInput

	ObjId      []string `json:"obj_id" help:"object id filter"`
	ObjType    []string `json:"obj_type" help:"object type (in singular form) filter"`
	ObjName    []string `json:"obj_name" help:"object name filter"`
	TaskName   []string `json:"task_name" help:"task name filter"`
	IsMulti    *bool    `json:"is_multi" negative:"is_single" help:"is multi task"`
	IsComplete *bool    `json:"is_complete" negative:"not_complete" help:"is task completed, either fail or complete"`
	IsInit     *bool    `json:"is_init" negative:"not_init" help:"is task started?"`
	Stage      []string `json:"stage" help:"tasks in stages"`
	NotStage   []string `json:"not_stage" help:"tasks not in stages"`
	ParentId   []string `json:"parent_id" help:"filter tasks by parent_task_id"`
	IsRoot     *bool    `json:"is_root" help:"filter root tasks"`

	ParentTaskId string `json:"parent_task_id" help:"filter by parent_task_id"`

	SubTask *bool `json:"sub_task" help:"show sub task states"`
}

type TaskListInput struct {
	ModelBaseListInput

	TaskBaseListInput

	Id []string `json:"id" help:"id filter"`
}

type ArchivedTaskListInput struct {
	LogBaseListInput

	TaskBaseListInput

	TaskId []string `json:"task_id" help:"filter by task_id"`
}

type TaskDetails struct {
	ModelBaseDetails
	ProjectizedResourceInfo

	// 资源创建时间
	CreatedAt time.Time `json:"created_at"`
	// 资源更新时间
	UpdatedAt time.Time `json:"updated_at"`
	// 资源被更新次数
	UpdateVersion int `json:"update_version"`
	// 开始任务时间
	StartAt time.Time `json:"start_at"`
	// 完成任务时间
	EndAt time.Time `json:"end_at"`

	DomainId  string `json:"domain_id"`
	ProjectId string `json:"tenant_id"`

	Id           string
	ObjName      string
	ObjId        string
	TaskName     string
	Params       jsonutils.JSONObject
	UserCred     jsonutils.JSONObject
	Stage        string
	ParentTaskId string
}

type TaskCancelInput struct {
}
