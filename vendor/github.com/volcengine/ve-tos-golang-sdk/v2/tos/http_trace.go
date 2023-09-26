package tos

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"sync/atomic"
)

type accessLogRequest struct {
	clientDnsCost                int64
	clientDialCost               int64
	clientTlsHandShakeCost       int64
	clientSendHeadersAndBodyCost int64
	clientWaitResponseCost       int64
	clientSendRequestCost        int64
	actionStartMs                int64
}

func newAccessLogRequest(actionStartMs int64) *accessLogRequest {
	return &accessLogRequest{
		clientDnsCost:                -1,
		clientDialCost:               -1,
		clientTlsHandShakeCost:       -1,
		clientSendHeadersAndBodyCost: -1,
		clientWaitResponseCost:       -1,
		clientSendRequestCost:        -1,
		actionStartMs:                actionStartMs,
	}
}

func (r *accessLogRequest) PrintAccessLog(logger Logger, req *http.Request, response *http.Response) {
	if logger == nil {
		return
	}
	atomic.CompareAndSwapInt64(&r.clientSendRequestCost, -1, GetUnixTimeMs()-r.actionStartMs)
	var requestId *string
	if response != nil {
		requestId = StringPtr(response.Header.Get(HeaderRequestID))
	}
	prefix := buildPrefix(requestId)
	if req != nil {
		logger.Debug(fmt.Sprintf("%s, method: %s, host: %s, request uri: %s, dns cost: %d ms, dial cost: %d ms, tls handshake cost: %d ms, send headers and body cost: %d ms, wait response cost: %d ms, request cost: %d ms",
			prefix, req.Method, req.URL.Host, req.URL.EscapedPath(), r.clientDnsCost, r.clientDialCost, r.clientTlsHandShakeCost,
			r.clientSendHeadersAndBodyCost, r.clientWaitResponseCost, r.clientSendRequestCost))
	} else {
		logger.Debug(fmt.Sprintf("%s, dns cost: %d ms, dial cost: %d ms, tls handshake cost: %d ms, send headers and body cost: %d ms, wait response cost: %d ms, request cost: %d ms",
			prefix, r.clientDnsCost, r.clientDialCost, r.clientTlsHandShakeCost, r.clientSendHeadersAndBodyCost, r.clientWaitResponseCost, r.clientSendRequestCost))
	}
}

func buildPrefix(requestId *string) string {
	prefix := ""

	if requestId != nil {
		prefix = fmt.Sprintf("[requestId: %s] %s", *requestId, prefix)
	}
	return prefix
}

func getClientTrace(actionStartMs int64) (*httptrace.ClientTrace, *accessLogRequest) {
	var dnsStart int64
	var dialStart int64
	var tlsHandShakeStart int64
	var sendHeadersAndBodyStart int64
	var waitResponseStart int64
	r := newAccessLogRequest(actionStartMs)
	trace := &httptrace.ClientTrace{
		GotFirstResponseByte: func() {
			r.clientWaitResponseCost = GetUnixTimeMs() - waitResponseStart
		},
		DNSStart: func(info httptrace.DNSStartInfo) {
			dnsStart = GetUnixTimeMs()
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			r.clientDnsCost = GetUnixTimeMs() - dnsStart
		},
		ConnectStart: func(network, addr string) {
			dialStart = GetUnixTimeMs()
		},
		ConnectDone: func(network, addr string, err error) {
			now := GetUnixTimeMs()
			sendHeadersAndBodyStart = now
			r.clientDialCost = now - dialStart
		},
		TLSHandshakeStart: func() {
			tlsHandShakeStart = GetUnixTimeMs()
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			now := GetUnixTimeMs()
			sendHeadersAndBodyStart = now
			r.clientTlsHandShakeCost = now - tlsHandShakeStart
		},

		GotConn: func(httptrace.GotConnInfo) {
			sendHeadersAndBodyStart = GetUnixTimeMs()
		},

		WroteRequest: func(info httptrace.WroteRequestInfo) {
			waitResponseStart = GetUnixTimeMs()
			r.clientSendHeadersAndBodyCost = waitResponseStart - sendHeadersAndBodyStart
		},
	}
	return trace, r
}
