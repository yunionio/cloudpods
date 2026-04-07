// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

package httpsec

import (
	"encoding/json"
	"net"
	"os"
	"sort"
	"strings"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/appsec/dyngo/instrumentation"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/log"
)

const (
	// envClientIPHeader is the name of the env var used to specify the IP header to be used for client IP collection.
	envClientIPHeader = "DD_TRACE_CLIENT_IP_HEADER"

	// multipleIPHeadersTag sets the multiple ip header tag used internally to tell the backend an error occurred when
	// retrieving an HTTP request client IP.
	multipleIPHeadersTag = "_dd.multiple-ip-headers"

	// BlockedRequestTag used to convey whether a request is blocked
	BlockedRequestTag = "appsec.blocked"
)

var (
	ipv6SpecialNetworks = []*instrumentation.NetaddrIPPrefix{
		ippref("fec0::/10"), // site local
	}

	// List of IP-related headers leveraged to retrieve the public client IP address.
	defaultIPHeaders = []string{
		"x-forwarded-for",
		"x-real-ip",
		"x-client-ip",
		"x-forwarded",
		"x-cluster-client-ip",
		"forwarded-for",
		"forwarded",
		"via",
		"true-client-ip",
	}

	// List of HTTP headers we collect and send.
	collectedHTTPHeaders = append(defaultIPHeaders,
		"host",
		"content-length",
		"content-type",
		"content-encoding",
		"content-language",
		"forwarded",
		"user-agent",
		"accept",
		"accept-encoding",
		"accept-language")

	clientIPHeaderCfg string
)

func init() {
	// Required by sort.SearchStrings
	sort.Strings(defaultIPHeaders[:])
	sort.Strings(collectedHTTPHeaders[:])
	clientIPHeaderCfg = os.Getenv(envClientIPHeader)
}

// SetSecurityEventTags sets the AppSec-specific span tags when a security event occurred into the service entry span.
func SetSecurityEventTags(span instrumentation.TagSetter, events []json.RawMessage, headers, respHeaders map[string][]string) {
	if err := instrumentation.SetEventSpanTags(span, events); err != nil {
		log.Error("appsec: unexpected error while creating the appsec event tags: %v", err)
	}
	for h, v := range NormalizeHTTPHeaders(headers) {
		span.SetTag("http.request.headers."+h, v)
	}
	for h, v := range NormalizeHTTPHeaders(respHeaders) {
		span.SetTag("http.response.headers."+h, v)
	}
}

// NormalizeHTTPHeaders returns the HTTP headers following Datadog's
// normalization format.
func NormalizeHTTPHeaders(headers map[string][]string) (normalized map[string]string) {
	if len(headers) == 0 {
		return nil
	}
	normalized = make(map[string]string)
	for k, v := range headers {
		k = strings.ToLower(k)
		if i := sort.SearchStrings(collectedHTTPHeaders[:], k); i < len(collectedHTTPHeaders) && collectedHTTPHeaders[i] == k {
			normalized[k] = strings.Join(v, ",")
		}
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

// ippref returns the IP network from an IP address string s. If not possible, it returns nil.
func ippref(s string) *instrumentation.NetaddrIPPrefix {
	if prefix, err := instrumentation.NetaddrParseIPPrefix(s); err == nil {
		return &prefix
	}
	return nil
}

// ClientIPTags generates the IP related span tags for a given request headers
func ClientIPTags(hdrs map[string][]string, remoteAddr string) (tags map[string]string, clientIP instrumentation.NetaddrIP) {
	tags = map[string]string{}
	monitoredHeaders := defaultIPHeaders
	if clientIPHeaderCfg != "" {
		monitoredHeaders = []string{clientIPHeaderCfg}
	}

	// Filter the list of headers
	foundHeaders := map[string][]string{}
	for k, v := range hdrs {
		k = strings.ToLower(k)
		if i := sort.SearchStrings(monitoredHeaders, k); i < len(monitoredHeaders) && monitoredHeaders[i] == k {
			if len(v) >= 1 && v[0] != "" {
				foundHeaders[k] = v
			}
		}
	}

	// If more than one IP header is present, report them and don't return any client ip
	if len(foundHeaders) > 1 {
		var headers []string
		for header, ips := range foundHeaders {
			tags[ext.HTTPRequestHeaders+"."+header] = strings.Join(ips, ",")
			headers = append(headers, header)
		}
		sort.Strings(headers) // produce a predictable value
		tags[multipleIPHeadersTag] = strings.Join(headers, ",")
		return tags, instrumentation.NetaddrIP{}
	}

	// Walk IP-related headers
	var foundIP instrumentation.NetaddrIP
	for _, v := range foundHeaders {
		// Handle multi-value headers by flattening the list of values
		var ips []string
		for _, ip := range v {
			ips = append(ips, strings.Split(ip, ",")...)
		}

		// Look for the first valid or global IP address in the comma-separated list
		for _, ipstr := range ips {
			ip := parseIP(strings.TrimSpace(ipstr))
			if !ip.IsValid() {
				continue
			}
			// Replace foundIP if still not valid in order to keep the oldest
			if !foundIP.IsValid() {
				foundIP = ip
			}
			if isGlobal(ip) {
				foundIP = ip
				break
			}
		}
	}

	// Decide which IP address is the client one by starting with the remote IP
	remoteIP := parseIP(remoteAddr)
	if remoteIP.IsValid() {
		tags["network.client.ip"] = remoteIP.String()
		clientIP = remoteIP
	}

	// The IP address found in the headers supersedes a private remote IP address.
	if foundIP.IsValid() && !isGlobal(remoteIP) || isGlobal(foundIP) {
		clientIP = foundIP
	}

	if clientIP.IsValid() {
		tags[ext.HTTPClientIP] = clientIP.String()
	}

	return tags, clientIP
}

func parseIP(s string) instrumentation.NetaddrIP {
	if ip, err := instrumentation.NetaddrParseIP(s); err == nil {
		return ip
	}
	if h, _, err := net.SplitHostPort(s); err == nil {
		if ip, err := instrumentation.NetaddrParseIP(h); err == nil {
			return ip
		}
	}
	return instrumentation.NetaddrIP{}
}

func isGlobal(ip instrumentation.NetaddrIP) bool {
	// IsPrivate also checks for ipv6 ULA.
	// We care to check for these addresses are not considered public, hence not global.
	// See https://www.rfc-editor.org/rfc/rfc4193.txt for more details.
	isGlobal := ip.IsValid() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast()
	if !isGlobal || !ip.Is6() {
		return isGlobal
	}
	for _, n := range ipv6SpecialNetworks {
		if n.Contains(ip) {
			return false
		}
	}
	return isGlobal
}
