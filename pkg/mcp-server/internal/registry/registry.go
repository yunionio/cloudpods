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

package registry

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
	"sync"
)

type Registry struct {
	lock        sync.RWMutex
	tools       map[string]*ToolRegistration
	mcpServer   *server.MCPServer
	logger      *logrus.Logger
	initialized bool
}

type ToolRegistration struct {
	Tool    mcp.Tool
	Handler server.ToolHandlerFunc
}

func NewRegistry(logger *logrus.Logger) *Registry {
	return &Registry{
		tools:  make(map[string]*ToolRegistration),
		logger: logger,
	}
}
