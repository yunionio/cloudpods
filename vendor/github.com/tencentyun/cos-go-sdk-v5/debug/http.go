package debug

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
)

// DebugRequestTransport 会打印请求和响应信息, 方便调试.
type DebugRequestTransport struct {
	RequestHeader  bool
	RequestBody    bool // RequestHeader 为 true 时,这个选项才会生效
	ResponseHeader bool
	ResponseBody   bool // ResponseHeader 为 true 时,这个选项才会生效

	// debug 信息输出到 Writer 中, 默认是 os.Stderr
	Writer io.Writer

	Transport http.RoundTripper
}

// RoundTrip implements the RoundTripper interface.
func (t *DebugRequestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = cloneRequest(req) // per RoundTrip contract
	w := t.Writer
	if w == nil {
		w = os.Stderr
	}

	if t.RequestHeader {
		a, _ := httputil.DumpRequest(req, t.RequestBody)
		fmt.Fprintf(w, "%s\n\n", string(a))
	}

	resp, err := t.transport().RoundTrip(req)
	if err != nil {
		return resp, err
	}

	if t.ResponseHeader {

		b, _ := httputil.DumpResponse(resp, t.ResponseBody)
		fmt.Fprintf(w, "%s\n", string(b))
	}

	return resp, err
}

func (t *DebugRequestTransport) transport() http.RoundTripper {
	if t.Transport != nil {
		return t.Transport
	}
	return http.DefaultTransport
}

// cloneRequest returns a clone of the provided *http.Request. The clone is a
// shallow copy of the struct and its Header map.
func cloneRequest(r *http.Request) *http.Request {
	// shallow copy of the struct
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header, len(r.Header))
	for k, s := range r.Header {
		r2.Header[k] = append([]string(nil), s...)
	}
	return r2
}
