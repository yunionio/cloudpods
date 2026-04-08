package framestream

import (
	"io"
	"net"
	"time"
)

type timeoutConn struct {
	conn                      net.Conn
	readTimeout, writeTimeout time.Duration
}

func (toc *timeoutConn) Write(b []byte) (int, error) {
	if toc.writeTimeout != 0 {
		toc.conn.SetWriteDeadline(time.Now().Add(toc.writeTimeout))
	}
	return toc.conn.Write(b)
}

func (toc *timeoutConn) Read(b []byte) (int, error) {
	if toc.readTimeout != 0 {
		toc.conn.SetReadDeadline(time.Now().Add(toc.readTimeout))
	}
	return toc.conn.Read(b)
}

func timeoutWriter(w io.Writer, opt *WriterOptions) io.Writer {
	if !opt.Bidirectional {
		return w
	}
	if opt.Timeout == 0 {
		return w
	}
	if c, ok := w.(net.Conn); ok {
		return &timeoutConn{
			conn:         c,
			readTimeout:  opt.Timeout,
			writeTimeout: opt.Timeout,
		}
	}
	return w
}

func timeoutReader(r io.Reader, opt *ReaderOptions) io.Reader {
	if !opt.Bidirectional {
		return r
	}
	if opt.Timeout == 0 {
		return r
	}
	if c, ok := r.(net.Conn); ok {
		return &timeoutConn{
			conn:         c,
			readTimeout:  opt.Timeout,
			writeTimeout: opt.Timeout,
		}
	}
	return r
}

func disableReadTimeout(r io.Reader) {
	if tc, ok := r.(*timeoutConn); ok {
		tc.readTimeout = 0
		tc.conn.SetReadDeadline(time.Time{})
	}
}
