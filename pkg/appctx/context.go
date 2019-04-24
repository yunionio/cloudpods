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

package appctx

import (
	"context"
	"time"

	"yunion.io/x/pkg/trace"
)

type AppContextKey string

const (
	APP_CONTEXT_KEY_DB              = AppContextKey("db")
	APP_CONTEXT_KEY_CACHE           = AppContextKey("cache")
	APP_CONTEXT_KEY_APPNAME         = AppContextKey("appname")
	APP_CONTEXT_KEY_APP             = AppContextKey("application")
	APP_CONTEXT_KEY_CUR_PATH        = AppContextKey("currentpath")
	APP_CONTEXT_KEY_CUR_ROOT        = AppContextKey("currentroot")
	APP_CONTEXT_KEY_PARAMS          = AppContextKey("parameters")
	APP_CONTEXT_KEY_METADATA        = AppContextKey("metadata")
	APP_CONTEXT_KEY_TRACE           = AppContextKey("trace")
	APP_CONTEXT_KEY_REQUEST_ID      = AppContextKey("requestid")
	APP_CONTEXT_KEY_TASK_ID         = AppContextKey("taskid")
	APP_CONTEXT_KEY_TASK_NOTIFY_URL = AppContextKey("tasknotifyurl")
	APP_CONTEXT_KEY_OBJECT_ID       = AppContextKey("objectid")
	APP_CONTEXT_KEY_OBJECT_TYPE     = AppContextKey("objecttype")
	APP_CONTEXT_KEY_START_TIME      = AppContextKey("starttime")
)

func AppContextServiceName(ctx context.Context) string {
	val := ctx.Value(APP_CONTEXT_KEY_APPNAME)
	if val != nil {
		return val.(string)
	} else {
		return ""
	}
}

func AppContextCurrentPath(ctx context.Context) []string {
	val := ctx.Value(APP_CONTEXT_KEY_CUR_PATH)
	if val != nil {
		return val.([]string)
	} else {
		return nil
	}
}

func AppContextCurrentRoot(ctx context.Context) []string {
	val := ctx.Value(APP_CONTEXT_KEY_CUR_ROOT)
	if val != nil {
		return val.([]string)
	} else {
		return nil
	}
}

func AppContextParams(ctx context.Context) map[string]string {
	val := ctx.Value(APP_CONTEXT_KEY_PARAMS)
	if val != nil {
		return val.(map[string]string)
	} else {
		return nil
	}
}

func AppContextMetadata(ctx context.Context) map[string]interface{} {
	val := ctx.Value(APP_CONTEXT_KEY_METADATA)
	if val != nil {
		return val.(map[string]interface{})
	} else {
		return nil
	}
}

func AppContextTrace(ctx context.Context) *trace.STrace {
	val := ctx.Value(APP_CONTEXT_KEY_TRACE)
	if val != nil {
		return val.(*trace.STrace)
	} else {
		return nil
	}
}

func AppContextRequestId(ctx context.Context) string {
	val := ctx.Value(APP_CONTEXT_KEY_REQUEST_ID)
	if val != nil {
		return val.(string)
	} else {
		return ""
	}
}

func AppContextTaskId(ctx context.Context) string {
	val := ctx.Value(APP_CONTEXT_KEY_TASK_ID)
	if val != nil {
		return val.(string)
	} else {
		return ""
	}
}

func AppContextTaskNotifyUrl(ctx context.Context) string {
	val := ctx.Value(APP_CONTEXT_KEY_TASK_NOTIFY_URL)
	if val != nil {
		return val.(string)
	} else {
		return ""
	}
}

func AppContextObjectID(ctx context.Context) string {
	val := ctx.Value(APP_CONTEXT_KEY_OBJECT_ID)
	if val != nil {
		return val.(string)
	} else {
		return ""
	}
}

func AppContextObjectType(ctx context.Context) string {
	val := ctx.Value(APP_CONTEXT_KEY_OBJECT_TYPE)
	if val != nil {
		return val.(string)
	} else {
		return ""
	}
}

func AppContextStartTime(ctx context.Context) time.Time {
	val := ctx.Value(APP_CONTEXT_KEY_START_TIME)
	if val != nil {
		return val.(time.Time)
	} else {
		return time.Time{}
	}
}

type AppContextData struct {
	Trace         trace.STrace
	RequestId     string
	ObjectType    string
	ObjectId      string
	TaskId        string
	TaskNotifyUrl string
	ServiceName   string
}

func (self *AppContextData) IsZero() bool {
	return len(self.TaskNotifyUrl) == 0 && len(self.TaskId) == 0 && len(self.ObjectId) == 0 && len(self.ObjectType) == 0 && len(self.RequestId) == 0 && self.Trace.IsZero() && len(self.ServiceName) == 0
}

func FetchAppContextData(ctx context.Context) AppContextData {
	tracePtr := AppContextTrace(ctx)
	requestId := AppContextRequestId(ctx)
	objectType := AppContextObjectType(ctx)
	objectId := AppContextObjectID(ctx)
	taskId := AppContextTaskId(ctx)
	taskNotifyUrl := AppContextTaskNotifyUrl(ctx)
	serviceName := AppContextServiceName(ctx)

	var trace trace.STrace
	if tracePtr != nil {
		trace = *tracePtr
	}
	return AppContextData{Trace: trace,
		RequestId:     requestId,
		ObjectType:    objectType,
		ObjectId:      objectId,
		TaskId:        taskId,
		TaskNotifyUrl: taskNotifyUrl,
		ServiceName:   serviceName,
	}
}

func (self *AppContextData) GetContext() context.Context {
	ctx := context.Background()
	if len(self.Trace.Id) > 0 {
		ctx = context.WithValue(ctx, APP_CONTEXT_KEY_TRACE, &self.Trace)
	}
	if len(self.RequestId) > 0 {
		ctx = context.WithValue(ctx, APP_CONTEXT_KEY_REQUEST_ID, self.RequestId)
	}
	if len(self.ObjectType) > 0 {
		ctx = context.WithValue(ctx, APP_CONTEXT_KEY_OBJECT_TYPE, self.ObjectType)
	}
	if len(self.ObjectId) > 0 {
		ctx = context.WithValue(ctx, APP_CONTEXT_KEY_OBJECT_ID, self.ObjectId)
	}
	if len(self.TaskId) > 0 {
		ctx = context.WithValue(ctx, APP_CONTEXT_KEY_TASK_ID, self.TaskId)
	}
	if len(self.TaskNotifyUrl) > 0 {
		ctx = context.WithValue(ctx, APP_CONTEXT_KEY_TASK_NOTIFY_URL, self.TaskNotifyUrl)
	}
	if len(self.ServiceName) > 0 {
		ctx = context.WithValue(ctx, APP_CONTEXT_KEY_APPNAME, self.ServiceName)
	}
	return ctx
}
