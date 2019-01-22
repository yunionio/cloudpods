package tasks

import (
	"context"
	"fmt"
	"runtime/debug"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var baremetalTaskWorkerMan *appsrv.SWorkerManager

func init() {
	baremetalTaskWorkerMan = appsrv.NewWorkerManager("BaremetalTaskWorkerManager", 8, 1024, false)
}

func ExecuteTask(task ITask, args interface{}) {
	baremetalTaskWorkerMan.Run(func() {
		executeTask(task, args)
	}, nil, nil)
}

func executeTask(task ITask, args interface{}) {
	if task == nil {
		return
	}
	curStage := task.GetStage()
	if curStage == nil {
		return
	}
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Execute task panic: %v", err)
			debug.PrintStack()
			SetTaskFail(task, fmt.Errorf("%v", err))
		}
	}()
	err := curStage(context.Background(), args)
	if err != nil {
		log.Errorf("Execute task %s error: %v", task.GetName(), err)
		SetTaskFail(task, err)
	}
}

func SetTaskComplete(task ITask, data jsonutils.JSONObject) {
	taskId := task.GetTaskId()
	if taskId != "" {
		session := task.GetClientSession()
		modules.ComputeTasks.TaskComplete(session, taskId, data)
	}
	onTaskEnd(task)
}

func SetTaskFail(task ITask, err error) {
	taskId := task.GetTaskId()
	if taskId != "" {
		session := task.GetClientSession()
		modules.ComputeTasks.TaskFailed(session, taskId, err)
	}
	onTaskEnd(task)
}

func onTaskEnd(task ITask) {
	task.SetStage(nil)
	ExecuteTask(task.GetTaskQueue().PopTask(), nil)
}

func OnInitStage(task ITask) error {
	log.Infof("Start task %s", task.GetName())
	return nil
}
