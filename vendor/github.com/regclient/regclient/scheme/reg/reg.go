// Package reg implements the OCI registry scheme used by most images (host:port/repo:tag)
package reg

import (
	"net/http"
	"sync"
	"time"

	"github.com/regclient/regclient/config"
	"github.com/regclient/regclient/internal/reghttp"
	"github.com/regclient/regclient/scheme"
	"github.com/sirupsen/logrus"
)

const (
	// blobChunkMinHeader is returned by registries requesting a minimum chunk size
	blobChunkMinHeader = "OCI-Chunk-Min-Length"
	// defaultBlobChunk 1M chunks, this is allocated in a memory buffer
	defaultBlobChunk = 1024 * 1024
	// defaultBlobChunkLimit 1G chunks, prevents a memory exhaustion attack
	defaultBlobChunkLimit = 1024 * 1024 * 1024
	// defaultBlobMax is disabled to support registries without chunked upload support
	defaultBlobMax = -1
)

// Reg is used for interacting with remote registry servers
type Reg struct {
	reghttp        *reghttp.Client
	reghttpOpts    []reghttp.Opts
	log            *logrus.Logger
	hosts          map[string]*config.Host
	blobChunkSize  int64
	blobChunkLimit int64
	blobMaxPut     int64
	mu             sync.Mutex
}

// Opts provides options to access registries
type Opts func(*Reg)

// New returns a Reg pointer with any provided options
func New(opts ...Opts) *Reg {
	r := Reg{
		reghttpOpts:    []reghttp.Opts{},
		blobChunkSize:  defaultBlobChunk,
		blobChunkLimit: defaultBlobChunkLimit,
		blobMaxPut:     defaultBlobMax,
		hosts:          map[string]*config.Host{},
	}
	for _, opt := range opts {
		opt(&r)
	}
	r.reghttp = reghttp.NewClient(r.reghttpOpts...)
	return &r
}

// Info is experimental and may be removed in the future
func (reg *Reg) Info() scheme.Info {
	return scheme.Info{}
}

func (reg *Reg) hostGet(hostname string) *config.Host {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	if _, ok := reg.hosts[hostname]; !ok {
		reg.hosts[hostname] = config.HostNewName(hostname)
	}
	return reg.hosts[hostname]
}

// WithBlobSize overrides default blob sizes
func WithBlobSize(size, max int64) Opts {
	return func(r *Reg) {
		if size > 0 {
			r.blobChunkSize = size
		}
		if max != 0 {
			r.blobMaxPut = max
		}
	}
}

// WithBlobLimit overrides default blob limit
func WithBlobLimit(limit int64) Opts {
	return func(r *Reg) {
		if limit > 0 {
			r.blobChunkLimit = limit
		}
		if r.blobMaxPut > 0 && r.blobMaxPut < limit {
			r.blobMaxPut = limit
		}
	}
}

// WithCerts adds certificates
func WithCerts(certs [][]byte) Opts {
	return func(r *Reg) {
		r.reghttpOpts = append(r.reghttpOpts, reghttp.WithCerts(certs))
	}
}

// WithCertDirs adds certificate directories for host specific certs
func WithCertDirs(dirs []string) Opts {
	return func(r *Reg) {
		r.reghttpOpts = append(r.reghttpOpts, reghttp.WithCertDirs(dirs))
	}
}

// WithCertFiles adds certificates by filename
func WithCertFiles(files []string) Opts {
	return func(r *Reg) {
		r.reghttpOpts = append(r.reghttpOpts, reghttp.WithCertFiles(files))
	}
}

// WithConfigHosts adds host configs for credentials
func WithConfigHosts(configHosts []*config.Host) Opts {
	return func(r *Reg) {
		for _, host := range configHosts {
			if host.Name == "" {
				continue
			}
			r.hosts[host.Name] = host
		}
		r.reghttpOpts = append(r.reghttpOpts, reghttp.WithConfigHosts(configHosts))
	}
}

// WithDelay initial time to wait between retries (increased with exponential backoff)
func WithDelay(delayInit time.Duration, delayMax time.Duration) Opts {
	return func(r *Reg) {
		r.reghttpOpts = append(r.reghttpOpts, reghttp.WithDelay(delayInit, delayMax))
	}
}

// WithHTTPClient uses a specific http client with retryable requests
func WithHTTPClient(hc *http.Client) Opts {
	return func(r *Reg) {
		r.reghttpOpts = append(r.reghttpOpts, reghttp.WithHTTPClient(hc))
	}
}

// WithLog injects a logrus Logger configuration
func WithLog(log *logrus.Logger) Opts {
	return func(r *Reg) {
		r.log = log
		r.reghttpOpts = append(r.reghttpOpts, reghttp.WithLog(log))
	}
}

// WithRetryLimit restricts the number of retries (defaults to 5)
func WithRetryLimit(l int) Opts {
	return func(r *Reg) {
		r.reghttpOpts = append(r.reghttpOpts, reghttp.WithRetryLimit(l))
	}
}

// WithTransport uses a specific http transport with retryable requests
func WithTransport(t *http.Transport) Opts {
	return func(r *Reg) {
		r.reghttpOpts = append(r.reghttpOpts, reghttp.WithTransport(t))
	}
}

// WithUserAgent sets a user agent header
func WithUserAgent(ua string) Opts {
	return func(r *Reg) {
		r.reghttpOpts = append(r.reghttpOpts, reghttp.WithUserAgent(ua))
	}
}
