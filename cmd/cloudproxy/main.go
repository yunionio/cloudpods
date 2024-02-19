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

	api "yunion.io/x/onecloud/pkg/apis/cloudproxy"
	common_app "yunion.io/x/onecloud/pkg/cloudcommon/app"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/cloudproxy/agent/worker"
	"yunion.io/x/onecloud/pkg/cloudproxy/options"
	"yunion.io/x/onecloud/pkg/cloudproxy/service"
	"yunion.io/x/onecloud/pkg/util/atexit"
)

func main() {
	defer atexit.Handle()

	wg := &sync.WaitGroup{}
	ctx := context.Background()
	ctx = context.WithValue(ctx, "wg", wg)
	ctx, cancelFunc := context.WithCancel(ctx)

	var (
		opts = options.Get()
	)

	common_app.InitAuth(&opts.CommonOptions, func() {
		log.Infof("Auth complete")
	})

	common_options.StartOptionManager(opts, opts.ConfigSyncPeriodSeconds, api.SERVICE_TYPE, api.SERVICE_VERSION, options.OnOptionsChange)

	wg.Add(1)
	go func() {
		defer wg.Done()
		service.StartService()
	}()

	if opts.EnableProxyAgent {
		const d = "10m"
		log.Infof("set proxy_agent_init_wait to %s", d)
		opts.Options.ProxyAgentInitWait = d
		if err := opts.Options.ValidateThenInit(); err != nil {
			log.Fatalf("proxy agent options validation: %v", err)
		}
		worker := worker.NewWorker(&opts.CommonOptions, &opts.Options)
		go func() {
			worker.Start(ctx)
			pid := os.Getpid()
			p, err := os.FindProcess(pid)
			if err != nil {
				log.Fatalf("find process of my pid %d: %v", pid, err)
			}
			p.Signal(syscall.SIGTERM)
		}()
	}
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
