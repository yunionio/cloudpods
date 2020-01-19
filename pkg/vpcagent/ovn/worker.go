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

package ovn

import (
	"context"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/vpcagent/apihelper"
	agentmodels "yunion.io/x/onecloud/pkg/vpcagent/models"
	"yunion.io/x/onecloud/pkg/vpcagent/options"
	"yunion.io/x/onecloud/pkg/vpcagent/ovnutil"
	"yunion.io/x/onecloud/pkg/vpcagent/worker"
)

type Worker struct {
	opts *options.Options

	apih *apihelper.APIHelper
}

func NewWorker(opts *options.Options) worker.IWorker {
	modelSets := agentmodels.NewModelSets()
	apih, err := apihelper.NewAPIHelper(opts, modelSets)
	if err != nil {
		return nil
	}
	w := &Worker{
		opts: opts,
		apih: apih,
	}
	return w
}

func (w *Worker) Start(ctx context.Context) {
	wg := ctx.Value("wg").(*sync.WaitGroup)
	defer func() {
		log.Infoln("ovn: worker bye")
		wg.Done()
	}()

	wg.Add(1)
	go w.apih.Start(ctx)

	tickDuration := time.Duration(w.opts.OvnWorkerCheckInterval) * time.Second
	tick := time.NewTimer(tickDuration)
	defer tick.Stop()

	var mss *agentmodels.ModelSets
	for {
		select {
		case imss := <-w.apih.ModelSets():
			log.Infof("ovn: got new data from api helper")
			mss = imss.(*agentmodels.ModelSets)
			if err := w.run(ctx, mss); err != nil {
				log.Errorf("ovn: %v", err)
			}
		case <-tick.C:
			if mss != nil {
				log.Infof("ovn: tick check")
				if err := w.run(ctx, mss); err != nil {
					log.Errorf("ovn: %v", err)
				}
			}
			tick.Reset(tickDuration)
		case <-ctx.Done():
			return
		}
	}
}

func (w *Worker) run(ctx context.Context, mss *agentmodels.ModelSets) (err error) {
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

	ovnnbctl := ovnutil.NewOvnNbCtl(w.opts.OvnNorthDatabase)
	ovndb, err := DumpOVNNorthbound(ctx, ovnnbctl)
	if err != nil {
		return err
	}

	ovndb.Mark(ctx)
	for _, vpc := range mss.Vpcs {
		ovndb.ClaimVpc(ctx, vpc)
		for _, network := range vpc.Networks {
			ovndb.ClaimNetwork(ctx, network)
			for _, guestnetwork := range network.Guestnetworks {
				ovndb.ClaimGuestnetwork(ctx, guestnetwork)
			}
		}
	}
	ovndb.Sweep(ctx)
	return nil
}
