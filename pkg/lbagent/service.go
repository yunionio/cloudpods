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

package lbagent

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	execlient "yunion.io/x/executor/client"
	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"

	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/ovnutils"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

func StartService() {
	opts := &Options{}
	commonOpts := &opts.CommonOptions
	{
		common_options.ParseOptions(opts, os.Args, "lbagent.conf", "lbagent")
		if len(opts.CommonConfigFile) > 0 && fileutils2.Exists(opts.CommonConfigFile) {
			log.Infof("read common config file: %s", opts.CommonConfigFile)
			commonCfg := &LbagentCommonOptions{}
			commonCfg.Config = opts.CommonConfigFile
			common_options.ParseOptions(commonCfg, []string{os.Args[0]}, "common.conf", "lbagent")
			baseOpt := opts.BaseOptions.BaseOptions
			opts.LbagentCommonOptions = *commonCfg
			// keep base options
			opts.BaseOptions.BaseOptions = baseOpt
		}
		app_common.InitAuth(commonOpts, func() {
			log.Infof("auth finished ok")
		})
	}
	if err := opts.ValidateThenInit(); err != nil {
		log.Fatalf("opts validate: %s", err)
	}

	if opts.EnableRemoteExecutor {
		execlient.Init(opts.ExecutorSocketPath)
		execlient.SetTimeoutSeconds(opts.ExecutorConnectTimeoutSeconds)
		procutils.SetRemoteExecutor()
	}

	// register lbagent
	ctx := context.Background()
	ctx = context.WithValue(ctx, appctx.APP_CONTEXT_KEY_APPNAME, "lbagent")
	lbagentId, err := register(ctx, opts)
	if err != nil {
		log.Fatalf("register lbagent failed: %s", err)
	}

	if !opts.DisableLocalVpc {
		err := ovnutils.InitOvn(opts.SOvnOptions)
		if err != nil {
			log.Fatalf("ovn init fail: %s", err)
		}
	}

	tuneSystem()

	var haproxyHelper *HaproxyHelper
	var apiHelper *ApiHelper
	var haStateWatcher *HaStateWatcher
	{
		haStateWatcher, err = NewHaStateWatcher(opts)
		if err != nil {
			log.Fatalf("init ha state watcher failed: %s", err)
		}
	}
	{
		haproxyHelper, err = NewHaproxyHelper(opts, lbagentId)
		if err != nil {
			log.Fatalf("init haproxy helper failed: %s", err)
		}
	}
	{
		apiHelper, err = NewApiHelper(opts, lbagentId)
		if err != nil {
			log.Fatalf("init api helper failed: %s", err)
		}
		apiHelper.SetHaStateProvider(haStateWatcher)
	}

	{
		wg := &sync.WaitGroup{}
		cmdChan := make(chan *LbagentCmd) // internal
		var cancelFunc context.CancelFunc
		ctx, cancelFunc = context.WithCancel(ctx)
		ctx = context.WithValue(ctx, "wg", wg)
		ctx = context.WithValue(ctx, "cmdChan", cmdChan)
		wg.Add(3)
		go haStateWatcher.Run(ctx)
		go haproxyHelper.Run(ctx)
		go apiHelper.Run(ctx)

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

func tuneSystem() {
	minMemKB := fmt.Sprintf("%d", 128*1024)
	kv := map[string]string{
		"/proc/sys/vm/swappiness":                             "0",
		"/proc/sys/vm/vfs_cache_pressure":                     "350",
		"/proc/sys/vm/min_free_kbytes":                        minMemKB,
		"/proc/sys/net/ipv4/tcp_mtu_probing":                  "2",
		"/proc/sys/net/ipv4/neigh/default/gc_thresh1":         "1024",
		"/proc/sys/net/ipv4/neigh/default/gc_thresh2":         "4096",
		"/proc/sys/net/ipv4/neigh/default/gc_thresh3":         "8192",
		"/proc/sys/net/netfilter/nf_conntrack_tcp_be_liberal": "1",
	}
	for k, v := range kv {
		sysutils.SetSysConfig(k, v)
	}
}
