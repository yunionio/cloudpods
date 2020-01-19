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

package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"yunion.io/x/log"

	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/util/atexit"
	"yunion.io/x/onecloud/pkg/vpcagent/options"
	_ "yunion.io/x/onecloud/pkg/vpcagent/ovn"
	"yunion.io/x/onecloud/pkg/vpcagent/worker"
)

func main() {
	defer atexit.Handle()

	opts := &options.Options{}
	commonOpts := &opts.CommonOptions
	{
		common_options.ParseOptions(opts, os.Args, "vpcagent.conf", "vpcagent")
		app_common.InitAuth(commonOpts, func() {
			log.Infof("auth finished ok")
		})
	}
	if err := opts.ValidateThenInit(); err != nil {
		log.Fatalf("opts validate: %s", err)
	}

	w := worker.NewWorker(opts)
	if w == nil {
		log.Fatalf("new worker failed")
	}

	{
		wg := &sync.WaitGroup{}
		ctx, cancelFunc := context.WithCancel(context.Background())
		ctx = context.WithValue(ctx, "wg", wg)
		wg.Add(1)
		go w.Start(ctx)

		go func() {
			sigChan := make(chan os.Signal)
			signal.Notify(sigChan, syscall.SIGINT)
			signal.Notify(sigChan, syscall.SIGTERM)
			sig := <-sigChan
			log.Infof("signal received: %s", sig)
			cancelFunc()
		}()
		wg.Wait()
	}
}
