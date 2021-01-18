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

package informer

import (
	"context"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/util/nopanic"
)

var (
	informerWorkerMan *appsrv.SWorkerManager
)

func init() {
	informerWorkerMan = appsrv.NewWorkerManager("InformerWorkerManager", 10, 10240, false)
}

/*type noCancel struct {
	ctx context.Context
}

func (c noCancel) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

func (c noCancel) Done() <-chan struct{} {
	return nil
}

func (c noCancel) Err() error {
	return nil
}

func (c noCancel) Value(key interface{}) interface{} {
	return c.ctx.Value(key)
}*/

type informerTask struct {
	be IInformerBackend
	f  func(ctx context.Context, be IInformerBackend) error
}

func (t *informerTask) Run() {
	nopanic.Run(func() {
		// outside context ignored cause of run in worker
		if err := t.f(context.Background(), t.be); err != nil {
			log.Errorf("run informer error: %v", err)
		}
	})
}

func (t *informerTask) Dump() string {
	return ""
}

func run(ctx context.Context, f func(ctx context.Context, be IInformerBackend) error) error {
	be := GetDefaultBackend()
	if be == nil {
		return ErrBackendNotInit
	}
	task := informerTask{
		f:  f,
		be: be,
	}
	informerWorkerMan.Run(&task, nil, nil)
	return nil
}
