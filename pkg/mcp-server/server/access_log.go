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

package server

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"
)

var accessLogHostId = genAccessLogHostId()

func genAccessLogHostId() string {
	hostname, _ := os.Hostname()
	h := sha256.New()
	fmt.Fprintf(h, "mcp-server:%s", hostname)
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

type accessLogResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *accessLogResponseWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *accessLogResponseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

func (w *accessLogResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *accessLogResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *accessLogResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("not a hijacker")
	}
	return h.Hijack()
}

func genAccessRequestId(w http.ResponseWriter, r *http.Request) string {
	rid := r.Header.Get("X-Request-Id")
	if len(rid) == 0 {
		rid = utils.GenRequestId(3)
	} else {
		rid = fmt.Sprintf("%s-%s", rid, utils.GenRequestId(3))
	}
	w.Header().Set("X-Request-Id", rid)
	w.Header().Set("X-Request-Host-Id", accessLogHostId)
	return rid
}

// withAccessLog wraps handler with appsrv-style access logs:
//
//	hostId status requestId METHOD /path (remote) durationMs
func withAccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lrw := &accessLogResponseWriter{ResponseWriter: w, status: 0}
		rid := genAccessRequestId(lrw, r)
		start := time.Now()
		next.ServeHTTP(lrw, r)
		if lrw.status == 0 {
			lrw.status = http.StatusOK
		}
		durationMs := float64(time.Since(start).Nanoseconds()) / 1e6
		remote := r.RemoteAddr
		if peer := r.Header.Get("X-Yunion-Peer-Service-Name"); peer != "" {
			remote = fmt.Sprintf("%s:%s", r.RemoteAddr, peer)
		}
		log.Infof("%s %d %s %s %s (%s) %.2fms",
			accessLogHostId, lrw.status, rid, r.Method, r.URL, remote, durationMs)
	})
}
