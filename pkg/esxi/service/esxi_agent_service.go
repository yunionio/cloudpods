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

package service

import (
	"os"
	"path/filepath"

	mcesxi "yunion.io/x/cloudmux/pkg/multicloud/esxi"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	options_common "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/service"
	"yunion.io/x/onecloud/pkg/esxi"
	"yunion.io/x/onecloud/pkg/esxi/handler"
	"yunion.io/x/onecloud/pkg/esxi/options"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/deployclient"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
)

type SExsiAgentService struct {
	service.SServiceBase
}

func New() *SExsiAgentService {
	return &SExsiAgentService{}
}

func (s *SExsiAgentService) StartService() {
	options_common.ParseOptions(&options.Options, os.Args, "esxiagent.conf", "esxiagent")

	if len(options.Options.ImageCachePath) == 0 {
		options.Options.ImageCachePath = filepath.Join(options.Options.EsxiAgentPath, "image_cache")
		log.Infof("No cachepath, use default %s", options.Options.ImageCachePath)
		err := os.MkdirAll(options.Options.ImageCachePath, 0760)
		if err != nil {
			log.Fatalf("fail to create ImageCachePath %s", options.Options.ImageCachePath)
		}
	}
	if len(options.Options.AgentTempPath) == 0 {
		options.Options.AgentTempPath = filepath.Join(options.Options.EsxiAgentPath, "agent_tmp")
		log.Infof("No agent temp path, use default %s", options.Options.AgentTempPath)
		err := os.MkdirAll(options.Options.AgentTempPath, 0760)
		if err != nil {
			log.Fatalf("fail to create AgentTempPath %s", options.Options.AgentTempPath)
		}
	}

	// init lockman
	s.InitLockman()

	app_common.InitAuth(&options.Options.CommonOptions, func() {
		log.Infof("auth complete")
	})

	options_common.StartOptionManager(&options.Options.CommonOptions, options.Options.ConfigSyncPeriodSeconds, "", "", options_common.OnCommonOptionsChange)

	fsdriver.Init("")
	deployclient.Init(options.Options.DeployServerSocketPath)

	hostutils.InitWorkerManagerWithCount(options.Options.HostDelayTaskWorkerCount)
	app := app_common.InitApp(&options.Options.BaseOptions, false)
	handler.InitHandlers(app)

	mcesxi.InitEsxiConfig(options.Options.EsxiOptions)

	s.startAgent(app)

	app_common.ServeForeverWithCleanup(app, &options.Options.BaseOptions, func() {
		esxi.Stop()
	})
}

func (s *SExsiAgentService) startAgent(app *appsrv.Application) {
	err := esxi.Start(app)
	if err != nil {
		log.Fatalf("Start agent error: %v", err)
	}
}

func (s *SExsiAgentService) InitLockman() {
	log.Infof("using inmemory lockman")
	lm := lockman.NewInMemoryLockManager()
	lockman.Init(lm)
}
