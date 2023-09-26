package tos

import (
	"fmt"
	"os"
	"time"
)

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

const (
	EventPartSucceed = 3
	EventPartFailed  = 4
	EventPartAborted = 5 // The task needs to be interrupted in case of 403, 404, 405 errors
)

type task interface {
	do() (interface{}, error)
	getBaseInput() interface{}
}

type checkPoint interface {
	WriteToFile() error
	UpdatePartsInfo(result interface{})
	GetCheckPointFilePath() string
}

type taskGroup interface {
	// Wait 等待执行结果, success 是此次成功的 task 数量
	Wait() (success int, err error)
	// RunWorker 启动worker
	RunWorker()
	// Scheduler 分发任务
	Scheduler()
}

type postEvent interface {
	PostEvent(eventType int, result interface{}, taskErr error)
}

type taskGroupImpl struct {
	cancelHandle     chan struct{}
	abortHandle      chan struct{}
	errCh            chan error
	resultsCh        chan interface{}
	tasksCh          chan task
	routinesNum      int
	tasks            []task
	checkPoint       checkPoint
	enableCheckPoint bool
	postEvent        postEvent
}

func (t *taskGroupImpl) Wait() (int, error) {
	successNum := 0
	failNum := 0
Loop:
	for successNum+failNum < len(t.tasks) {
		select {
		case <-t.abortHandle:
			break Loop
		case <-t.cancelHandle:
			break Loop
		case part := <-t.resultsCh:
			successNum++
			t.checkPoint.UpdatePartsInfo(part)
			if t.enableCheckPoint {
				t.checkPoint.WriteToFile()
			}
			t.postEvent.PostEvent(EventPartSucceed, part, nil)
		case taskErr := <-t.errCh:
			if StatusCode(taskErr) == 403 || StatusCode(taskErr) == 404 || StatusCode(taskErr) == 405 {
				close(t.abortHandle)
				_ = os.Remove(t.checkPoint.GetCheckPointFilePath())
				t.postEvent.PostEvent(EventPartAborted, nil, taskErr)

				return successNum, fmt.Errorf("status code not service error, err:%s. ", taskErr.Error())
			}
			t.postEvent.PostEvent(EventPartFailed, nil, taskErr)
			failNum++
		}
	}
	return successNum, nil
}

func newTaskGroup(cancelHandle chan struct{}, routinesNum int, checkPoint checkPoint, postEvent postEvent, enableCheckPoint bool, tasks []task) taskGroup {
	taskBufferSize := min(routinesNum, DefaultTaskBufferSize)
	tasksCh := make(chan task, taskBufferSize)
	return &taskGroupImpl{
		cancelHandle:     cancelHandle,
		abortHandle:      make(chan struct{}),
		errCh:            make(chan error),
		resultsCh:        make(chan interface{}),
		tasksCh:          tasksCh,
		routinesNum:      routinesNum,
		tasks:            tasks,
		checkPoint:       checkPoint,
		enableCheckPoint: enableCheckPoint,
		postEvent:        postEvent,
	}
}

func (t *taskGroupImpl) RunWorker() {
	for i := 0; i < t.routinesNum; i++ {
		go t.worker()
	}

}

func (t *taskGroupImpl) Scheduler() {
	go func() {
		for _, task := range t.tasks {
			select {
			case <-t.cancelHandle:
				return
			case <-t.abortHandle:
				return
			default:
				t.tasksCh <- task
			}
		}

		close(t.tasksCh)
	}()

}

func (t *taskGroupImpl) worker() {
	for {
		select {
		case <-t.cancelHandle:
			return
		case <-t.abortHandle:
			return
		case task, ok := <-t.tasksCh:
			if !ok {
				return
			}
			result, err := task.do()
			if err != nil {
				t.errCh <- err
			}
			if result != nil {
				t.resultsCh <- result
			}
		}
	}
}

func GetUnixTimeMs() int64 {
	return ToMillis(time.Now())
}

func ToMillis(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

func StringPtr(input string) *string {
	return &input
}
