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
	"strings"

	"yunion.io/x/log"

	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/mcp-server/climcgen"
	"yunion.io/x/onecloud/pkg/mcp-server/options"
	"yunion.io/x/onecloud/pkg/mcp-server/server"
)

const (
	serviceType    = "mcpserver"
	serviceVersion = ""
)

// StartOptions 启动时可覆盖的选项。
type StartOptions struct {
	// Instructions 若非空则完全替换默认 ServerInstructions（已按 PlatformName 生成）
	Instructions string
	// ExtraInstructions 追加到默认（或 Instructions）说明之后
	ExtraInstructions string
}

func StartService() {
	StartServiceWithOptions(nil)
}

func StartServiceWithOptions(startOpt *StartOptions) {
	opts := &options.Options
	common_options.ParseOptions(opts, os.Args, "mcpserver.conf", serviceType)

	commonOpts := &opts.CommonOptions
	if len(commonOpts.AuthURL) > 0 && len(commonOpts.AdminUser) > 0 &&
		len(commonOpts.AdminPassword) > 0 && len(commonOpts.AdminProject) > 0 {
		app_common.InitAuth(commonOpts, func() {
			log.Infof("Auth complete!!")
		})
	} else {
		log.Infof("Auth configuration incomplete, skipping auth initialization. AuthURL: %s, AdminUser: %s, AdminPasswordSet: %v, AdminProject: %s",
			commonOpts.AuthURL, commonOpts.AdminUser, len(commonOpts.AdminPassword) > 0, commonOpts.AdminProject)
	}

	common_options.StartOptionManager(opts, opts.ConfigSyncPeriodSeconds, serviceType, serviceVersion, options.OnOptionsChange)

	srvOpt := &server.NewServerOptions{
		Instructions: resolveInstructions(startOpt),
	}
	srv := server.NewServerWithOptions(srvOpt)

	if err := srv.Initialize(); err != nil {
		log.Fatalf("Fail to init mcp server: %s", err)
	}

	if err := srv.Start(); err != nil {
		log.Fatalf("Fail to start mcp server: %s", err)
	}
}

func resolveInstructions(startOpt *StartOptions) string {
	base := climcgen.BuildServerInstructions(options.ResolvedPlatformName())
	if startOpt != nil {
		if strings.TrimSpace(startOpt.Instructions) != "" {
			base = startOpt.Instructions
		}
		if extra := strings.TrimSpace(startOpt.ExtraInstructions); extra != "" {
			base = strings.TrimSpace(base) + "\n\n" + extra
		}
	}
	return base
}
