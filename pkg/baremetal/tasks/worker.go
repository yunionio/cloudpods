package tasks

import (
	"context"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
)

var baremetalTaskWorkerMan *appsrv.SWorkerManager

func init() {
	baremetalTaskWorkerMan = appsrv.NewWorkerManager("BaremetalTaskWorkerManager", 8, 1024)
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
	err := curStage(context.Background(), args)
	if err != nil {
		log.Errorf("Execute task %s error: %v", task.GetName(), err)
		SetTaskFail(task, err)
	}
}

func SetTaskComplete(task ITask) {
	taskId := task.GetTaskId()
	if taskId != "" {
		// TODO: notify region complete
	}
	onTaskEnd(task)
}

func SetTaskFail(task ITask, err error) {
	taskId := task.GetTaskId()
	if taskId != "" {
		// TODO: notify region task fail
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
