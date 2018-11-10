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

type SHandlerInfo struct {
	method         string
	path           []string
	name           string
	handler        func(context.Context, http.ResponseWriter, *http.Request)
	metadata       map[string]interface{}
	tags           map[string]string
	counter2XX     handlerRequestCounter
	counter4XX     handlerRequestCounter
	counter5XX     handlerRequestCounter
	processTimeout time.Duration
	workerMan      *SWorkerManager
	skipLog        bool
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

func (hi *SHandlerInfo) SetWorkerManager(workerMan *SWorkerManager) *SHandlerInfo {
	hi.workerMan = workerMan
	return hi
}

func (hi *SHandlerInfo) SetSkipLog(skip bool) *SHandlerInfo {
	hi.skipLog = skip
	return hi
}
