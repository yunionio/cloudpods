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

package trace

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
)

type TraceKind string

const UNKNOWN_SERVICE_NAME string = "(unknown_service)"

const (
	X_YUNION_PEER_SERVICE_NAME = "X-Yunion-Peer-Service-Name"
	// X_YUNION_SERVICE_NAME   = "X-Yunion-Service-Name"
	X_YUNION_TRACE_ID    = "X-Yunion-STrace-Id"
	X_YUNION_SPAN_NAME   = "X-Yunion-Span-Name"
	X_YUNION_PARENT_ID   = "X-Yunion-Parent-Id"
	X_YUNION_SPAN_ID     = "X-Yunion-Span-Id"
	X_YUNION_TRACE_DEBUG = "X-Yunion-STrace-Debug"
	X_YUNION_REMOTE_ADDR = "X-Yunion-Remote-Addr"
	X_YUNION_TRACE_TAG   = "X-Yunion-STrace-Tag-"
)

const (
	TRACE_KIND_CLIENT TraceKind = "CLIENT"
	TRACE_KIND_SERVER TraceKind = "SERVER"
)

type STraceEndpoint struct {
	ServiceName string
	Addr        string
	Port        int
}

type STrace struct {
	TraceId        string
	Name           string
	ParentId       string
	Id             string
	clientId       int // valid for ServerTrace
	Kind           TraceKind
	Timestamp      time.Time
	Duration       time.Duration
	Debug          bool
	Shared         bool
	LocalEndpoint  STraceEndpoint
	RemoteEndpoint STraceEndpoint
	Tags           map[string]string
}

func (tr *STrace) IsZero() bool {
	return len(tr.TraceId) == 0
}

func (tr *STrace) String() string {
	return fmt.Sprintf("[%s %s.%s] %s %s %fms", timeutils.ShortDate(tr.Timestamp), tr.TraceId, tr.Id,
		tr.Kind, tr.Name, tr.Duration.Seconds()*1000)
}

func StartServerTrace(w http.ResponseWriter, r *http.Request, localSpanName string, localServiceName string, srvTags map[string]string) *STrace {
	traceId := r.Header.Get(X_YUNION_TRACE_ID)
	var spanId string
	var spanName string
	var parentId string
	var peerSrvName string
	var debug bool
	var localAddr string
	var localPort int
	var isBorder bool
	if len(traceId) == 0 {
		isBorder = true
		traceId = utils.GenRequestId(4)
		spanId = "0"
		spanName = localSpanName
		parentId = ""
		peerSrvName = UNKNOWN_SERVICE_NAME
		debug = true
		localAddr = ""
		localPort = 0
	} else {
		remoteAddr := r.Header.Get(X_YUNION_REMOTE_ADDR)
		if len(remoteAddr) > 0 {
			addr, port := utils.GetAddrPort(remoteAddr)
			localAddr = addr
			localPort = port
		}
		spanId = r.Header.Get(X_YUNION_SPAN_ID)
		spanName = localSpanName
		if len(spanName) == 0 {
			spanName = r.Header.Get(X_YUNION_SPAN_NAME)
		}
		parentId = r.Header.Get(X_YUNION_PARENT_ID)
		peerSrvName = r.Header.Get(X_YUNION_PEER_SERVICE_NAME)
		if r.Header.Get(X_YUNION_TRACE_DEBUG) == "true" {
			debug = true
		} else {
			debug = false
		}
	}
	shared := false
	localEp := STraceEndpoint{ServiceName: localServiceName,
		Addr: localAddr, Port: localPort}
	remoteEp := STraceEndpoint{ServiceName: peerSrvName}
	if len(r.RemoteAddr) != 0 {
		addr, port := utils.GetAddrPort(r.RemoteAddr)
		remoteEp.Addr = addr
		remoteEp.Port = port
	}
	var tags map[string]string
	if srvTags != nil {
		tags = make(map[string]string)
		for k, v := range srvTags {
			tags[k] = v
		}
	}
	trace := STrace{TraceId: traceId,
		Name:           spanName,
		ParentId:       parentId,
		Id:             spanId,
		clientId:       0, // valid for ServerTrace
		Kind:           TRACE_KIND_SERVER,
		Timestamp:      time.Now(),
		Duration:       0,
		Debug:          debug,
		Shared:         shared,
		LocalEndpoint:  localEp,
		RemoteEndpoint: remoteEp,
		Tags:           tags,
	}
	if !isBorder {
		// do not distribute trace info across border
		w.Header().Set(X_YUNION_SPAN_NAME, spanName)
		w.Header().Set(X_YUNION_PEER_SERVICE_NAME, localServiceName)
		w.Header().Set(X_YUNION_REMOTE_ADDR, r.RemoteAddr)
		if tags != nil {
			for k, v := range tags {
				w.Header().Set(X_YUNION_TRACE_TAG+k, v)
			}
		}
	}
	return &trace
}

func (tr *STrace) EndTrace() {
	tr.Duration = time.Now().Sub(tr.Timestamp)
	SubmitTrace(tr)
}

func StartClientTrace(ctxTrace *STrace, remoteAddr string, remotePort int, localServiceName string) *STrace {
	var traceId string
	var parentId string
	var spanId string
	var debug bool
	var share bool
	if ctxTrace != nil {
		traceId = ctxTrace.TraceId
		parentId = ctxTrace.Id
		spanId = fmt.Sprintf("%s.%d", ctxTrace.Id, ctxTrace.clientId)
		ctxTrace.clientId += 1
		debug = ctxTrace.Debug
		share = ctxTrace.Shared
	} else {
		traceId = utils.GenRequestId(4)
		parentId = ""
		spanId = "0"
		debug = true
		share = false
	}
	localEp := STraceEndpoint{ServiceName: localServiceName,
		Addr: "", Port: 0}
	remoteEp := STraceEndpoint{ServiceName: "",
		Addr: remoteAddr, Port: remotePort}

	trace := STrace{TraceId: traceId,
		Name:           "",
		ParentId:       parentId,
		Id:             spanId,
		Kind:           TRACE_KIND_CLIENT,
		Timestamp:      time.Now(),
		Duration:       0,
		Debug:          debug,
		Shared:         share,
		LocalEndpoint:  localEp,
		RemoteEndpoint: remoteEp,
	}
	return &trace
}

func (tr *STrace) EndClientTraceHeader(header http.Header) {
	spanName := header.Get(X_YUNION_SPAN_NAME)
	srvName := header.Get(X_YUNION_PEER_SERVICE_NAME)
	localAddr := header.Get(X_YUNION_REMOTE_ADDR)
	tags := make(map[string]string)

	for k, v := range header {
		if strings.HasPrefix(k, X_YUNION_TRACE_TAG) {
			tagName := k[:len(X_YUNION_TRACE_TAG)]
			tags[tagName] = v[0]
		}
	}
	tr.EndClientTrace(spanName, srvName, localAddr, tags)
}

func (tr *STrace) EndClientTrace(spanName, remoteServiceName, localAddr string, tags map[string]string) {
	tr.Name = spanName
	tr.RemoteEndpoint.ServiceName = remoteServiceName
	var addr string
	var port int
	if len(localAddr) > 0 {
		addr, port = utils.GetAddrPort(localAddr)
	} else {
		addr = fmt.Sprintf("%s", utils.GetOutboundIP())
	}

	tr.LocalEndpoint.Addr = addr
	tr.LocalEndpoint.Port = port
	if tags != nil {
		if tr.Tags == nil {
			tr.Tags = make(map[string]string)
		}
		for k, v := range tags {
			tr.Tags[k] = v
		}
	}
	tr.EndTrace()
}

func (tr *STrace) AddClientRequestHeader(header http.Header) {
	header.Set(X_YUNION_TRACE_ID, tr.TraceId)
	header.Set(X_YUNION_REMOTE_ADDR, fmt.Sprintf("%s:%d", tr.RemoteEndpoint.Addr, tr.RemoteEndpoint.Port))
	header.Set(X_YUNION_SPAN_ID, tr.Id)
	header.Set(X_YUNION_SPAN_NAME, tr.Name)
	header.Set(X_YUNION_PARENT_ID, tr.ParentId)
	header.Set(X_YUNION_PEER_SERVICE_NAME, tr.LocalEndpoint.ServiceName)
	if tr.Debug {
		header.Set(X_YUNION_TRACE_DEBUG, "true")
	}
}

func SubmitTrace(trace *STrace) {
}
