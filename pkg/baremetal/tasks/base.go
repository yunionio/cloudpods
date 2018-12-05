package tasks

import (
	"container/list"
	"context"
	"fmt"
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

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

	GetTaskQueue() *TaskQueue
	// GetData return TaskData from region
	GetData() jsonutils.JSONObject
	GetName() string

	Execute(ITask ITask, args interface{})
	SSHExecute(task ITask, remoteIP string, passwd string, args interface{})
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

type IBaremetalTask interface {
	ITask
	NeedPXEBoot() bool
}

type SBaremetalTaskBase struct {
	Baremetal    IBaremetal
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

func (task *SBaremetalTaskBase) SSHExecute(
	iTask ITask,
	remoteIP string,
	password string,
	args interface{},
) {
	iTask.SetStage(sshStageW(iTask.GetSSHStage(), remoteIP, password).Do)
	ExecuteTask(iTask, args)
}

func (task *SBaremetalTaskBase) CallNextStage(iTask ITask, stage TaskStageFunc, args interface{}) {
	iTask.SetStage(stage)
	ExecuteTask(iTask, args)
}

type IPXEBootTask interface {
	ITask
	OnPXEBoot(ctx context.Context, args interface{}) error
}

type SBaremetalPXEBootTaskBase struct {
	*SBaremetalTaskBase
}

func newBaremetalPXEBootTaskBase(
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
	pxeBootTask IPXEBootTask,
) *SBaremetalPXEBootTaskBase {
	baseTask := newBaremetalTaskBase(baremetal, taskId, data)
	self := &SBaremetalPXEBootTaskBase{
		SBaremetalTaskBase: baseTask,
	}
	//OnInitStage(pxeBootTask)
	sshConf, _ := self.Baremetal.GetSSHConfig()
	if sshConf != nil && self.Baremetal.TestSSHConfig() {
		pxeBootTask.SetStage(pxeBootTask.OnPXEBoot)
		return self
	}
	// Do soft reboot
	if data != nil && jsonutils.QueryBoolean(data, "soft_boot", false) {
		//self.SetStage(self.WaitForShutdown)
		// self.start_time = time.time()
		self.Baremetal.DoPowerShutdown(true)
		self.CallNextStage(self, self.WaitForShutdown, nil)
		return self
	}
	// shutdown and power up to PXE mode
	self.EnsurePowerShutdown(false)
	self.EnsurePowerUp("pxe")
	// this stage will be called by baremetalInstance when pxe start
	self.SetStage(pxeBootTask.OnPXEBoot)
	return self
}

func (self *SBaremetalPXEBootTaskBase) WaitForShutdown(ctx context.Context, args interface{}) error {
	return nil
}

func (self *SBaremetalPXEBootTaskBase) GetName() string {
	return "BaremetalPXEBootTaskBase"
}

func (self *SBaremetalPXEBootTaskBase) EnsurePowerShutdown(soft bool) {

}

func (self *SBaremetalPXEBootTaskBase) EnsurePowerUp(bootdev string) {
	log.Infof("[EnsurePowerUp] bootdev: %s", bootdev)
}
