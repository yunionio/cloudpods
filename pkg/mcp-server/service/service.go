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

	"yunion.io/x/log"

	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/mcp-server/options"
	"yunion.io/x/onecloud/pkg/mcp-server/server"
)

func StartService() {

	opts := &options.Options
	common_options.ParseOptions(opts, os.Args, "mcpserver.conf", "mcpserver")

	// 如果配置了认证信息，初始化 auth manager
	commonOpts := &opts.CommonOptions
	// 只有当所有必需的认证配置都存在时，才初始化 auth manager
	if len(commonOpts.AuthURL) > 0 && len(commonOpts.AdminUser) > 0 &&
		len(commonOpts.AdminPassword) > 0 && len(commonOpts.AdminProject) > 0 {
		app_common.InitAuth(commonOpts, func() {
			log.Infof("Auth complete!!")
		})
	} else {
		log.Infof("Auth configuration incomplete, skipping auth initialization. AuthURL: %s, AdminUser: %s, AdminPassword: %s, AdminProject: %s", commonOpts.AuthURL, commonOpts.AdminUser, commonOpts.AdminPassword, commonOpts.AdminProject)
	}

	// 创建服务器
	srv := server.NewServer()

	// 初始化服务器
	if err := srv.Initialize(); err != nil {
		log.Fatalf("Fail to init mcp server: %s", err)
	}

	// 启动服务器
	if err := srv.Start(); err != nil {
		log.Fatalf("Fail to start mcp server: %s", err)
	}
}
