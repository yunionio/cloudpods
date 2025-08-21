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
	"flag"
	"github.com/sirupsen/logrus"
	"yunion.io/x/onecloud/pkg/mcp-server/config"
	"yunion.io/x/onecloud/pkg/mcp-server/server"
)

var (
	logLevel = flag.String("log-level", "info", "日志级别 (debug, info, warn, error)")
)

func main() {
	flag.Parse()

	// 初始化日志
	logger := logrus.New()
	level, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		logger.WithError(err).Fatal("无效的日志级别")
	}
	logger.SetLevel(level)
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		logger.WithError(err).Fatal("加载配置文件失败")
	}

	// 创建服务器
	srv := server.NewServer(cfg, logger)

	// 初始化服务器
	if err := srv.Initialize(); err != nil {
		logger.WithError(err).Fatal("服务器初始化失败")
	}

	// 启动服务器
	if err := srv.Start(); err != nil {
		logger.WithError(err).Error("服务器启动失败")
	}
}
