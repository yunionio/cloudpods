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

package appsrv

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

type workerTask struct {
	counter int
}

func (t *workerTask) Run() {
	t.counter += 1
	time.Sleep(1 * time.Second)
}

func (t *workerTask) Dump() string {
	return fmt.Sprintf("counter: %d", t.counter)
}

func TestWorkerManager(t *testing.T) {
	enableDebug()
	startTime := time.Now()
	// end := make(chan int)
	wm := NewWorkerManager("testwm", 2, 10, false)
	task := &workerTask{
		counter: 0,
	}
	for i := 0; i < 10; i += 1 {
		wm.Run(task, nil, nil)
	}
	for wm.ActiveWorkerCount() != 0 {
		time.Sleep(time.Second)
	}
	if time.Since(startTime) < 5*time.Second {
		t.Error("Incorrect timing")
	}
}

type errWorkerTask struct {
	wg *sync.WaitGroup
}

func (t *errWorkerTask) Run() {
	t.wg.Done()
}

func (t *errWorkerTask) Dump() string {
	return ""
}

type panicWorkerTask struct {
	wg *sync.WaitGroup
}

func (t *panicWorkerTask) Run() {
	defer t.wg.Done()
	panic("panic inside worker")
}

func (t *panicWorkerTask) Dump() string {
	return ""
}

func TestWorkerManagerError(t *testing.T) {
	wm := NewWorkerManager("testwm", 2, 10, false)
	errCbFactory := func(wg *sync.WaitGroup, errMark *bool) func(error) {
		return func(error) {
			defer wg.Done()
			if errMark != nil && !*errMark {
				*errMark = true
			}
		}
	}
	t.Run("normal", func(t *testing.T) {
		task := &errWorkerTask{
			wg: &sync.WaitGroup{},
		}
		errMark := false
		errCb := errCbFactory(task.wg, &errMark)
		task.wg.Add(1)
		wm.Run(task, nil, errCb)
		task.wg.Wait()
		if errMark {
			t.Errorf("should be normal")
		}
	})
	t.Run("panic", func(t *testing.T) {
		task := &panicWorkerTask{
			wg: &sync.WaitGroup{},
		}
		errMark := false
		errCb := errCbFactory(task.wg, &errMark)
		task.wg.Add(2) // 1 for errCb
		wm.Run(task, nil, errCb)
		task.wg.Wait()
		if !errMark {
			t.Errorf("expecting error")
		}
	})
}
