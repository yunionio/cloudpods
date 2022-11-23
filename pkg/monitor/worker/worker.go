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

package worker

import (
	"context"
	"runtime"
	"runtime/debug"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apihelper"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/monitor/models"
	"yunion.io/x/onecloud/pkg/monitor/options"
	"yunion.io/x/onecloud/pkg/vpcagent/worker"
)

type Worker struct {
	opts *options.AlerterOptions

	apih *apihelper.APIHelper
}

func NewWorker(opts *options.AlerterOptions) (worker.IWorker, error) {
	man := models.MonitorResourceManager
	modelSets := man.GetModelSets()
	apiOpts := &apihelper.Options{
		CommonOptions:        opts.CommonOptions,
		SyncIntervalSeconds:  opts.APISyncIntervalSeconds,
		RunDelayMilliseconds: opts.APIRunDelayMilliseconds,
		ListBatchSize:        opts.APIListBatchSize,
		IncludeDetails:       true,
		IncludeOtherCloudEnv: true,
	}
	apih, err := apihelper.NewAPIHelper(apiOpts, modelSets)
	if err != nil {
		return nil, errors.Wrap(err, "NewAPIHelper")
	}
	man.SetAPIHelper(apih)
	w := &Worker{
		opts: opts,
		apih: apih,
	}
	return w, nil
}

func (w *Worker) Start(ctx context.Context, app *appsrv.Application, prefix string) {
	defer func() {
		log.Infoln("monitor resource: worker bye")
	}()

	log.Infoln("start to get api Resource")
	go w.apih.Start(ctx, nil, "")

	var mss *models.MonitorResModelSets
	for {
		select {
		case imss := <-w.apih.ModelSets():
			log.Infof("monitorRes: got new data from api helper")
			mss = imss.(*models.MonitorResModelSets)
			if err := w.run(ctx, mss); err != nil {
				log.Errorf("monitorResWork err: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (w *Worker) run(ctx context.Context, mss *models.MonitorResModelSets) (err error) {
	defer func() {
		if panicVal := recover(); panicVal != nil {
			if panicErr, ok := panicVal.(runtime.Error); ok {
				err = errors.Wrap(panicErr, string(debug.Stack()))
			} else if panicErr, ok := panicVal.(error); ok {
				err = panicErr
			} else {
				panic(panicVal)
			}
		}
	}()
	err = models.MonitorResourceManager.SyncResources(ctx, mss)
	if err != nil {
		return err
	}
	return nil
}
