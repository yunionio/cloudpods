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
	rootLock          *sync.Mutex
	connMax           int
	idleTimeout       time.Duration
	readTimeout       time.Duration
	readHeaderTimeout time.Duration
	writeTimeout      time.Duration
	processTimeout    time.Duration
	defHandlerInfo    SHandlerInfo
	cors              *Cors
	middlewares       []MiddlewareFunc
}

const (
	DEFAULT_BACKLOG             = 256
	DEFAULT_IDLE_TIMEOUT        = 10 * time.Second
	DEFAULT_READ_TIMEOUT        = 0
	DEFAULT_READ_HEADER_TIMEOUT = 10 * time.Second
	DEFAULT_WRITE_TIMEOUT       = 0
	DEFAULT_PROCESS_TIMEOUT     = 15 * time.Second
)

func NewApplication(name string, connMax int) *Application {
	app := Application{name: name,
		context:           context.Background(),
		connMax:           connMax,
		session:           NewWorkerManager("HttpRequestWorkerManager", connMax, DEFAULT_BACKLOG),
		systemSession:     NewWorkerManager("InternalHttpRequestWorkerManager", 1, DEFAULT_BACKLOG),
		roots:             make(map[string]*RadixNode),
		rootLock:          &sync.Mutex{},
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

func (app *Application) getRootLocked(method string) *RadixNode {
	app.rootLock.Lock()
	defer app.rootLock.Unlock()
	if _, ok := app.roots[method]; !ok {
		app.roots[method] = NewRadix()
	}
	return app.roots[method]
}

func (app *Application) getRoot(method string) *RadixNode {
	if v, ok := app.roots[method]; ok {
		return v
	} else {
		return app.getRootLocked(method)
	}
}

func (app *Application) AddReverseProxyHandler(prefix string, ef *proxy.SEndpointFactory) {
	handler := proxy.NewHTTPReverseProxy(ef).ServeHTTP
	for _, method := range []string{"GET", "HEAD", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"} {
		app.AddHandler(method, prefix, handler)
	}
}

func (app *Application) AddHandler(method string, prefix string, handler func(context.Context, http.ResponseWriter, *http.Request)) {
	app.AddHandler2(method, prefix, handler, nil, "", nil)
}

func (app *Application) AddHandler2(method string, prefix string, handler func(context.Context, http.ResponseWriter, *http.Request), metadata map[string]interface{}, name string, tags map[string]string) {
	log.Debugf("%s - %s", method, prefix)
	segs := SplitPath(prefix)
	hi := newHandlerInfo(method, segs, handler, metadata, name, tags)
	app.AddHandler3(hi)
}

func (app *Application) AddHandler3(hi *SHandlerInfo) {
	e := app.getRoot(hi.method).Add(hi.path, hi)
	if e != nil {
		log.Fatalf("Fail to register %s %s: %s", hi.method, hi.path, e)
	}
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
	hi := app.defaultHandle(lrw, r, rid)
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
	if !hi.skipLog {
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

func (app *Application) defaultHandle(w http.ResponseWriter, r *http.Request, rid string) *SHandlerInfo {
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
						func() {
							span := trace.StartServerTrace(&fw, r, hand.GetName(params), app.GetName(), hand.GetTags())
							defer span.EndTrace()
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
			return hand
		} else {
			log.Errorf("Invalid handler for %s", r.URL)
			httperrors.InternalServerError(w, "Invalid handler %s", r.URL)
		}
	} else if !isCors {
		log.Errorf("Handler not found")
		httperrors.NotFoundError(w, "Handler not found")
	}
	return nil
}

func (app *Application) addDefaultHandler(method string, prefix string, handler func(context.Context, http.ResponseWriter, *http.Request), name string) {
	segs := SplitPath(prefix)
	hi := newHandlerInfo(method, segs, handler, nil, name, nil)
	hi.SetSkipLog(true).SetWorkerManager(app.systemSession)
	app.AddHandler3(hi)
}

func (app *Application) addDefaultHandlers() {
	app.addDefaultHandler("GET", "/version", VersionHandler, "version")
	app.addDefaultHandler("GET", "/stats", StatisticHandler, "stats")
	app.addDefaultHandler("POST", "/ping", PingHandler, "ping")
	app.addDefaultHandler("GET", "/ping", PingHandler, "ping")
	app.addDefaultHandler("GET", "/worker_stats", WorkerStatsHandler, "worker_stats")
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
	db := AppContextDB(app.context)
	if db != nil {
		db.SetMaxIdleConns(app.connMax + 1)
		db.SetMaxOpenConns(app.connMax + 1)
	}
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
	s := app.initServer(addr)
	err := s.ListenAndServe()
	if err != nil {
		log.Fatalf("ListAndServer fail: %s", err)
	}
}

func (app *Application) ListenAndServeTLS(addr string, certFile, keyFile string) {
	s := app.initServer(addr)
	err := s.ListenAndServeTLS(certFile, keyFile)
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("ListAndServer fail: %s", err)
	}
}

func FetchEnv(ctx context.Context, w http.ResponseWriter, r *http.Request) (map[string]string, jsonutils.JSONObject, jsonutils.JSONObject) {
	params := appctx.AppContextParams(ctx)
	query, e := jsonutils.ParseQueryString(r.URL.RawQuery)
	if e != nil {
		log.Errorf("Parse query string %s failed: %s", r.URL.RawQuery, e)
	}
	var body jsonutils.JSONObject = nil
	if r.Method == "PUT" || r.Method == "POST" || r.Method == "DELETE" || r.Method == "PATCH" {
		body, e = FetchJSON(r)
		if e != nil {
			log.Errorf("Fail to decode JSON request body: %s", e)
		}
	}
	return params, query, body
}
