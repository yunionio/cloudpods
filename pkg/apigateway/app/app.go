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

package app

import (
	"yunion.io/x/onecloud/pkg/apigateway/handler"
	"yunion.io/x/onecloud/pkg/appsrv"
)

type Application struct {
	*appsrv.Application

	AuthHandler          handler.IHandler
	MiscHandler          handler.IHandler
	K8sHandler           handler.IHandler
	ResourceHandler      handler.IHandler
	CSRFResourceHandler  handler.IHandler
	RPCHandler           handler.IHandler
	InfluxdbProxyHandler handler.IHandler
}

func NewApp(app *appsrv.Application) *Application {
	svcApp := &Application{
		Application: app,
	}
	return svcApp
}

func (app *Application) InitHandlers() *Application {
	// bind auth handlers
	app.AuthHandler = handler.NewAuthHandlers("/api/v1/auth", nil)

	// bind misc handlers
	app.MiscHandler = handler.NewMiscHandler("/api/v1/")

	// bind k8s resource handlers
	app.K8sHandler = handler.NewK8sResourceHandler("/api/v1/_raw")

	// bind restful resource handlers
	app.ResourceHandler = handler.NewResourceHandlers("/api").
		AddGet(handler.FetchAuthToken).
		AddPost(handler.FetchAuthToken).
		AddPut(handler.FetchAuthToken).
		AddPatch(handler.FetchAuthToken).
		AddDelete(handler.FetchAuthToken)

	// bind csrf handler
	app.CSRFResourceHandler = handler.NewCSRFResourceHandler("/api")

	// bind rpc handler
	app.RPCHandler = handler.NewRPCHandlers("/api").
		AddGet(handler.FetchAuthToken).
		AddPost(handler.FetchAuthToken)

	app.InfluxdbProxyHandler = handler.NewInfluxdbProxyHandler("/query")

	return app
}

func (app *Application) Bind() {
	for _, h := range []handler.IHandler{
		app.InfluxdbProxyHandler,
		app.MiscHandler,
		app.AuthHandler,
		app.K8sHandler,
		app.RPCHandler,
		app.ResourceHandler,
		app.CSRFResourceHandler,
	} {
		h.Bind(app.Application)
	}
}
