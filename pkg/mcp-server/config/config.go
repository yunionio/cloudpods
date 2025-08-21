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
	BilibiliAPI  BilibiliConfig  `mapstructure:"bilibili"`
	CloudpodsAPI CloudpodsConfig `mapstructure:"cloudpods"`
	Cloudpods    CloudpodsConfig `mapstructure:"cloudpods"`
}

type BilibiliConfig struct {
	BaseURL string `mapstructure:"base_url"`
	Cookie  string `mapstructure:"cookie"`
}

type CloudpodsConfig struct {
	BaseURL   string `mapstructure:"base_url" yaml:"base_url"`
	APIKey    string `mapstructure:"api_key" yaml:"api_key"`
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
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()

	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("fail to load config: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("fail to parse config: %w", err)
	}

	return &config, nil
}

func setDefaults() {
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("mcp.name", "cloudpods-mcp-server")
	viper.SetDefault("mcp.version", "1.0.0")
	viper.SetDefault("mcp.description", "the mcp server of the cloudpods server")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")
	viper.SetDefault("external.cloudpods.base_url", "https://api.cloudpods.com")
	viper.SetDefault("external.cloudpods.timeout", 30)
}
