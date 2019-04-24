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

package tasks

import (
	"container/list"
	"context"
	"fmt"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

type Queue struct {
	objList     *list.List
	objListLock *sync.Mutex
}

func NewQueue() *Queue {
	return &Queue{
		objList:     list.New(),
		objListLock: new(sync.Mutex),
	}
}

func (q *Queue) Append(obj interface{}) *Queue {
	q.objListLock.Lock()
	defer q.objListLock.Unlock()
	q.objList.PushBack(obj)
	return q
}

func (q *Queue) First() interface{} {
	q.objListLock.Lock()
	defer q.objListLock.Unlock()
	if q.objList.Len() == 0 {
		return nil
	}
	return q.objList.Front().Value
}

func (q *Queue) IsEmpty() bool {
	return q.First() == nil
}

func (q *Queue) Pop() interface{} {
	q.objListLock.Lock()
	defer q.objListLock.Unlock()
	if q.objList.Len() == 0 {
		return nil
	}
	first := q.objList.Front()
	q.objList.Remove(first)
	return first.Value
}

func (q *Queue) String() string {
	itemStrings := debugString(q.objList.Front())
	return fmt.Sprintf("%v", itemStrings)
}

func debugString(elem *list.Element) []string {
	if elem == nil {
		return nil
	}
	strings := []string{fmt.Sprintf("%v", elem.Value)}
	rest := debugString(elem.Next())
	if rest != nil {
		strings = append(strings, rest...)
	}
	return strings
}

type TaskQueue struct {
	*Queue
}

type TaskStageFunc func(ctx context.Context, args interface{}) error

type SSHTaskStageFunc func(ctx context.Context, cli *ssh.Client, args interface{}) error

type sshStageWrapper struct {
	sshStage SSHTaskStageFunc
	remoteIP string
	password string
}

func sshStageW(
	stage SSHTaskStageFunc,
	remoteIP string,
	password string,
) *sshStageWrapper {
	return &sshStageWrapper{
		sshStage: stage,
		remoteIP: remoteIP,
		password: password,
	}
}

func (sw *sshStageWrapper) Do(ctx context.Context, args interface{}) error {
	cli, err := ssh.NewClient(sw.remoteIP, 22, "root", sw.password, "")
	if err != nil {
		return err
	}
	return sw.sshStage(ctx, cli, args)
}

type ITask interface {
	// GetStage return current task stage func
	GetStage() TaskStageFunc
	// SetStage set task next execute stage func
	SetStage(stage TaskStageFunc)

	// GetSSHStage return current task ssh stage func
	GetSSHStage() SSHTaskStageFunc
	// SetSSHStage set task next execute ssh stage func
	SetSSHStage(stage SSHTaskStageFunc)

	// GetTaskId return remote service task id
	GetTaskId() string
	GetClientSession() *mcclient.ClientSession

	GetTaskQueue() *TaskQueue
	// GetData return TaskData from region
	GetData() jsonutils.JSONObject
	GetName() string

	Execute(ITask ITask, args interface{})
	SetSSHStageParams(task ITask, remoteIP string, passwd string)
	SSHExecute(task ITask, remoteIP string, passwd string, args interface{})
	NeedPXEBoot() bool
}

func NewTaskQueue() *TaskQueue {
	return &TaskQueue{
		Queue: NewQueue(),
	}
}

func (q *TaskQueue) GetTask() ITask {
	if q.IsEmpty() {
		return nil
	}
	return q.First().(ITask)
}

func (q *TaskQueue) PopTask() ITask {
	if q.IsEmpty() {
		return nil
	}
	return q.Pop().(ITask)
}

func (q *TaskQueue) AppendTask(task ITask) *TaskQueue {
	log.Infof("Append task %s", task.GetName())
	q.Append(task)
	return q
}

type TaskFactory func(bm IBaremetal, taskId string, data jsonutils.JSONObject) (ITask, error)

type SBaremetalTaskBase struct {
	Baremetal    IBaremetal
	userCred     mcclient.TokenCredential
	stageFunc    TaskStageFunc
	sshStageFunc SSHTaskStageFunc
	taskId       string
	data         jsonutils.JSONObject
}

func newBaremetalTaskBase(
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) *SBaremetalTaskBase {
	task := &SBaremetalTaskBase{
		Baremetal: baremetal,
		taskId:    taskId,
		data:      data,
	}
	return task
}

func (task *SBaremetalTaskBase) GetTaskQueue() *TaskQueue {
	return task.Baremetal.GetTaskQueue()
}

func (task *SBaremetalTaskBase) GetTaskId() string {
	return task.taskId
}

func (task *SBaremetalTaskBase) GetData() jsonutils.JSONObject {
	return task.data
}

func (task *SBaremetalTaskBase) GetStage() TaskStageFunc {
	return task.stageFunc
}

func (task *SBaremetalTaskBase) GetSSHStage() SSHTaskStageFunc {
	return task.sshStageFunc
}

func (task *SBaremetalTaskBase) SetStage(stage TaskStageFunc) {
	task.stageFunc = stage
}

func (task *SBaremetalTaskBase) SetSSHStage(stage SSHTaskStageFunc) {
	task.sshStageFunc = stage
}

func (task *SBaremetalTaskBase) Execute(iTask ITask, args interface{}) {
	ExecuteTask(iTask, args)
}

func (task *SBaremetalTaskBase) SetSSHStageParams(iTask ITask, remoteIP string, password string) {
	iTask.SetStage(sshStageW(iTask.GetSSHStage(), remoteIP, password).Do)
}

func (task *SBaremetalTaskBase) SSHExecute(
	iTask ITask,
	remoteIP string,
	password string,
	args interface{},
) {
	//iTask.SetStage(sshStageW(iTask.GetSSHStage(), remoteIP, password).Do)
	task.SetSSHStageParams(iTask, remoteIP, password)
	ExecuteTask(iTask, args)
}

//func (task *SBaremetalTaskBase) CallNextStage(iTask ITask, stage TaskStageFunc, args interface{}) {
//iTask.SetStage(stage)
//ExecuteTask(iTask, args)
//}

func (task *SBaremetalTaskBase) GetClientSession() *mcclient.ClientSession {
	return task.Baremetal.GetClientSession()
}

func (self *SBaremetalTaskBase) EnsurePowerShutdown(soft bool) error {
	log.Infof("EnsurePowerShutdown: soft=%v", soft)
	status, err := self.Baremetal.GetPowerStatus()
	if err != nil {
		return err
	}
	startTime := time.Now()
	maxWait := 60 * time.Second
	for status == "" || status == types.POWER_STATUS_ON {
		if time.Since(startTime).Seconds() >= maxWait.Seconds() && soft {
			soft = false
		}
		err = self.Baremetal.DoPowerShutdown(soft)
		if err != nil {
			log.Errorf("DoPowerShutdown: %v", err)
		}
		time.Sleep(20 * time.Second)
		status, err = self.Baremetal.GetPowerStatus()
		if err != nil {
			log.Errorf("GetPowerStatus: %v", err)
		}
	}
	if status != types.POWER_STATUS_OFF {
		return fmt.Errorf("Baremetal invalid status %s for shutdown", status)
	}
	return nil
}

func (self *SBaremetalTaskBase) EnsurePowerUp() error {
	log.Infof("EnsurePowerUp: bootdev=pxe")
	status, err := self.Baremetal.GetPowerStatus()
	if err != nil {
		return err
	}
	for status == "" || status == types.POWER_STATUS_OFF {
		if status == types.POWER_STATUS_OFF {
			err = self.Baremetal.DoPXEBoot()
			if err != nil {
				return err
			}
		}
		status, err = self.Baremetal.GetPowerStatus()
		if err != nil {
			return err
		}
		if status == "" || status == types.POWER_STATUS_OFF {
			time.Sleep(40 * time.Second)
			status, err = self.Baremetal.GetPowerStatus()
			if err != nil {
				return err
			}
		}
	}
	if status != types.POWER_STATUS_ON {
		return fmt.Errorf("Baremetal invalid restart status: %s", status)
	}
	return nil
}

func (self *SBaremetalTaskBase) NeedPXEBoot() bool {
	return false
}

type IPXEBootTask interface {
	ITask
	OnPXEBoot(ctx context.Context, cli *ssh.Client, args interface{}) error
}

type SBaremetalPXEBootTaskBase struct {
	*SBaremetalTaskBase
	pxeBootTask IPXEBootTask
	startTime   time.Time
}

func newBaremetalPXEBootTaskBase(
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) *SBaremetalPXEBootTaskBase {
	baseTask := newBaremetalTaskBase(baremetal, taskId, data)
	self := &SBaremetalPXEBootTaskBase{
		SBaremetalTaskBase: baseTask,
	}
	return self

}

func (self *SBaremetalPXEBootTaskBase) InitPXEBootTask(pxeBootTask IPXEBootTask, data jsonutils.JSONObject) (*SBaremetalPXEBootTaskBase, error) {
	self.pxeBootTask = pxeBootTask
	//OnInitStage(pxeBootTask)
	sshConf, _ := self.Baremetal.GetSSHConfig()
	if sshConf != nil && self.Baremetal.TestSSHConfig() {
		pxeBootTask.SetSSHStage(pxeBootTask.OnPXEBoot)
		pxeBootTask.SetSSHStageParams(pxeBootTask, sshConf.RemoteIP, sshConf.Password)
		return self, nil
	}
	// Do soft reboot
	if data != nil && jsonutils.QueryBoolean(data, "soft_boot", false) {
		self.startTime = time.Now()
		if err := self.Baremetal.DoPowerShutdown(true); err != nil {
			log.Errorf("DoPowerShutdown error: %v", err)
		}
		//self.CallNextStage(self, self.WaitForShutdown, nil)
		self.SetStage(self.WaitForShutdown)

		return self, nil
	}
	// shutdown and power up to PXE mode
	if err := self.EnsurePowerShutdown(false); err != nil {
		return self, fmt.Errorf("EnsurePowerShutdown: %v", err)
	}
	if err := self.EnsurePowerUp(); err != nil {
		return self, fmt.Errorf("EnsurePowerUp to pxe: %v", err)
	}
	// this stage will be called by baremetalInstance when pxe start notify
	self.SetSSHStage(pxeBootTask.OnPXEBoot)
	return self, nil
}

func (self *SBaremetalPXEBootTaskBase) NeedPXEBoot() bool {
	return true
}

func (self *SBaremetalPXEBootTaskBase) WaitForShutdown(ctx context.Context, args interface{}) error {
	self.SetStage(self.OnStopComplete)
	status, err := self.Baremetal.GetPowerStatus()
	if err != nil {
		return err
	}
	if status == types.POWER_STATUS_OFF {
		self.Execute(self, nil)
	} else if time.Since(self.startTime) >= 90*time.Second {
		err = self.Baremetal.DoPowerShutdown(false)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SBaremetalPXEBootTaskBase) OnStopComplete(ctx context.Context, args interface{}) error {
	err := self.EnsurePowerUp()
	if err != nil {
		return err
	}
	self.SetSSHStage(self.pxeBootTask.OnPXEBoot)
	return nil
}

func (self *SBaremetalPXEBootTaskBase) GetName() string {
	return "BaremetalPXEBootTaskBase"
}
