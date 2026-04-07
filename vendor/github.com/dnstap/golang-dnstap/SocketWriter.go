/*
 * Copyright (c) 2019 by Farsight Security, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dnstap

import (
	"net"
	"sync"
	"time"

	framestream "github.com/farsightsec/golang-framestream"
)

// A SocketWriter writes data to a Frame Streams TCP or Unix domain socket,
// establishing or restarting the connection if needed.
type socketWriter struct {
	w    Writer
	c    net.Conn
	addr net.Addr
	opt  SocketWriterOptions
}

// SocketWriterOptions provides configuration options for a SocketWriter
type SocketWriterOptions struct {
	// Timeout gives the time the SocketWriter will wait for reads and
	// writes to complete.
	Timeout time.Duration
	// FlushTimeout is the maximum duration data will be buffered while
	// being written to the socket.
	FlushTimeout time.Duration
	// RetryInterval is how long the SocketWriter will wait between
	// connection attempts.
	RetryInterval time.Duration
	// Dialer is the dialer used to establish the connection. If nil,
	// SocketWriter will use a default dialer with a 30 second timeout.
	Dialer *net.Dialer
	// Logger provides the logger for connection establishment, reconnection,
	// and error events of the SocketWriter.
	Logger Logger
}

type flushWriter struct {
	m           sync.Mutex
	w           *framestream.Writer
	d           time.Duration
	timer       *time.Timer
	timerActive bool
	lastFlushed time.Time
	stopped     bool
}

type flusherConn struct {
	net.Conn
	lastWritten *time.Time
}

func (c *flusherConn) Write(p []byte) (int, error) {
	n, err := c.Conn.Write(p)
	*c.lastWritten = time.Now()
	return n, err
}

func newFlushWriter(c net.Conn, d time.Duration) (*flushWriter, error) {
	var err error
	fw := &flushWriter{timer: time.NewTimer(d), d: d}
	if !fw.timer.Stop() {
		<-fw.timer.C
	}

	fc := &flusherConn{
		Conn:        c,
		lastWritten: &fw.lastFlushed,
	}

	fw.w, err = framestream.NewWriter(fc,
		&framestream.WriterOptions{
			ContentTypes:  [][]byte{FSContentType},
			Bidirectional: true,
			Timeout:       d,
		})
	if err != nil {
		return nil, err
	}
	go fw.runFlusher()
	return fw, nil
}

func (fw *flushWriter) runFlusher() {
	for range fw.timer.C {
		fw.m.Lock()
		if fw.stopped {
			fw.m.Unlock()
			return
		}
		last := fw.lastFlushed
		elapsed := time.Since(last)
		if elapsed < fw.d {
			fw.timer.Reset(fw.d - elapsed)
			fw.m.Unlock()
			continue
		}
		fw.w.Flush()
		fw.timerActive = false
		fw.m.Unlock()
	}
}

func (fw *flushWriter) WriteFrame(p []byte) (int, error) {
	fw.m.Lock()
	n, err := fw.w.WriteFrame(p)
	if !fw.timerActive {
		fw.timer.Reset(fw.d)
		fw.timerActive = true
	}
	fw.m.Unlock()
	return n, err
}

func (fw *flushWriter) Close() error {
	fw.m.Lock()
	fw.stopped = true
	fw.timer.Reset(0)
	err := fw.w.Close()
	fw.m.Unlock()
	return err
}

// NewSocketWriter creates a SocketWriter which writes data to a connection
// to the given addr. The SocketWriter maintains and re-establishes the
// connection to this address as needed.
func NewSocketWriter(addr net.Addr, opt *SocketWriterOptions) Writer {
	if opt == nil {
		opt = &SocketWriterOptions{}
	}

	if opt.Logger == nil {
		opt.Logger = &nullLogger{}
	}
	return &socketWriter{addr: addr, opt: *opt}
}

func (sw *socketWriter) openWriter() error {
	var err error
	sw.c, err = sw.opt.Dialer.Dial(sw.addr.Network(), sw.addr.String())
	if err != nil {
		return err
	}

	wopt := WriterOptions{
		Bidirectional: true,
		Timeout:       sw.opt.Timeout,
	}

	if sw.opt.FlushTimeout == 0 {
		sw.w, err = NewWriter(sw.c, &wopt)
	} else {
		sw.w, err = newFlushWriter(sw.c, sw.opt.FlushTimeout)
	}
	if err != nil {
		sw.c.Close()
		return err
	}
	return nil
}

// Close shuts down the SocketWriter, closing any open connection.
func (sw *socketWriter) Close() error {
	var err error
	if sw.w != nil {
		err = sw.w.Close()
		if err == nil {
			return sw.c.Close()
		}
		sw.c.Close()
		return err
	}
	if sw.c != nil {
		return sw.c.Close()
	}
	return nil
}

// Write writes the data in p as a Dnstap frame to a connection to the
// SocketWriter's address. Write may block indefinitely while the SocketWriter
// attempts to establish or re-establish the connection and FrameStream session.
func (sw *socketWriter) WriteFrame(p []byte) (int, error) {
	for ; ; time.Sleep(sw.opt.RetryInterval) {
		if sw.w == nil {
			if err := sw.openWriter(); err != nil {
				sw.opt.Logger.Printf("%s: open failed: %v", sw.addr, err)
				continue
			}
		}

		n, err := sw.w.WriteFrame(p)
		if err != nil {
			sw.opt.Logger.Printf("%s: write failed: %v", sw.addr, err)
			sw.Close()
			continue
		}

		return n, nil
	}
}
