package appsrv

import (
	"context"
	"net/http"
	"strings"
)

type handlerRequestCounter struct {
	hit      int64
	duration float64
}

type handlerInfo struct {
	method     string
	path       []string
	name       string
	handler    func(context.Context, http.ResponseWriter, *http.Request)
	metadata   map[string]interface{}
	tags       map[string]string
	counter2XX handlerRequestCounter
	counter4XX handlerRequestCounter
	counter5XX handlerRequestCounter
}

func (this *handlerInfo) GetName(params map[string]string) string {
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

func (this *handlerInfo) GetTags() map[string]string {
	return this.tags
}

func newHandlerInfo(method string, path []string, handler func(context.Context, http.ResponseWriter, *http.Request), metadata map[string]interface{}, name string, tags map[string]string) *handlerInfo {
	hand := handlerInfo{method: method, path: path,
		handler:  handler,
		metadata: metadata,
		name:     name,
		tags:     tags}
	return &hand
}
