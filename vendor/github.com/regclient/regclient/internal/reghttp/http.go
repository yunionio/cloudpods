// Package reghttp is used for HTTP requests to a registry
package reghttp

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	// crypto libraries included for go-digest
	_ "crypto/sha256"
	_ "crypto/sha512"

	"github.com/opencontainers/go-digest"
	"github.com/regclient/regclient/config"
	"github.com/regclient/regclient/internal/auth"
	"github.com/regclient/regclient/types"
	"github.com/regclient/regclient/types/warning"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
)

var defaultDelayInit, _ = time.ParseDuration("1s")
var defaultDelayMax, _ = time.ParseDuration("30s")
var warnRegexp = regexp.MustCompile(`^299\s+-\s+"([^"]+)"`)

const (
	DefaultRetryLimit = 3
)

// Client is an HTTP client wrapper
// It handles features like authentication, retries, backoff delays, TLS settings
type Client struct {
	host       map[string]*clientHost
	httpClient *http.Client
	rootCAPool [][]byte
	rootCADirs []string
	retryLimit int
	delayInit  time.Duration
	delayMax   time.Duration
	log        *logrus.Logger
	userAgent  string
	mu         sync.Mutex
}

type clientHost struct {
	initialized  bool
	backoffCur   int
	backoffUntil time.Time
	config       *config.Host
	httpClient   *http.Client
	auth         map[string]auth.Auth
	newAuth      func() auth.Auth
	mu           sync.Mutex
	parallel     *semaphore.Weighted
	throttle     *time.Ticker
}

// Req is a request to send to a registry
type Req struct {
	Host      string
	NoMirrors bool
	APIs      map[string]ReqAPI // allow different types of registries (registry/2.0, OCI, default to empty string)
}

// ReqAPI handles API specific settings in a request
type ReqAPI struct {
	Method     string
	DirectURL  *url.URL
	NoPrefix   bool
	Repository string
	Path       string
	Query      url.Values
	BodyLen    int64
	BodyBytes  []byte
	BodyFunc   func() (io.ReadCloser, error)
	Headers    http.Header
	Digest     digest.Digest
	IgnoreErr  bool
}

// Resp is used to handle the result of a request
type Resp interface {
	io.ReadSeekCloser
	HTTPResponse() *http.Response
}

type clientResp struct {
	ctx              context.Context
	client           *Client
	req              *Req
	resp             *http.Response
	mirror           string
	done             bool
	digest           digest.Digest
	digester         digest.Digester
	reader           io.Reader
	readCur, readMax int64
	parallel         *semaphore.Weighted
}

// Opts is used to configure client options
type Opts func(*Client)

// NewClient returns a client for handling requests
func NewClient(opts ...Opts) *Client {
	c := Client{
		httpClient: &http.Client{},
		host:       map[string]*clientHost{},
		retryLimit: DefaultRetryLimit,
		delayInit:  defaultDelayInit,
		delayMax:   defaultDelayMax,
		log:        &logrus.Logger{Out: io.Discard},
		rootCAPool: [][]byte{},
		rootCADirs: []string{},
	}
	for _, opt := range opts {
		opt(&c)
	}
	return &c
}

// WithCerts adds certificates
func WithCerts(certs [][]byte) Opts {
	return func(c *Client) {
		c.rootCAPool = append(c.rootCAPool, certs...)
	}
}

// WithCertDirs adds directories to check for host specific certs
func WithCertDirs(dirs []string) Opts {
	return func(c *Client) {
		c.rootCADirs = append(c.rootCADirs, dirs...)
	}
}

// WithCertFiles adds certificates by filename
func WithCertFiles(files []string) Opts {
	return func(c *Client) {
		for _, f := range files {
			cert, err := os.ReadFile(f)
			if err != nil {
				c.log.WithFields(logrus.Fields{
					"err":  err,
					"file": f,
				}).Warn("Failed to read certificate")
			} else {
				c.rootCAPool = append(c.rootCAPool, cert)
			}
		}
	}
}

// WithConfigHosts adds a list of config.Host entries to use for connection settings
func WithConfigHosts(ch []*config.Host) Opts {
	return func(c *Client) {
		for _, cur := range ch {
			if cur.Name == "" {
				continue
			}
			if _, ok := c.host[cur.Name]; !ok {
				c.host[cur.Name] = &clientHost{}
			}
			c.host[cur.Name].config = cur
		}
	}
}

// WithDelay initial time to wait between retries (increased with exponential backoff)
func WithDelay(delayInit time.Duration, delayMax time.Duration) Opts {
	return func(c *Client) {
		if delayInit > 0 {
			c.delayInit = delayInit
		}
		// delayMax must be at least delayInit, if 0 initialize to 30x delayInit
		if delayMax > c.delayInit {
			c.delayMax = delayMax
		} else if delayMax > 0 {
			c.delayMax = c.delayInit
		} else {
			c.delayMax = c.delayInit * 30
		}
	}
}

// WithHTTPClient uses a specific http client with retryable requests
func WithHTTPClient(hc *http.Client) Opts {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithRetryLimit restricts the number of retries (defaults to 5)
func WithRetryLimit(rl int) Opts {
	return func(c *Client) {
		if rl > 0 {
			c.retryLimit = rl
		}
	}
}

// WithLog injects a logrus Logger configuration
func WithLog(log *logrus.Logger) Opts {
	return func(c *Client) {
		c.log = log
	}
}

// WithTransport uses a specific http transport with retryable requests
func WithTransport(t *http.Transport) Opts {
	return func(c *Client) {
		c.httpClient = &http.Client{Transport: t}
	}
}

// WithUserAgent sets a user agent header
func WithUserAgent(ua string) Opts {
	return func(c *Client) {
		c.userAgent = ua
	}
}

// Do runs a request, returning the response result
func (c *Client) Do(ctx context.Context, req *Req) (Resp, error) {
	resp := &clientResp{
		ctx:      ctx,
		client:   c,
		req:      req,
		digester: digest.Canonical.Digester(),
	}
	err := resp.Next()
	return resp, err
}

// Next sends requests until a mirror responds or all requests fail
func (resp *clientResp) Next() error {
	var err error
	c := resp.client
	req := resp.req
	// lookup reqHost entry
	reqHost := c.getHost(req.Host)
	// create sorted list of mirrors, based on backoffs, upstream, and priority
	hosts := make([]*clientHost, 0, 1+len(reqHost.config.Mirrors))
	if !req.NoMirrors {
		for _, m := range reqHost.config.Mirrors {
			hosts = append(hosts, c.getHost(m))
		}
	}
	hosts = append(hosts, reqHost)
	sort.Slice(hosts, sortHostsCmp(hosts, reqHost.config.Name))
	// loop over requests to mirrors and retries
	curHost := 0
	for {
		backoff := false
		dropHost := false
		retryHost := false
		if len(hosts) == 0 {
			if err != nil {
				return err
			}
			return types.ErrAllRequestsFailed
		}
		if curHost >= len(hosts) {
			curHost = 0
		}
		h := hosts[curHost]
		resp.mirror = h.config.Name

		// check that context isn't canceled/done
		ctxErr := resp.ctx.Err()
		if ctxErr != nil {
			return ctxErr
		}

		api, okAPI := req.APIs[h.config.API]
		if !okAPI {
			api, okAPI = req.APIs[""]
		}

		// try each host in a closure to handle all the backoff/dropHost from one place
		if h.parallel != nil {
			h.parallel.Acquire(resp.ctx, 1)
		}
		err = func() error {
			var err error
			if !okAPI {
				dropHost = true
				return fmt.Errorf("failed looking up api \"%s\" for host \"%s\": %w", h.config.API, h.config.Name, types.ErrAPINotFound)
			}
			if api.Method == "HEAD" && h.config.APIOpts != nil {
				var disableHead bool
				disableHead, err = strconv.ParseBool(h.config.APIOpts["disableHead"])
				if err == nil && disableHead {
					dropHost = true
					return fmt.Errorf("head requests disabled for host \"%s\": %w", h.config.Name, types.ErrUnsupportedAPI)
				}
			}

			// store the desired digest
			resp.digest = api.Digest

			// build the url
			var u url.URL
			if api.DirectURL != nil {
				u = *api.DirectURL
			} else {
				u = url.URL{
					Host:   h.config.Hostname,
					Scheme: "https",
				}
				path := strings.Builder{}
				path.WriteString("/v2")
				if h.config.PathPrefix != "" && !api.NoPrefix {
					path.WriteString("/" + h.config.PathPrefix)
				}
				if api.Repository != "" {
					path.WriteString("/" + api.Repository)
				}
				path.WriteString("/" + api.Path)
				u.Path = path.String()
				if h.config.TLS == config.TLSDisabled {
					u.Scheme = "http"
				}
				if api.Query != nil {
					u.RawQuery = api.Query.Encode()
				}
			}
			// close previous response
			if resp.resp != nil && resp.resp.Body != nil {
				resp.resp.Body.Close()
			}
			// delay for backoff if needed
			if !h.backoffUntil.IsZero() && h.backoffUntil.After(time.Now()) {
				sleepTime := time.Until(h.backoffUntil)
				c.log.WithFields(logrus.Fields{
					"Host":    h.config.Name,
					"Seconds": sleepTime.Seconds(),
				}).Warn("Sleeping for backoff")
				select {
				case <-resp.ctx.Done():
					return types.ErrCanceled
				case <-time.After(sleepTime):
				}
			}
			var httpReq *http.Request
			httpReq, err = http.NewRequestWithContext(resp.ctx, api.Method, u.String(), nil)
			if err != nil {
				dropHost = true
				return err
			}
			if api.BodyFunc != nil {
				body, err := api.BodyFunc()
				if err != nil {
					dropHost = true
					return err
				}
				httpReq.Body = body
				httpReq.GetBody = api.BodyFunc
				httpReq.ContentLength = api.BodyLen
			} else if len(api.BodyBytes) > 0 {
				body := io.NopCloser(bytes.NewReader(api.BodyBytes))
				httpReq.Body = body
				httpReq.GetBody = func() (io.ReadCloser, error) { return body, nil }
				httpReq.ContentLength = api.BodyLen
			}
			if len(api.Headers) > 0 {
				httpReq.Header = api.Headers.Clone()
			}
			if c.userAgent != "" && httpReq.Header.Get("User-Agent") == "" {
				httpReq.Header.Add("User-Agent", c.userAgent)
			}
			if resp.readCur > 0 && resp.readMax > 0 {
				if httpReq.Header.Get("Range") == "" {
					httpReq.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", resp.readCur, resp.readMax))
				} else {
					dropHost = true
					return fmt.Errorf("unable to resume a connection within a range request")
				}
			}

			hAuth := h.getAuth(api.Repository)
			if hAuth != nil {
				// include docker generated scope to emulate docker clients
				if api.Repository != "" {
					scope := "repository:" + api.Repository + ":pull"
					if api.Method != "HEAD" && api.Method != "GET" {
						scope = scope + ",push"
					}
					hAuth.AddScope(h.config.Hostname, scope)
				}
				// add auth headers
				err = hAuth.UpdateRequest(httpReq)
				if err != nil {
					backoff = true
					return err
				}
			}

			if h.throttle != nil {
				<-h.throttle.C
			}

			// update http client for insecure requests and root certs
			httpClient := *h.httpClient

			// send request
			resp.client.log.WithFields(logrus.Fields{
				"url":      httpReq.URL.String(),
				"method":   httpReq.Method,
				"withAuth": (len(httpReq.Header.Values("Authorization")) > 0),
			}).Debug("http req")
			resp.resp, err = httpClient.Do(httpReq)

			if err != nil {
				c.log.WithFields(logrus.Fields{
					"URL": u.String(),
					"err": err,
				}).Debug("Request failed")
				backoff = true
				return err
			}
			// extract any warnings
			for _, wh := range resp.resp.Header.Values("Warning") {
				if match := warnRegexp.FindStringSubmatch(wh); len(match) == 2 {
					// TODO: pass other fields (registry hostname) with structured logging
					warning.Handle(resp.ctx, resp.client.log, match[1])
				}
			}
			statusCode := resp.resp.StatusCode
			if statusCode < 200 || statusCode >= 300 {
				switch statusCode {
				case http.StatusUnauthorized:
					// if auth can be done, retry same host without delay, otherwise drop/backoff
					if hAuth != nil {
						err = hAuth.HandleResponse(resp.resp)
					} else {
						err = fmt.Errorf("authentication handler unavailable")
					}
					if err != nil {
						if errors.Is(err, types.ErrEmptyChallenge) || errors.Is(err, types.ErrNoNewChallenge) || errors.Is(err, types.ErrHTTPUnauthorized) {
							c.log.WithFields(logrus.Fields{
								"URL": u.String(),
								"Err": err,
							}).Debug("Failed to handle auth request")
						} else {
							c.log.WithFields(logrus.Fields{
								"URL": u.String(),
								"Err": err,
							}).Warn("Failed to handle auth request")
						}
						backoff = true
						dropHost = true
					} else {
						err = fmt.Errorf("authentication required")
						retryHost = true
					}
					return err
				case http.StatusNotFound:
					// if not found, drop mirror for this req, but other requests don't need backoff
					dropHost = true
				case http.StatusRequestedRangeNotSatisfiable:
					// if range request error (blob push), drop mirror for this req, but other requests don't need backoff
					dropHost = true
				case http.StatusTooManyRequests, http.StatusRequestTimeout, http.StatusGatewayTimeout:
					// server is likely overloaded, backoff but still retry
					backoff = true
				default:
					// all other errors indicate a bigger issue, don't retry and set backoff
					backoff = true
					dropHost = true
				}
				c.log.WithFields(logrus.Fields{
					"URL":    u.String(),
					"Status": http.StatusText(statusCode),
				}).Debug("Request failed")
				errHTTP := HTTPError(resp.resp.StatusCode)
				errBody, _ := io.ReadAll(resp.resp.Body)
				resp.resp.Body.Close()
				return fmt.Errorf("request failed: %w: %s", errHTTP, errBody)
			}

			// update digester
			resp.reader = io.TeeReader(resp.resp.Body, resp.digester.Hash())
			resp.done = false
			// set variables from headers if found
			if resp.readCur == 0 && resp.readMax == 0 && resp.resp.Header.Get("Content-Length") != "" {
				cl, parseErr := strconv.ParseInt(resp.resp.Header.Get("Content-Length"), 10, 64)
				if parseErr == nil {
					resp.readMax = cl
				}
			}
			// verify Content-Range header when range request used, fail if missing
			if httpReq.Header.Get("Range") != "" && resp.resp.Header.Get("Content-Range") == "" {
				dropHost = true
				resp.resp.Body.Close()
				return fmt.Errorf("range request not supported by server")
			}
			return nil
		}()
		// return on success
		if err == nil {
			resp.parallel = h.parallel
			resp.backoffClear()
			return nil
		}
		// backoff, dropHost, and/or go to next host in the list
		if h.parallel != nil {
			h.parallel.Release(1)
		}
		if backoff {
			if api.IgnoreErr {
				// don't set a backoff, immediately drop the host when errors ignored
				dropHost = true
			} else {
				boErr := resp.backoffSet()
				if boErr != nil {
					// reached backoff limit
					dropHost = true
				}
			}
		}
		if dropHost {
			hosts = append(hosts[:curHost], hosts[curHost+1:]...)
		} else if !retryHost {
			curHost++
		}
	}
}

func (resp *clientResp) HTTPResponse() *http.Response {
	return resp.resp
}

func (resp *clientResp) Read(b []byte) (int, error) {
	if resp.done {
		return 0, io.EOF
	}
	if resp.resp == nil {
		return 0, types.ErrNotFound
	}
	// perform the read
	i, err := resp.reader.Read(b)
	resp.readCur += int64(i)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		if resp.resp.Request.Method == "HEAD" || resp.readCur >= resp.readMax {
			resp.done = true
		} else {
			// short read, retry?
			resp.client.log.WithFields(logrus.Fields{
				"curRead":    resp.readCur,
				"contentLen": resp.readMax,
			}).Debug("EOF before reading all content, retrying")
			// retry
			resp.backoffSet()
			respErr := resp.Next()
			// unrecoverable EOF
			if respErr != nil {
				resp.client.log.WithFields(logrus.Fields{
					"err": respErr,
				}).Warn("Failed to recover from short read")
				resp.done = true
				return i, err
			}
			// retry successful, no EOF
			return i, nil
		}
		// validate the digest if specified
		if resp.resp.Request.Method != "HEAD" && resp.digest != "" && resp.digest != resp.digester.Digest() {
			resp.client.log.WithFields(logrus.Fields{
				"expected": resp.digest,
				"computed": resp.digester.Digest(),
			}).Warn("Digest mismatch")
			resp.done = true
			return i, fmt.Errorf("%w, expected %s, computed %s", types.ErrDigestMismatch,
				resp.digest.String(), resp.digester.Digest().String())
		}
	}

	if err == nil {
		return i, nil
	}
	return i, err
}

func (resp *clientResp) Close() error {
	if resp.parallel != nil {
		resp.parallel.Release(1)
		resp.parallel = nil
	}
	if resp.resp == nil {
		return types.ErrNotFound
	}
	resp.done = true
	return resp.resp.Body.Close()
}

func (resp *clientResp) Seek(offset int64, whence int) (int64, error) {
	newOffset := resp.readCur
	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset += offset
	case io.SeekEnd:
		if resp.readMax <= 0 {
			return resp.readCur, fmt.Errorf("seek from end is not supported")
		} else if resp.readMax+offset < 0 {
			return resp.readCur, fmt.Errorf("seek past beginning of the file is not supported")
		}
		newOffset = resp.readMax + offset
	default:
		return resp.readCur, fmt.Errorf("unknown value of whence: %d", whence)
	}
	if newOffset == 0 {
		// reset digester
		resp.digester = digest.Canonical.Digester()
		resp.readCur = 0
		// rerun the request to restart
		err := resp.Next()
		if err != nil {
			return resp.readCur, err
		}
	} else if newOffset != resp.readCur {
		return resp.readCur, fmt.Errorf("seek to arbitrary position is not supported")
	}
	return resp.readCur, nil
}

func (resp *clientResp) backoffClear() {
	c := resp.client
	c.mu.Lock()
	defer c.mu.Unlock()
	ch := c.host[resp.mirror]
	if ch.backoffCur > c.retryLimit {
		ch.backoffCur = c.retryLimit
	}
	if ch.backoffCur > 0 {
		ch.backoffCur--
		if ch.backoffCur == 0 {
			ch.backoffUntil = time.Time{}
		}
	}
}

func (resp *clientResp) backoffSet() error {
	c := resp.client
	c.mu.Lock()
	defer c.mu.Unlock()
	ch := c.host[resp.mirror]
	ch.backoffCur++
	// sleep for backoff time
	sleepTime := c.delayInit << ch.backoffCur
	// limit to max delay
	if sleepTime > c.delayMax {
		sleepTime = c.delayMax
	}
	// check rate limit header
	if resp.resp != nil && resp.resp.Header.Get("Retry-After") != "" {
		ras := resp.resp.Header.Get("Retry-After")
		ra, _ := time.ParseDuration(ras + "s")
		if ra > c.delayMax {
			sleepTime = c.delayMax
		} else if ra > sleepTime {
			sleepTime = ra
		}
	}

	ch.backoffUntil = time.Now().Add(sleepTime)

	if ch.backoffCur >= c.retryLimit {
		return fmt.Errorf("%w: backoffs %d", types.ErrBackoffLimit, ch.backoffCur)
	}

	return nil
}

func (c *Client) getHost(host string) *clientHost {
	c.mu.Lock()
	defer c.mu.Unlock()
	h, ok := c.host[host]
	if ok && h.initialized {
		return h
	}
	if !ok {
		h = &clientHost{}
	}
	if h.config == nil {
		h.config = config.HostNewName(host)
	}
	if h.auth == nil {
		h.auth = map[string]auth.Auth{}
	}
	if h.throttle == nil && h.config.ReqPerSec > 0 {
		h.throttle = time.NewTicker(time.Duration(float64(time.Second) / h.config.ReqPerSec))
	}
	if h.parallel == nil && h.config.ReqConcurrent > 0 {
		h.parallel = semaphore.NewWeighted(h.config.ReqConcurrent)
	}

	if h.httpClient == nil {
		h.httpClient = c.httpClient
		// update http client for insecure requests and root certs
		if h.config.TLS == config.TLSInsecure || len(c.rootCAPool) > 0 || len(c.rootCADirs) > 0 || h.config.RegCert != "" {
			// create a new client and modify the transport
			httpClient := *c.httpClient
			if httpClient.Transport == nil {
				httpClient.Transport = http.DefaultTransport.(*http.Transport).Clone()
			}
			t, ok := httpClient.Transport.(*http.Transport)
			if ok {
				var tlsc *tls.Config
				if t.TLSClientConfig != nil {
					tlsc = t.TLSClientConfig.Clone()
				} else {
					tlsc = &tls.Config{}
				}
				if h.config.TLS == config.TLSInsecure {
					tlsc.InsecureSkipVerify = true
				} else {
					rootPool, err := makeRootPool(c.rootCAPool, c.rootCADirs, h.config.Hostname, h.config.RegCert)
					if err != nil {
						c.log.WithFields(logrus.Fields{
							"err": err,
						}).Warn("failed to setup CA pool")
					} else {
						tlsc.RootCAs = rootPool
					}
				}
				t.TLSClientConfig = tlsc
				httpClient.Transport = t
			}
			h.httpClient = &httpClient
		}
	}

	if h.newAuth == nil {
		h.newAuth = func() auth.Auth {
			return auth.NewAuth(
				auth.WithLog(c.log),
				auth.WithHTTPClient(h.httpClient),
				auth.WithCreds(h.AuthCreds()),
				auth.WithClientID(c.userAgent),
			)
		}
	}

	h.initialized = true
	c.host[host] = h
	return h
}

// getAuth returns an auth, which may be repository specific
func (ch *clientHost) getAuth(repo string) auth.Auth {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	if !ch.config.RepoAuth {
		repo = "" // without RepoAuth, unset the provided repo
	}
	if _, ok := ch.auth[repo]; !ok {
		ch.auth[repo] = ch.newAuth()
	}
	return ch.auth[repo]
}

func (ch *clientHost) AuthCreds() func(h string) auth.Cred {
	if ch == nil || ch.config == nil {
		return auth.DefaultCredsFn
	}
	return func(h string) auth.Cred {
		hCred := ch.config.GetCred()
		return auth.Cred{User: hCred.User, Password: hCred.Password, Token: hCred.Token}
	}
}

// HTTPError returns an error based on the status code
func HTTPError(statusCode int) error {
	switch statusCode {
	case 401:
		return fmt.Errorf("%w [http %d]", types.ErrHTTPUnauthorized, statusCode)
	case 403:
		return fmt.Errorf("%w [http %d]", types.ErrHTTPUnauthorized, statusCode)
	case 404:
		return fmt.Errorf("%w [http %d]", types.ErrNotFound, statusCode)
	case 429:
		return fmt.Errorf("%w [http %d]", types.ErrHTTPRateLimit, statusCode)
	default:
		return fmt.Errorf("%w: %s [http %d]", types.ErrHTTPStatus, http.StatusText(statusCode), statusCode)
	}
}

func makeRootPool(rootCAPool [][]byte, rootCADirs []string, hostname string, hostcert string) (*x509.CertPool, error) {
	pool, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	for _, ca := range rootCAPool {
		if ok := pool.AppendCertsFromPEM(ca); !ok {
			return nil, fmt.Errorf("failed to load ca: %s", ca)
		}
	}
	for _, dir := range rootCADirs {
		hostDir := filepath.Join(dir, hostname)
		files, err := os.ReadDir(hostDir)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to read directory %s: %v", hostDir, err)
			}
			continue
		}
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			if strings.HasSuffix(f.Name(), ".crt") {
				f := filepath.Join(hostDir, f.Name())
				cert, err := os.ReadFile(f)
				if err != nil {
					return nil, fmt.Errorf("failed to read %s: %v", f, err)
				}
				if ok := pool.AppendCertsFromPEM(cert); !ok {
					return nil, fmt.Errorf("failed to import cert from %s", f)
				}
			}
		}
	}
	if hostcert != "" {
		if ok := pool.AppendCertsFromPEM([]byte(hostcert)); !ok {
			// try to parse the certificate and generate a useful error
			block, _ := pem.Decode([]byte(hostcert))
			if block == nil {
				err = fmt.Errorf("pem.Decode is nil")
			} else {
				_, err = x509.ParseCertificate(block.Bytes)
			}
			return nil, fmt.Errorf("failed to load host specific ca (registry: %s): %w: %s", hostname, err, hostcert)
		}
	}
	return pool, nil
}

// sortHostCmp to sort host list of mirrors
func sortHostsCmp(hosts []*clientHost, upstream string) func(i, j int) bool {
	now := time.Now()
	// sort by backoff first, then priority decending, then upstream name last
	return func(i, j int) bool {
		if now.Before(hosts[i].backoffUntil) || now.Before(hosts[j].backoffUntil) {
			return hosts[i].backoffUntil.Before(hosts[j].backoffUntil)
		}
		if hosts[i].config.Priority != hosts[j].config.Priority {
			return hosts[i].config.Priority < hosts[j].config.Priority
		}
		return hosts[i].config.Name != upstream
	}
}
