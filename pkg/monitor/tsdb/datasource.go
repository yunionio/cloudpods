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

package tsdb

import (
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"time"

	"yunion.io/x/onecloud/pkg/monitor/options"
)

type DataSource struct {
	Id                string
	Name              string
	Type              string
	Url               string
	User              string
	Password          string
	Database          string
	BasicAuth         bool
	BasicAuthUser     string
	BasicAuthPassword string
	TimeInterval      string
	Updated           time.Time
}

type proxyTransportCache struct {
	cache map[string]cachedTransport
	sync.Mutex
}

// dataSourceTransport implements http.RoundTripper (https://golang.org/pkg/net/http/#RoundTripper)
type dataSourceTransport struct {
	headers   map[string]string
	transport *http.Transport
}

// RoundTrip executes a single HTTP transaction, returning a Response for the provided Request.
func (d *dataSourceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for key, value := range d.headers {
		req.Header.Set(key, value)
	}

	return d.transport.RoundTrip(req)
}

type cachedTransport struct {
	updated time.Time

	*dataSourceTransport
}

var ptc = proxyTransportCache{
	cache: make(map[string]cachedTransport),
}

func (ds *DataSource) GetHttpClient() (*http.Client, error) {
	transport, err := ds.GetHttpTransport()

	if err != nil {
		return nil, err
	}

	return &http.Client{
		//Timeout:   30 * time.Second,
		Transport: transport,
	}, nil
}

// getCustomHeaders returns a map with all the to be set headers
// The map key represents the HeaderName and the value represetns this header's value
func (ds *DataSource) getCustomHeaders() map[string]string {
	headers := make(map[string]string)
	// TODO: datasource support config customize headers
	return headers
}

func (ds *DataSource) GetHttpTransport() (*dataSourceTransport, error) {
	ptc.Lock()
	defer ptc.Unlock()

	if t, present := ptc.cache[ds.Id]; present && ds.Updated.Equal(t.updated) {
		return t.dataSourceTransport, nil
	}

	tlsConfig, err := ds.GetTLSConfig()
	if err != nil {
		return nil, err
	}

	tlsConfig.Renegotiation = tls.RenegotiateFreelyAsClient

	// Create transport which adds all
	// TODO: use httputils.HttpTransport instead -- qj
	customHeaders := ds.getCustomHeaders()
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
		Proxy:           http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(options.Options.DataProxyTimeout) * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
	}

	dsTransport := &dataSourceTransport{
		headers:   customHeaders,
		transport: transport,
	}

	ptc.cache[ds.Id] = cachedTransport{
		dataSourceTransport: dsTransport,
		updated:             ds.Updated,
	}

	return dsTransport, nil
}

func (ds *DataSource) GetTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	return tlsConfig, nil
}

/*
func (ds *DataSource) DecryptedBasicAuthPassword() string {
	return ds.decryptedValue("basicAuthPassword", ds.BasicAuthPassword)
}

func (ds *DataSource) DecryptedPassword() string {
	return ds.decryptedValue("password", ds.Password)
}

func (ds *DataSource) decryptedValue(field string, fallback string) string {
	if value, ok := ds.DecryptedValue(field); ok {
		return value
	}
	return fallback
}

// DecryptedValue returns cached decrypted value from cached data
func (ds *DataSource) DecryptedValue(key string) (string, bool) {
	value, exists := ds.DecryptedValues()[key]
	return value, exists
}

var dsDescryptionCache =

func (ds *DataSource) DecryptedValues() map[string]string {

}*/
