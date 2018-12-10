package handler

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var delayTaskWorkerMan *appsrv.SWorkerManager

func init() {
	delayTaskWorkerMan = appsrv.NewWorkerManager("DelayTaskWorkerManager", 8, 1024)
}

type ProcessFunc func() (jsonutils.JSONObject, error)

type delayTask struct {
	process ProcessFunc
	taskId  string
	session *mcclient.ClientSession
}

func newDelayTask(process ProcessFunc, session *mcclient.ClientSession, taskId string) *delayTask {
	return &delayTask{
		process: process,
		taskId:  taskId,
		session: session,
	}
}

func DelayProcess(process ProcessFunc, session *mcclient.ClientSession, taskId string) {
	delayTaskWorkerMan.Run(func() {
		executeDelayProcess(newDelayTask(process, session, taskId))
	}, nil, nil)
}

func executeDelayProcess(task *delayTask) {
	ret, err := task.process()
	if err != nil {
		modules.ComputeTasks.TaskFailed(task.session, task.taskId, err)
		return
	}
	modules.ComputeTasks.TaskComplete(task.session, task.taskId, ret)
}
