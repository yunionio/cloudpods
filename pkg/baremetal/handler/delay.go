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

type ProcessFunc func(data jsonutils.JSONObject) (jsonutils.JSONObject, error)

type delayTask struct {
	process ProcessFunc
	taskId  string
	session *mcclient.ClientSession
	data    jsonutils.JSONObject
}

func newDelayTask(process ProcessFunc, session *mcclient.ClientSession, taskId string, data jsonutils.JSONObject) *delayTask {
	return &delayTask{
		process: process,
		taskId:  taskId,
		session: session,
		data:    data,
	}
}

func DelayProcess(process ProcessFunc, session *mcclient.ClientSession, taskId string, data jsonutils.JSONObject) {
	delayTaskWorkerMan.Run(func() {
		executeDelayProcess(newDelayTask(process, session, taskId, data))
	}, nil, nil)
}

func executeDelayProcess(task *delayTask) {
	ret, err := task.process(task.data)
	if err != nil {
		modules.ComputeTasks.TaskFailed(task.session, task.taskId, err)
		return
	}
	modules.ComputeTasks.TaskComplete(task.session, task.taskId, ret)
}
