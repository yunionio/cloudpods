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

func run(f func(be IInformerBackend) error) error {
	be := GetDefaultBackend()
	if be == nil {
		return ErrBackendNotInit
	}
	wf := func() {
		nopanic.Run(func() {
			if err := f(be); err != nil {
				log.Errorf("run informer error: %v", err)
			}
		})
	}
	informerWorkerMan.Run(wf, nil, nil)
	return nil
}
