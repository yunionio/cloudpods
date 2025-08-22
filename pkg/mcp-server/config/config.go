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

package config

import (
	"fmt"
	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	MCP      MCPConfig      `mapstructure:"mcp"`
	External ExternalConfig `mapstructure:"external"`
	Log      LogConfig      `mapstructure:"log"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type MCPConfig struct {
	Name        string `mapstructure:"name"`
	Version     string `mapstructure:"version"`
	Description string `mapstructure:"description"`
}

type ExternalConfig struct {
	Cloudpods CloudpodsConfig `mapstructure:"cloudpods"`
}

type CloudpodsConfig struct {
	BaseURL   string `mapstructure:"base_url" yaml:"base_url"`
	AccessKey string `mapstructure:"access_key" yaml:"access_key"`
	SecretKey string `mapstructure:"secret_key" yaml:"secret_key"`
	Timeout   int    `mapstructure:"timeout" yaml:"timeout"`

	Username string `mapstructure:"username" yaml:"username"`
	Password string `mapstructure:"password" yaml:"password"`
	Domain   string `mapstructure:"domain" yaml:"domain"`
	Project  string `mapstructure:"project" yaml:"project"`
	Region   string `mapstructure:"region" yaml:"region"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

func Load(configPath string) (*Config, error) {
	v := viper.New()
	v.SetConfigType("properties") // 显式指定 properties 格式

	// 如果指定了配置文件路径，则使用该路径
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// 否则使用默认路径
		v.SetConfigName("config")                 // 主配置文件名（无扩展名）
		v.AddConfigPath("/etc/yunion/mcp-server") // 系统配置文件路径
		v.AddConfigPath("./config")               // 配置文件路径（当前目录下的 conf 目录）
		v.AddConfigPath(".")                      // 备用路径：当前目录
	}

	v.AutomaticEnv()

	v.SetDefault("server.host", "localhost")
	v.SetDefault("server.port", 12001)
	v.SetDefault("mcp.name", "cloudpods-mcp-server")
	v.SetDefault("mcp.version", "1.0.0")
	v.SetDefault("mcp.description", "the mcp server of the cloudpods server")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("external.cloudpods.base_url", "https://<ip_or_domain_of_apigatway>/api/s/identity/v3")
	v.SetDefault("external.cloudpods.timeout", 30)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("fail to load config: %w", err)
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("fail to parse config: %w", err)
	}

	return &config, nil
}
