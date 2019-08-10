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

package handler

import (
	"context"
	"fmt"
	"net/http"
	"path"

	"yunion.io/x/onecloud/pkg/appsrv"
)

type handleFunc func(context.Context, http.ResponseWriter, *http.Request)

type SHandler struct {
	Method         string
	Prefix         string
	MiddlewareFunc appsrv.MiddlewareFunc
	HandleFunc     handleFunc
}

func NewHandler(method string, mf appsrv.MiddlewareFunc, hf handleFunc, paths ...string) SHandler {
	h := SHandler{
		Method:         method,
		MiddlewareFunc: mf,
		HandleFunc:     hf,
	}
	h.Prefix = path.Join(paths...)
	return h
}

func (h SHandler) indexKey() string {
	return fmt.Sprintf("%s-%s", h.Method, h.Prefix)
}

func (h SHandler) Bind(app *appsrv.Application) {
	f := h.HandleFunc
	if h.MiddlewareFunc != nil {
		f = h.MiddlewareFunc(f)
	}
	app.AddHandler(h.Method, h.Prefix, f)
}

func NewMethodHandlerFactory(method string, mf appsrv.MiddlewareFunc, prefix string) func(handleFunc, ...string) SHandler {
	return func(hf handleFunc, paths ...string) SHandler {
		pPaths := []string{prefix}
		pPaths = append(pPaths, paths...)
		return NewHandler(method, mf, hf, pPaths...)
	}
}

type SHandlers struct {
	prefix   string
	handlers map[string]SHandler
}

func NewHandlers(prefix string) *SHandlers {
	return &SHandlers{
		prefix:   prefix,
		handlers: make(map[string]SHandler),
	}
}

type HandlerPath struct {
	f     handleFunc
	paths []string
}

func (f *SHandlers) GetPrefix() string {
	return f.prefix
}

func (f *SHandlers) AddByMethod(method string, mf appsrv.MiddlewareFunc, hs ...HandlerPath) *SHandlers {
	newH := NewMethodHandlerFactory(method, mf, f.prefix)
	for _, h := range hs {
		hd := newH(h.f, h.paths...)
		idx := hd.indexKey()
		f.handlers[idx] = hd
	}
	return f
}

func (f *SHandlers) Bind(app *appsrv.Application) {
	for _, h := range f.handlers {
		h.Bind(app)
	}
}

func NewHP(f handleFunc, paths ...string) HandlerPath {
	return HandlerPath{f, paths}
}

type IHandler interface {
	Bind(*appsrv.Application)
}
