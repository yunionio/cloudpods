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

package misc

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/modules/devtool"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/mcclient/modules/image"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	"yunion.io/x/onecloud/pkg/mcclient/modules/notify"
)

func RegisterTaskCmds(service string, manager modulebase.Manager, archivedManager modulebase.Manager) {
	type TaskListOptions struct {
		apis.TaskListInput
	}
	R(&TaskListOptions{}, fmt.Sprintf("%s-task-list", service), fmt.Sprintf("List tasks on %s server", service), func(s *mcclient.ClientSession, args *TaskListOptions) error {
		params := jsonutils.Marshal(args)
		result, err := manager.List(s, params)
		if err != nil {
			return err
		}
		printList(result, manager.GetColumns(s))
		return nil
	})

	type TaskShowOptions struct {
		ID string `help:"ID or name of the task"`
	}
	R(&TaskShowOptions{}, fmt.Sprintf("%s-task-show", service), fmt.Sprintf("Show details of a %s task", service), func(s *mcclient.ClientSession, args *TaskShowOptions) error {
		result, err := manager.GetById(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&TaskShowOptions{}, fmt.Sprintf("%s-task-cancel", service), fmt.Sprintf("Cancel a %s task", service), func(s *mcclient.ClientSession, args *TaskShowOptions) error {
		result, err := manager.PerformAction(s, args.ID, "cancel", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&TaskListOptions{}, fmt.Sprintf("%s-archived-task-list", service), fmt.Sprintf("List archived tasks on %s server", service), func(s *mcclient.ClientSession, args *TaskListOptions) error {
		params := jsonutils.Marshal(args)
		result, err := archivedManager.List(s, params)
		if err != nil {
			return err
		}
		printList(result, archivedManager.GetColumns(s))
		return nil
	})

	R(&TaskShowOptions{}, fmt.Sprintf("%s-archived-task-show", service), fmt.Sprintf("Show details of an archived %s task", service), func(s *mcclient.ClientSession, args *TaskShowOptions) error {
		params := jsonutils.NewDict()
		params.Set("task_id", jsonutils.NewString(args.ID))
		result, err := archivedManager.List(s, params)
		if err != nil {
			return err
		}
		if result.Total == 1 {
			printObject(result.Data[0])
			return nil
		} else if result.Total == 0 {
			return errors.Wrapf(errors.ErrNotFound, "not found %s", args.ID)
		} else {
			printList(result, archivedManager.GetColumns(s))
			return errors.Wrapf(errors.ErrDuplicateId, "found %d record for %s", result.Total, args.ID)
		}
	})
}

func init() {
	cmds := []struct {
		service string
		manager modulebase.Manager

		archivedManager modulebase.Manager
	}{
		{
			service:         "region",
			manager:         &compute.ComputeTasks,
			archivedManager: &compute.ArchivedComputeTasks,
		},
		{
			service:         "devtool",
			manager:         &devtool.DevtoolTasks,
			archivedManager: &devtool.ArchivedDevtoolTasks,
		},
		{
			service:         "image",
			manager:         &image.Tasks,
			archivedManager: &image.ArchivedTasks,
		},
		{
			service:         "identity",
			manager:         &identity.Tasks,
			archivedManager: &identity.ArchivedTasks,
		},
		{
			service:         "k8s",
			manager:         k8s.KubeTasks,
			archivedManager: k8s.ArchivedKubeTasks,
		},
		{
			service:         "notify",
			manager:         &notify.Tasks,
			archivedManager: &notify.ArchivedTasks,
		},
	}
	for i := range cmds {
		c := cmds[i]
		RegisterTaskCmds(c.service, c.manager, c.archivedManager)
	}
}
