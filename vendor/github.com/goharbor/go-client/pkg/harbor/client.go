package harbor

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	assistclient "github.com/goharbor/go-client/pkg/sdk/assist/client"
	v2client "github.com/goharbor/go-client/pkg/sdk/v2.0/client"
	legacyclient "github.com/goharbor/go-client/pkg/sdk/v2.0/legacy/client"
)

// InsecureTransport provides a insecure RoundTripper and disable the HTTP2 try.
var InsecureTransport http.RoundTripper = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second, // nolint:gomnd
		KeepAlive: 30 * time.Second, // nolint:gomnd
		DualStack: true,
	}).DialContext,
	TLSClientConfig: &tls.Config{
		InsecureSkipVerify: true, // nolint:gosec
	},
	MaxIdleConns:          100,              // nolint:gomnd
	IdleConnTimeout:       90 * time.Second, // nolint:gomnd
	TLSHandshakeTimeout:   10 * time.Second, // nolint:gomnd
	ExpectContinueTimeout: 1 * time.Second,
}

// Config contains configs for constructing a client
type Config struct {
	// URL is the base URL of the upstream server
	URL *url.URL
	// Transport is an inner transport for the client
	Transport http.RoundTripper
	// AuthInfo is for authentication
	AuthInfo runtime.ClientAuthInfoWriter
}

// ClientSetConfig contains config for creating a ClientSet
type ClientSetConfig struct {
	URL      string
	Insecure bool
	Username string
	Password string
}

// ClientSet contains clients for V2, Assist, Legacy
type ClientSet struct {
	assistClient *assistclient.HarborAPI
	v2Client     *v2client.HarborAPI
	legacyClient *legacyclient.HarborAPI
}

// ToAssistConfig convert the Config to assistclient's Config
func (c *Config) ToAssistConfig() assistclient.Config {

	var (
		u url.URL                      = url.URL{}
		t http.RoundTripper            = c.Transport
		a runtime.ClientAuthInfoWriter = c.AuthInfo
	)

	if c.URL != nil {
		u = *c.URL
	}

	if len(u.Host) == 0 {
		u.Host = assistclient.DefaultHost
	}
	if len(u.Path) == 0 {
		u.Path = assistclient.DefaultBasePath
	}
	if len(u.Scheme) == 0 {
		// default to https if not set
		u.Scheme = assistclient.DefaultSchemes[1]
	}

	if t == nil {
		t = InsecureTransport
	}

	return assistclient.Config{
		URL:       &u,
		Transport: t,
		AuthInfo:  a,
	}
}

// ToV2Config convert the Config to v2client's Config
func (c *Config) ToV2Config() v2client.Config {
	var (
		u url.URL                      = url.URL{}
		t http.RoundTripper            = c.Transport
		a runtime.ClientAuthInfoWriter = c.AuthInfo
	)

	if c.URL != nil {
		u = *c.URL
	}

	if len(u.Host) == 0 {
		u.Host = v2client.DefaultHost
	}
	if len(u.Path) == 0 {
		u.Path = v2client.DefaultBasePath
	}
	if len(u.Scheme) == 0 {
		// default to https if not set
		u.Scheme = v2client.DefaultSchemes[1]
	}

	if t == nil {
		t = InsecureTransport
	}

	return v2client.Config{
		URL:       &u,
		Transport: t,
		AuthInfo:  a,
	}
}

// ToLegacyConfig convert the Config to legacyclient's Config
func (c *Config) ToLegacyConfig() legacyclient.Config {
	var (
		u url.URL                      = url.URL{}
		t http.RoundTripper            = c.Transport
		a runtime.ClientAuthInfoWriter = c.AuthInfo
	)

	if c.URL != nil {
		u = *c.URL
	}

	if len(u.Host) == 0 {
		u.Host = legacyclient.DefaultHost
	}
	if len(u.Path) == 0 {
		u.Path = legacyclient.DefaultBasePath
	}
	if len(u.Scheme) == 0 {
		// default to https if not set
		u.Scheme = legacyclient.DefaultSchemes[1]
	}

	if t == nil {
		t = InsecureTransport
	}

	return legacyclient.Config{
		URL:       &u,
		Transport: t,
		AuthInfo:  a,
	}
}

// NewAssistClient return assist Client
func NewClientSet(csc *ClientSetConfig) (*ClientSet, error) {
	if csc == nil {
		csc = &ClientSetConfig{}
	}

	u, err := url.Parse(csc.URL)
	if err != nil {
		return nil, err
	}

	c := &Config{
		URL:      u,
		AuthInfo: httptransport.BasicAuth(csc.Username, csc.Password),
	}

	if csc.Insecure {
		c.Transport = InsecureTransport
	}

	cs := &ClientSet{}
	cs.assistClient = assistclient.New(c.ToAssistConfig())
	cs.legacyClient = legacyclient.New(c.ToLegacyConfig())
	cs.v2Client = v2client.New(c.ToV2Config())

	return cs, nil
}

// Assist return AssistClient
func (c *ClientSet) Assist() *assistclient.HarborAPI {
	return c.assistClient
}

// V2 return V2Client
func (c *ClientSet) V2() *v2client.HarborAPI {
	return c.v2Client
}

// Legacy return LegacyClient
func (c *ClientSet) Legacy() *legacyclient.HarborAPI {
	return c.legacyClient
}
