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
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/trace"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/signalutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/proxy"
	"yunion.io/x/onecloud/pkg/util/ctx"
)

type Application struct {
	name              string
	context           context.Context
	session           *SWorkerManager
	readSession       *SWorkerManager
	systemSession     *SWorkerManager
	roots             map[string]*RadixNode
	rootLock          *sync.RWMutex
	connMax           int
	idleTimeout       time.Duration
	readTimeout       time.Duration
	readHeaderTimeout time.Duration
	writeTimeout      time.Duration
	processTimeout    time.Duration
	defHandlerInfo    SHandlerInfo
	cors              *Cors
	middlewares       []MiddlewareFunc
	hostId            string

	isExiting       bool
	idleConnsClosed chan struct{}
	httpServer      *http.Server
	slaveHttpServer *http.Server

	exception func(method, path string, body jsonutils.JSONObject, err error)

	isTLS bool
}

const (
	DEFAULT_BACKLOG             = 1024
	DEFAULT_IDLE_TIMEOUT        = 10 * time.Second
	DEFAULT_READ_TIMEOUT        = 0
	DEFAULT_READ_HEADER_TIMEOUT = 10 * time.Second
	DEFAULT_WRITE_TIMEOUT       = 0
	// set default process timeout to 60 seconds
	DEFAULT_PROCESS_TIMEOUT = 60 * time.Second
)

var quitHandlerRegisted bool

func NewApplication(name string, connMax int, db bool) *Application {
	app := Application{name: name,
		context:           ctx.CtxWithTime(),
		connMax:           connMax,
		session:           NewWorkerManager("HttpRequestWorkerManager", connMax, DEFAULT_BACKLOG, db),
		readSession:       NewWorkerManager("HttpGetRequestWorkerManager", connMax, DEFAULT_BACKLOG, db),
		systemSession:     NewWorkerManager("InternalHttpRequestWorkerManager", 1, DEFAULT_BACKLOG, false),
		roots:             make(map[string]*RadixNode),
		rootLock:          &sync.RWMutex{},
		idleTimeout:       DEFAULT_IDLE_TIMEOUT,
		readTimeout:       DEFAULT_READ_TIMEOUT,
		readHeaderTimeout: DEFAULT_READ_HEADER_TIMEOUT,
		writeTimeout:      DEFAULT_WRITE_TIMEOUT,
		processTimeout:    DEFAULT_PROCESS_TIMEOUT,
	}
	app.SetContext(appctx.APP_CONTEXT_KEY_APP, &app)
	app.SetContext(appctx.APP_CONTEXT_KEY_APPNAME, app.name)

	hm := sha1.New()
	hm.Write([]byte(name))
	hostname, _ := os.Hostname()
	hm.Write([]byte(hostname))
	outIp := utils.GetOutboundIP()
	hm.Write([]byte(outIp.String()))
	hostId := base64.URLEncoding.EncodeToString(hm.Sum(nil))

	log.Infof("App hostId: %s (%s,%s,%s)", hostId, name, hostname, outIp.String())
	app.hostId = hostId
	app.SetContext(appctx.APP_CONTEXT_KEY_HOST_ID, hostId)

	// initialize random seed
	rand.Seed(time.Now().UnixNano())

	return &app
}

func (self *Application) OnException(exception func(method, path string, body jsonutils.JSONObject, err error)) *Application {
	self.exception = exception
	return self
}

func SplitPath(path string) []string {
	ret := make([]string, 0)
	for _, seg := range strings.Split(path, "/") {
		seg = strings.Trim(seg, " \t\r\n")
		if len(seg) > 0 {
			ret = append(ret, seg)
		}
	}
	return ret
}

func (app *Application) GetName() string {
	return app.name
}

func (app *Application) getRoot(method string) *RadixNode {
	app.rootLock.RLock()
	if v, ok := app.roots[method]; ok {
		app.rootLock.RUnlock()
		return v
	}
	app.rootLock.RUnlock()

	v := NewRadix()
	app.rootLock.Lock()
	app.roots[method] = v
	app.rootLock.Unlock()
	return v
}

func (app *Application) AddReverseProxyHandler(prefix string, ef *proxy.SEndpointFactory, m proxy.RequestManipulator) {
	app.AddReverseProxyHandlerWithCallbackConfig(prefix, ef, m,
		func(method string, hi *SHandlerInfo) *SHandlerInfo {
			return hi
		},
	)
}

func (app *Application) AddReverseProxyHandlerWithCallbackConfig(prefix string, ef *proxy.SEndpointFactory, m proxy.RequestManipulator, confCb func(string, *SHandlerInfo) *SHandlerInfo) {
	handler := proxy.NewHTTPReverseProxy(ef, m).ServeHTTP
	for _, method := range []string{"GET", "HEAD", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"} {
		hi := &SHandlerInfo{}
		hi = confCb(method, hi)
		if hi != nil {
			hi.SetMethod(method)
			hi.SetPath(prefix)
			hi.SetHandler(handler)
			app.AddHandler3(hi)
		}
	}
}

func (app *Application) AddHandler(method string, prefix string,
	handler func(context.Context, http.ResponseWriter, *http.Request)) *SHandlerInfo {
	return app.AddHandler2(method, prefix, handler, nil, "", nil)
}

func (app *Application) AddHandler2(method string, prefix string,
	handler func(context.Context, http.ResponseWriter, *http.Request),
	metadata map[string]interface{}, name string, tags map[string]string) *SHandlerInfo {
	segs := SplitPath(prefix)
	hi := newHandlerInfo(method, segs, handler, metadata, name, tags)
	return app.AddHandler3(hi)
}

func (app *Application) AddHandler3(hi *SHandlerInfo) *SHandlerInfo {
	e := app.getRoot(hi.method).Add(hi.path, hi)
	if e != nil {
		log.Fatalf("Fail to register %s %s: %s", hi.method, hi.path, e)
	}
	return hi
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
	data   []byte
}

func (lrw *loggingResponseWriter) Write(data []byte) (int, error) {
	lrw.data = data
	return lrw.ResponseWriter.Write(data)
}

func (lrw *loggingResponseWriter) Hijack() (rwc net.Conn, buf *bufio.ReadWriter, err error) {
	if f, ok := lrw.ResponseWriter.(http.Hijacker); ok {
		return f.Hijack()
	}
	return nil, nil, fmt.Errorf("not a hijacker")
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	if code < 100 || code >= 600 {
		log.Errorf("Invalud status code %d, set code to 598", code)
		code = 598
	}
	lrw.status = code
	lrw.ResponseWriter.WriteHeader(code)
}

func genRequestId(w http.ResponseWriter, r *http.Request) string {
	rid := r.Header.Get("X-Request-Id")
	if len(rid) == 0 {
		rid = utils.GenRequestId(3)
	} else {
		rid = fmt.Sprintf("%s-%s", rid, utils.GenRequestId(3))
	}
	w.Header().Set("X-Request-Id", rid)
	return rid
}

func (app *Application) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rid := genRequestId(w, r)
	w.Header().Set("X-Request-Host-Id", app.hostId)
	lrw := &loggingResponseWriter{ResponseWriter: w, status: http.StatusOK, data: []byte{}}
	start := time.Now()
	hi, params := app.defaultHandle(lrw, r, rid)
	if hi == nil {
		hi = &app.defHandlerInfo
	}
	var counter *handlerRequestCounter
	if lrw.status < 400 {
		counter = &hi.counter2XX
	} else if lrw.status < 500 {
		counter = &hi.counter4XX
	} else {
		counter = &hi.counter5XX
	}
	duration := float64(time.Since(start).Nanoseconds()) / 1000000
	counter.hit += 1
	counter.duration += duration
	skipLog := false
	if params != nil {
		if params.SkipLog {
			skipLog = true
		}
	} else if hi.skipLog {
		skipLog = true
	}
	peerServiceName := r.Header.Get("X-Yunion-Peer-Service-Name")
	var remote string
	if len(peerServiceName) > 0 {
		remote = fmt.Sprintf("%s:%s", r.RemoteAddr, peerServiceName)
	} else {
		remote = r.RemoteAddr
	}
	if !skipLog {
		log.Infof("%s %d %s %s %s (%s) %.2fms", app.hostId, lrw.status, rid, r.Method, r.URL, remote, duration)
	}
	if lrw.status >= 500 && app.exception != nil {
		url := fmt.Sprintf("%d %s (%s) %.2fms", lrw.status, r.URL.String(), remote, duration)
		app.exception(r.Method, url, params.Body, errors.Errorf(string(lrw.data)))
	}
}

func (app *Application) handleCORS(w http.ResponseWriter, r *http.Request) bool {
	if app.cors == nil {
		return false
	}
	if r.Method == "OPTIONS" && r.Header.Get("Access-Control-Request-Method") != "" {
		app.cors.handlePreflight(w, r)
		return true
	} else {
		app.cors.handleActualRequest(w, r)
		return false
	}
}

type appTask struct {
	ctx       context.Context
	hand      *SHandlerInfo
	rid       string
	params    map[string]string
	appParams *SAppParams
	app       *Application
	fw        responseWriterChannel
	r         *http.Request
	segs      []string
	to        time.Duration
	cancel    context.CancelFunc
}

func (t *appTask) Run() {
	if t.ctx.Err() == nil {
		t.ctx = context.WithValue(t.ctx, appctx.APP_CONTEXT_KEY_REQUEST_ID, t.rid)
		t.ctx = context.WithValue(t.ctx, appctx.APP_CONTEXT_KEY_CUR_ROOT, t.hand.path)
		t.ctx = context.WithValue(t.ctx, appctx.APP_CONTEXT_KEY_CUR_PATH, t.segs[len(t.hand.path):])
		t.ctx = context.WithValue(t.ctx, appctx.APP_CONTEXT_KEY_PARAMS, t.params)
		t.ctx = context.WithValue(t.ctx, appctx.APP_CONTEXT_KEY_START_TIME, time.Now().UTC())
		if t.hand.metadata != nil {
			t.ctx = context.WithValue(t.ctx, appctx.APP_CONTEXT_KEY_METADATA, t.hand.metadata)
		}
		t.ctx = context.WithValue(t.ctx, APP_CONTEXT_KEY_APP_PARAMS, t.appParams)
		func() {
			span := trace.StartServerTrace(&t.fw, t.r, t.appParams.Name, t.app.GetName(), t.hand.GetTags())
			defer func() {
				if !t.appParams.SkipTrace {
					span.EndTrace()
				}
			}()
			t.ctx = context.WithValue(t.ctx, appctx.APP_CONTEXT_KEY_TRACE, span)
			t.hand.handler(t.ctx, &t.fw, t.r)
		}()
	} // otherwise, the task has been timeout
	t.fw.closeChannels()
}

func (t *appTask) Dump() string {
	return fmt.Sprintf("%s %s", t.r.Method, t.r.URL.String())
}

func (app *Application) defaultHandle(w http.ResponseWriter, r *http.Request, rid string) (*SHandlerInfo, *SAppParams) {
	segs := SplitPath(r.URL.EscapedPath())
	for i := range segs {
		if p, err := url.PathUnescape(segs[i]); err == nil {
			segs[i] = p
		}
	}
	params := make(map[string]string)
	w.Header().Set("Server", "Yunion AppServer/Go/2018.4")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	if app.isTLS {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	}
	isCors := app.handleCORS(w, r)
	handler := app.getRoot(r.Method).Match(segs, params)
	if handler != nil {
		// log.Print("Found handler", params)
		hand, ok := handler.(*SHandlerInfo)
		if ok {
			task := &appTask{
				ctx:    app.context,
				hand:   hand,
				rid:    rid,
				params: params,
				app:    app,
				fw:     newResponseWriterChannel(w),
				r:      r,
				segs:   segs,
				to:     hand.FetchProcessTimeout(r),
				cancel: nil,
			}

			currentWorker := make(chan *SWorker, 1) // make it a buffered channel
			if task.to == 0 {
				task.to = app.processTimeout
			}
			if task.to > 0 {
				task.ctx, task.cancel = context.WithTimeout(task.ctx, task.to)
			}
			if task.cancel != nil {
				defer task.cancel()
			}
			task.ctx = appctx.WithRequestLang(task.ctx, r)
			session := hand.workerMan
			if session == nil {
				if r.Method == "GET" || r.Method == "HEAD" {
					session = app.readSession
				} else {
					session = app.session
				}
			}
			task.appParams = hand.GetAppParams(params, segs)
			task.appParams.Request = r
			task.appParams.Response = w
			if r.Body != nil && r.ContentLength > 0 && getContentType(r) == ContentTypeJson {
				data, _ := ioutil.ReadAll(r.Body)
				task.appParams.Body, _ = jsonutils.Parse(data)
				r.Body = ioutil.NopCloser(bytes.NewBuffer(data))
			}
			session.Run(
				task,
				currentWorker,
				func(err error) {
					httperrors.InternalServerError(task.ctx, &task.fw, "Internal server error: %s", err)
					task.fw.closeChannels()
				},
			)
			runErr := task.fw.wait(task.ctx, currentWorker)
			if runErr != nil {
				switch runErr.(type) {
				case *httputils.JSONClientError:
					je := runErr.(*httputils.JSONClientError)
					httperrors.GeneralServerError(task.ctx, w, je)
				default:
					httperrors.InternalServerError(task.ctx, w, "Internal server error")
				}
			}
			task.fw.closeChannels()
			return hand, task.appParams
		} else {
			ctx := appctx.WithRequestLang(context.TODO(), r)
			httperrors.InternalServerError(ctx, w, "Invalid handler %s", r.URL)
		}
	} else if !isCors {
		ctx := appctx.WithRequestLang(context.TODO(), r)
		httperrors.NotFoundError(ctx, w, "Handler not found")
	}
	return nil, nil
}

func (app *Application) AddDefaultHandler(method string, prefix string, handler func(context.Context, http.ResponseWriter, *http.Request), name string) {
	segs := SplitPath(prefix)
	hi := newHandlerInfo(method, segs, handler, nil, name, nil)
	hi.SetSkipLog(true).SetWorkerManager(app.systemSession)
	app.AddHandler3(hi)
}

func (app *Application) addDefaultHandlers() {
	app.AddDefaultHandler("GET", "/version", VersionHandler, "version")
	app.AddDefaultHandler("GET", "/stats", StatisticHandler, "stats")
	app.AddDefaultHandler("POST", "/ping", PingHandler, "ping")
	app.AddDefaultHandler("GET", "/ping", PingHandler, "ping")
	app.AddDefaultHandler("GET", "/worker_stats", WorkerStatsHandler, "worker_stats")
	app.AddDefaultHandler("GET", "/process_stats", ProcessStatsHandler, "process_stats")
}

func timeoutHandle(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "upload") {
			// 上传文件接口默认不超时
			h.ServeHTTP(w, r)
		} else {
			// 服务器超时时间默认设置为10秒.
			http.TimeoutHandler(h, DEFAULT_IDLE_TIMEOUT, "").ServeHTTP(w, r)
		}
	}
}

func (app *Application) initServer(addr string) *http.Server {
	/* db := AppContextDB(app.context)
	if db != nil {
		db.SetMaxIdleConns(app.connMax + 1)
		db.SetMaxOpenConns(app.connMax + 1)
	}
	*/

	s := &http.Server{
		Addr:              addr,
		Handler:           app,
		IdleTimeout:       app.idleTimeout,
		ReadTimeout:       app.readTimeout,
		ReadHeaderTimeout: app.readHeaderTimeout,
		WriteTimeout:      app.writeTimeout,
		MaxHeaderBytes:    1 << 20,
	}
	return s
}

func (app *Application) registerCleanShutdown(s *http.Server, onStop func()) {
	if quitHandlerRegisted {
		log.Warningf("Application quit handler registed, duplicated!!!")
		return
	} else {
		quitHandlerRegisted = true
	}
	app.idleConnsClosed = make(chan struct{})

	signalutils.SetDumpStackSignal()

	quitSignals := []os.Signal{syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM}
	signalutils.RegisterSignal(func() {
		if app.isExiting {
			log.Infof("Quit signal received!!! clean up in progress, be patient...")
			return
		}
		app.isExiting = true
		log.Infof("Quit signal received!!! do cleanup...")

		if err := s.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Errorf("HTTP server Shutdown: %v", err)
		}
		if onStop != nil {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Errorf("app exiting error: %s", r)
					}
				}()
				onStop()
			}()
		}
		close(app.idleConnsClosed)
	}, quitSignals...)

	signalutils.StartTrap()
}

func (app *Application) waitCleanShutdown() {
	<-app.idleConnsClosed
	log.Infof("Service stopped.")
}

func (app *Application) ListenAndServe(addr string) {
	app.ListenAndServeWithCleanup(addr, nil)
}

func (app *Application) ListenAndServeTLS(addr string, certFile, keyFile string) {
	app.ListenAndServeTLSWithCleanup(addr, certFile, keyFile, nil)
}

func (app *Application) ListenAndServeWithCleanup(addr string, onStop func()) {
	app.ListenAndServeTLSWithCleanup(addr, "", "", onStop)
}

func (app *Application) ListenAndServeTLSWithCleanup(addr string, certFile, keyFile string, onStop func()) {
	app.ListenAndServeTLSWithCleanup2(addr, certFile, keyFile, onStop, true)
}

func (app *Application) ListenAndServeWithoutCleanup(addr, certFile, keyFile string) {
	app.ListenAndServeTLSWithCleanup2(addr, certFile, keyFile, nil, false)
}

func (app *Application) ListenAndServeTLSWithCleanup2(addr string, certFile, keyFile string, onStop func(), isMaster bool) {
	app.isTLS = true
	httpSrv := app.initServer(addr)
	if isMaster {
		app.addDefaultHandlers()
		AddPProfHandler("", app)
		app.httpServer = httpSrv
		app.registerCleanShutdown(app.httpServer, onStop)
	} else {
		app.slaveHttpServer = httpSrv
	}
	app.listenAndServeInternal(httpSrv, certFile, keyFile)
	if isMaster {
		app.waitCleanShutdown()
	}
}

func (app *Application) Stop(ctx context.Context) error {
	if app.httpServer != nil {
		return app.httpServer.Shutdown(ctx)
	}
	return nil
}

func (app *Application) listenAndServeInternal(s *http.Server, certFile, keyFile string) {
	var err error
	if len(certFile) == 0 && len(keyFile) == 0 {
		err = s.ListenAndServe()
	} else {
		err = s.ListenAndServeTLS(certFile, keyFile)
	}
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("ListAndServer fail: %s (cert=%s key=%s)", err, certFile, keyFile)
	}
}

func isJsonContentType(r *http.Request) bool {
	contType := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.HasPrefix(contType, "application/json") {
		return true
	}
	return false
}

func isFormContentType(r *http.Request) bool {
	contType := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.HasPrefix(contType, "application/json") {
		return true
	}
	return false
}

type TContentType string

const (
	ContentTypeJson    = TContentType("Json")
	ContentTypeForm    = TContentType("Form")
	ContentTypeUnknown = TContentType("Unknown")
)

func getContentType(r *http.Request) TContentType {
	contType := func() string {
		for _, k := range []string{"Content-Type", "content-type"} {
			contentType := r.Header.Get(k)
			if len(contentType) > 0 {
				return strings.ToLower(contentType)
			}
		}
		return ""
	}()
	for k, v := range map[string]TContentType{
		"application/json":                  ContentTypeJson,
		"application/x-www-form-urlencoded": ContentTypeForm,
	} {
		if strings.HasPrefix(contType, k) {
			return v
		}
	}
	return ContentTypeUnknown
}

func FetchEnv(ctx context.Context, w http.ResponseWriter, r *http.Request) (params map[string]string, query jsonutils.JSONObject, body jsonutils.JSONObject) {
	var err error
	params = appctx.AppContextParams(ctx)
	query, err = jsonutils.ParseQueryString(r.URL.RawQuery)
	if err != nil {
		log.Errorf("Parse query string %s failed: %v", r.URL.RawQuery, err)
	}
	//var body jsonutils.JSONObject = nil
	if r.Method == "PUT" || r.Method == "POST" || r.Method == "DELETE" || r.Method == "PATCH" {
		switch getContentType(r) {
		case ContentTypeJson:
			if r.ContentLength > 0 {
				body, err = FetchJSON(r)
				if err != nil {
					log.Warningf("Fail to decode JSON request body: %v", err)
				}
			}
		case ContentTypeForm:
			err := r.ParseForm()
			if err != nil {
				log.Warningf("ParseForm %s error: %v", r.URL.String(), err)
			}
			query, err = jsonutils.ParseQueryString(r.PostForm.Encode())
			if err != nil {
				log.Warningf("Parse query string %s failed: %v", r.PostForm.Encode(), err)
			}
		default:
			log.Warningf("%s invalid contentType with header %v", r.URL.String(), r.Header)
		}
	}
	return params, query, body
}

func (app *Application) GetContext() context.Context {
	return app.context
}
