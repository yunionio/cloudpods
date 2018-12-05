package appsrv

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/trace"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/proxy"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type Application struct {
	name              string
	context           context.Context
	session           *SWorkerManager
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

	// record Http server for handle shotdown
	server *http.Server
}

const (
	DEFAULT_BACKLOG             = 1024
	DEFAULT_IDLE_TIMEOUT        = 10 * time.Second
	DEFAULT_READ_TIMEOUT        = 0
	DEFAULT_READ_HEADER_TIMEOUT = 10 * time.Second
	DEFAULT_WRITE_TIMEOUT       = 0
	DEFAULT_PROCESS_TIMEOUT     = 15 * time.Second
)

func NewApplication(name string, connMax int, db bool) *Application {
	app := Application{name: name,
		context:           context.Background(),
		connMax:           connMax,
		session:           NewWorkerManager("HttpRequestWorkerManager", connMax, DEFAULT_BACKLOG, db),
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

	// initialize random seed
	rand.Seed(time.Now().UnixNano())

	return &app
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

func (app *Application) AddReverseProxyHandler(prefix string, ef *proxy.SEndpointFactory) {
	handler := proxy.NewHTTPReverseProxy(ef).ServeHTTP
	for _, method := range []string{"GET", "HEAD", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"} {
		app.AddHandler(method, prefix, handler)
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
	// log.Printf("defaultHandler %s %s", r.Method, r.URL.Path)
	rid := genRequestId(w, r)
	lrw := &loggingResponseWriter{w, http.StatusOK}
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
	if !skipLog {
		log.Infof("%d %s %s %s (%s) %.2fms", lrw.status, rid, r.Method, r.URL, r.RemoteAddr, duration)
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

func (app *Application) defaultHandle(w http.ResponseWriter, r *http.Request, rid string) (*SHandlerInfo, *SAppParams) {
	segs := SplitPath(r.URL.Path)
	params := make(map[string]string)
	w.Header().Set("Server", "Yunion AppServer/Go/2018.4")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	isCors := app.handleCORS(w, r)
	handler := app.getRoot(r.Method).Match(segs, params)
	if handler != nil {
		// log.Print("Found handler", params)
		hand, ok := handler.(*SHandlerInfo)
		if ok {
			fw := newResponseWriterChannel(w)
			worker := make(chan *SWorker)
			to := hand.processTimeout
			if to == 0 {
				to = app.processTimeout
			}
			ctx, cancel := context.WithTimeout(app.context, to)
			defer cancel()
			session := hand.workerMan
			if session == nil {
				session = app.session
			}
			appParams := hand.GetAppParams(params, segs)
			appParams.Request = r
			appParams.Response = w
			session.Run(
				func() {
					if ctx.Err() == nil {
						ctx = context.WithValue(ctx, appctx.APP_CONTEXT_KEY_REQUEST_ID, rid)
						ctx = context.WithValue(ctx, appctx.APP_CONTEXT_KEY_CUR_ROOT, hand.path)
						ctx = context.WithValue(ctx, appctx.APP_CONTEXT_KEY_CUR_PATH, segs[len(hand.path):])
						ctx = context.WithValue(ctx, appctx.APP_CONTEXT_KEY_PARAMS, params)
						if hand.metadata != nil {
							ctx = context.WithValue(ctx, appctx.APP_CONTEXT_KEY_METADATA, hand.metadata)
						}
						ctx = context.WithValue(ctx, APP_CONTEXT_KEY_APP_PARAMS, appParams)
						func() {
							span := trace.StartServerTrace(&fw, r, appParams.Name, app.GetName(), hand.GetTags())
							defer func() {
								if !appParams.SkipTrace {
									span.EndTrace()
								}
							}()
							ctx = context.WithValue(ctx, appctx.APP_CONTEXT_KEY_TRACE, span)
							hand.handler(ctx, &fw, r)
						}()
					} // otherwise, the task has been timeout
					fw.closeChannels()
				},
				worker,
				func(err error) {
					httperrors.InternalServerError(&fw, "Internal server error: %s", err)
					fw.closeChannels()
				},
			)
			runErr := fw.wait(ctx, worker)
			if runErr != nil {
				switch runErr.(type) {
				case *httputils.JSONClientError:
					je := runErr.(*httputils.JSONClientError)
					httperrors.GeneralServerError(w, je)
				default:
					httperrors.InternalServerError(w, "Internal server error")
				}
			}
			fw.closeChannels()
			return hand, appParams
		} else {
			log.Errorf("Invalid handler for %s", r.URL)
			httperrors.InternalServerError(w, "Invalid handler %s", r.URL)
		}
	} else if !isCors {
		log.Errorf("Handler not found")
		httperrors.NotFoundError(w, "Handler not found")
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
	app.addDefaultHandlers()
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

func (app *Application) ListenAndServe(addr string) {
	app.server = app.initServer(addr)
	err := app.server.ListenAndServe()
	if err != nil {
		log.Fatalf("ListAndServer fail: %s", err)
	}
}

func (app *Application) ShowDown(ctx context.Context) error {
	if app.server != nil {
		return app.server.Shutdown(ctx)
	}
	return fmt.Errorf("Not init http server ??")
}

func (app *Application) ListenAndServeTLS(addr string, certFile, keyFile string) {
	s := app.initServer(addr)
	err := s.ListenAndServeTLS(certFile, keyFile)
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("ListAndServer fail: %s", err)
	}
}

func isJsonContentType(r *http.Request) bool {
	contType := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.HasPrefix(contType, "application/json") {
		return true
	}
	return false
}

func FetchEnv(ctx context.Context, w http.ResponseWriter, r *http.Request) (params map[string]string, query jsonutils.JSONObject, body jsonutils.JSONObject) {
	var err error
	params = appctx.AppContextParams(ctx)
	query, err = jsonutils.ParseQueryString(r.URL.RawQuery)
	if err != nil {
		log.Errorf("Parse query string %s failed: %s", r.URL.RawQuery, err)
	}
	//var body jsonutils.JSONObject = nil
	if (r.Method == "PUT" || r.Method == "POST" || r.Method == "DELETE" || r.Method == "PATCH") && r.ContentLength > 0 && isJsonContentType(r) {
		body, err = FetchJSON(r)
		if err != nil {
			log.Errorf("Fail to decode JSON request body: %s", err)
		}
	}
	return params, query, body
}

func (app *Application) GetContext() context.Context {
	return app.context
}
