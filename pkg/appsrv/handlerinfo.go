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

package appsrv

import (
	"context"
	"net/http"
	"strings"
	"time"
)

type handlerRequestCounter struct {
	hit      int64
	duration float64
}

type TProcessTimeoutCallback func(*SHandlerInfo, *http.Request) time.Duration

type SHandlerInfo struct {
	method     string
	path       []string
	name       string
	handler    func(context.Context, http.ResponseWriter, *http.Request)
	metadata   map[string]interface{}
	tags       map[string]string
	counter2XX handlerRequestCounter
	counter4XX handlerRequestCounter
	counter5XX handlerRequestCounter
	workerMan  *SWorkerManager
	skipLog    bool

	processTimeout         time.Duration
	processTimeoutCallback TProcessTimeoutCallback
}

func (this *SHandlerInfo) FetchProcessTimeout(r *http.Request) time.Duration {
	if this.processTimeoutCallback != nil {
		tm := this.processTimeoutCallback(this, r)
		if tm < this.processTimeout {
			tm = this.processTimeout
		}
		return tm
	} else {
		return this.processTimeout
	}
}

func (this *SHandlerInfo) SetProcessTimeoutCallback(callback TProcessTimeoutCallback) {
	this.processTimeoutCallback = callback
}

func (this *SHandlerInfo) GetName(params map[string]string) string {
	if len(this.name) > 0 {
		return this.name
	}
	path := make([]string, len(this.path)+1)
	path[0] = strings.ToLower(this.method)
	for i, seg := range this.path {
		if params != nil {
			rep, ok := params[seg]
			if ok {
				seg = rep
			}
		}
		path[i+1] = strings.ToLower(seg)
	}
	return strings.Join(path, "_")
}

func (this *SHandlerInfo) GetTags() map[string]string {
	return this.tags
}

func newHandlerInfo(method string, path []string, handler func(context.Context, http.ResponseWriter, *http.Request), metadata map[string]interface{}, name string, tags map[string]string) *SHandlerInfo {
	hand := SHandlerInfo{method: method, path: path,
		handler:  handler,
		metadata: metadata,
		name:     name,
		tags:     tags}
	return &hand
}

func (hi *SHandlerInfo) SetMethod(method string) *SHandlerInfo {
	hi.method = method
	return hi
}

func (hi *SHandlerInfo) SetPath(path string) *SHandlerInfo {
	hi.path = SplitPath(path)
	return hi
}

func (hi *SHandlerInfo) SetHandler(hand func(context.Context, http.ResponseWriter, *http.Request)) *SHandlerInfo {
	hi.handler = hand
	return hi
}

func (hi *SHandlerInfo) SetMetadata(meta map[string]interface{}) *SHandlerInfo {
	hi.metadata = meta
	return hi
}

func (hi *SHandlerInfo) SetName(name string) *SHandlerInfo {
	hi.name = name
	return hi
}

func (hi *SHandlerInfo) SetTags(tags map[string]string) *SHandlerInfo {
	hi.tags = tags
	return hi
}

func (hi *SHandlerInfo) SetProcessTimeout(to time.Duration) *SHandlerInfo {
	hi.processTimeout = to
	return hi
}

func (hi *SHandlerInfo) SetProcessNoTimeout() *SHandlerInfo {
	hi.processTimeout = -1
	return hi
}

func (hi *SHandlerInfo) SetWorkerManager(workerMan *SWorkerManager) *SHandlerInfo {
	hi.workerMan = workerMan
	return hi
}

func (hi *SHandlerInfo) SetSkipLog(skip bool) *SHandlerInfo {
	hi.skipLog = skip
	return hi
}

func (hi *SHandlerInfo) GetAppParams(params map[string]string, path []string) *SAppParams {
	appParams := SAppParams{}
	appParams.Name = hi.GetName(params)
	appParams.SkipLog = hi.skipLog
	appParams.Params = params
	appParams.Path = path
	return &appParams
}
