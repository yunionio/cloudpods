package tos

import (
	"context"
	"crypto/tls"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptrace"
	"time"
)

type TransportConfig struct {

	// MaxIdleConns same as http.Transport MaxIdleConns. Default is 1024.
	MaxIdleConns int

	// MaxIdleConnsPerHost same as http.Transport MaxIdleConnsPerHost. Default is 1024.
	MaxIdleConnsPerHost int

	// MaxConnsPerHost same as http.Transport MaxConnsPerHost. Default is no limit.
	MaxConnsPerHost int

	// RequestTimeout same as http.Client Timeout
	// Deprecated: use ReadTimeout or WriteTimeout instead
	RequestTimeout time.Duration

	// DialTimeout same as net.Dialer Timeout
	DialTimeout time.Duration

	// KeepAlive same as net.Dialer KeepAlive
	KeepAlive time.Duration

	// IdleConnTimeout same as http.Transport IdleConnTimeout
	IdleConnTimeout time.Duration

	// TLSHandshakeTimeout same as http.Transport TLSHandshakeTimeout
	TLSHandshakeTimeout time.Duration

	// ResponseHeaderTimeout same as http.Transport ResponseHeaderTimeout
	ResponseHeaderTimeout time.Duration

	// ExpectContinueTimeout same as http.Transport ExpectContinueTimeout
	ExpectContinueTimeout time.Duration

	// ReadTimeout see net.Conn SetReadDeadline
	ReadTimeout time.Duration

	// WriteTimeout set net.Conn SetWriteDeadline
	WriteTimeout time.Duration

	// InsecureSkipVerify set tls.Config InsecureSkipVerify
	InsecureSkipVerify bool

	// DNSCacheTime Set Dns Cache Time.
	DNSCacheTime time.Duration

	// Proxy Set http proxy for http client.
	Proxy *Proxy
}

type Transport interface {
	RoundTrip(context.Context, *Request) (*Response, error)
}

type DefaultTransport struct {
	client http.Client
	logger Logger
}

func (d *DefaultTransport) WithDefaultTransportLogger(logger Logger) {
	d.logger = logger
}

// NewDefaultTransport create a DefaultTransport with config
func NewDefaultTransport(config *TransportConfig) *DefaultTransport {
	var r *resolver

	if config.DNSCacheTime >= time.Minute {
		r = newResolver(config.DNSCacheTime)
	}

	transport := &http.Transport{
		DialContext: (&TimeoutDialer{
			Dialer: net.Dialer{
				Timeout:   config.DialTimeout,
				KeepAlive: config.KeepAlive,
			},
			resolver:     r,
			ReadTimeout:  config.ReadTimeout,
			WriteTimeout: config.WriteTimeout,
		}).DialContext,
		MaxIdleConns:          config.MaxIdleConns,
		MaxIdleConnsPerHost:   config.MaxIdleConnsPerHost,
		MaxConnsPerHost:       config.MaxConnsPerHost,
		IdleConnTimeout:       config.IdleConnTimeout,
		TLSHandshakeTimeout:   config.TLSHandshakeTimeout,
		ResponseHeaderTimeout: config.ResponseHeaderTimeout,
		ExpectContinueTimeout: config.ExpectContinueTimeout,
		DisableCompression:    true,
		// #nosec G402
		TLSClientConfig: &tls.Config{InsecureSkipVerify: config.InsecureSkipVerify},
	}

	if config.Proxy != nil && config.Proxy.proxyHost != "" {
		transport.Proxy = http.ProxyURL(config.Proxy.Url())
	}

	return &DefaultTransport{
		client: http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Transport: transport,
		},
	}
}

//   newDefaultTranposrtWithHTTPTransport
func newDefaultTranposrtWithHTTPTransport(transport http.RoundTripper) *DefaultTransport {
	return &DefaultTransport{
		client: http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Transport: transport,
		},
	}
}

// NewDefaultTransportWithClient crate a DefaultTransport with a http.Client
func NewDefaultTransportWithClient(client http.Client) *DefaultTransport {
	return &DefaultTransport{client: client}
}

func (dt *DefaultTransport) RoundTrip(ctx context.Context, req *Request) (*Response, error) {
	hr, err := http.NewRequestWithContext(ctx, req.Method, req.URL(), req.Content)
	if err != nil {
		return nil, newTosClientError(err.Error(), err)
	}

	if req.ContentLength != nil {
		hr.ContentLength = *req.ContentLength
	}

	for key, values := range req.Header {
		hr.Header[key] = values
	}

	var accessLog *accessLogRequest
	if dt.logger != nil {
		var trace *httptrace.ClientTrace
		trace, accessLog = getClientTrace(GetUnixTimeMs())
		ctx = httptrace.WithClientTrace(ctx, trace)
		hr = hr.WithContext(ctx)
	}
	res, err := dt.client.Do(hr)

	if accessLog != nil {
		accessLog.PrintAccessLog(dt.logger, hr, res)
	}

	if err != nil {
		return nil, newTosClientError(err.Error(), err)
	}

	return &Response{
		StatusCode:    res.StatusCode,
		ContentLength: res.ContentLength,
		Header:        res.Header,
		Body:          res.Body,
	}, nil
}

type TimeoutConn struct {
	net.Conn
	readTimeout  time.Duration
	writeTimeout time.Duration
	zero         time.Time
}

func NewTimeoutConn(conn net.Conn, readTimeout, writeTimeout time.Duration) *TimeoutConn {
	return &TimeoutConn{
		Conn:         conn,
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
	}
}

func (tc *TimeoutConn) Read(b []byte) (n int, err error) {
	timeout := tc.readTimeout > 0
	if timeout {
		_ = tc.SetReadDeadline(time.Now().Add(tc.readTimeout))
	}

	n, err = tc.Conn.Read(b)
	if timeout {
		_ = tc.SetReadDeadline(time.Now().Add(tc.readTimeout * 5))
	}
	return n, err
}

func (tc *TimeoutConn) Write(b []byte) (n int, err error) {
	timeout := tc.writeTimeout > 0
	if timeout {
		_ = tc.SetWriteDeadline(time.Now().Add(tc.writeTimeout))
	}

	n, err = tc.Conn.Write(b)
	if tc.readTimeout > 0 {
		_ = tc.SetReadDeadline(time.Now().Add(tc.readTimeout * 5))
	}
	return n, err
}

type TimeoutDialer struct {
	net.Dialer
	resolver     *resolver
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func (d *TimeoutDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if d.resolver != nil {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ipList, err := d.resolver.GetIpList(ctx, host)
		if err != nil {
			return nil, err
		}

		// 随机打乱 IP List
		rand.Shuffle(len(ipList), func(i, j int) {
			ipList[i], ipList[j] = ipList[j], ipList[i]
		})

		for _, ip := range ipList {
			conn, err := d.Dialer.DialContext(ctx, network, ip+":"+port)
			if err == nil {
				return NewTimeoutConn(conn, d.ReadTimeout, d.WriteTimeout), nil
			} else {
				d.resolver.Remove(address, ip)
			}
		}
	}

	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	return NewTimeoutConn(conn, d.ReadTimeout, d.WriteTimeout), nil
}
