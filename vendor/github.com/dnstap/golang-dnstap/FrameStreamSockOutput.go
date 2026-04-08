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
	"time"
)

// A FrameStreamSockOutput manages a socket connection and sends dnstap
// data over a framestream connection on that socket.
type FrameStreamSockOutput struct {
	address       net.Addr
	outputChannel chan []byte
	wait          chan bool
	wopt          SocketWriterOptions
}

// NewFrameStreamSockOutput creates a FrameStreamSockOutput manaaging a
// connection to the given address.
func NewFrameStreamSockOutput(address net.Addr) (*FrameStreamSockOutput, error) {
	return &FrameStreamSockOutput{
		address:       address,
		outputChannel: make(chan []byte, outputChannelSize),
		wait:          make(chan bool),
		wopt: SocketWriterOptions{
			FlushTimeout:  5 * time.Second,
			RetryInterval: 10 * time.Second,
			Dialer: &net.Dialer{
				Timeout: 30 * time.Second,
			},
			Logger: &nullLogger{},
		},
	}, nil
}

// SetTimeout sets the write timeout for data and control messages and the
// read timeout for handshake responses on the FrameStreamSockOutput's
// connection. The default timeout is zero, for no timeout.
func (o *FrameStreamSockOutput) SetTimeout(timeout time.Duration) {
	o.wopt.Timeout = timeout
}

// SetFlushTimeout sets the maximum time data will be kept in the output
// buffer.
//
// The default flush timeout is five seconds.
func (o *FrameStreamSockOutput) SetFlushTimeout(timeout time.Duration) {
	o.wopt.FlushTimeout = timeout
}

// SetRetryInterval specifies how long the FrameStreamSockOutput will wait
// before re-establishing a failed connection. The default retry interval
// is 10 seconds.
func (o *FrameStreamSockOutput) SetRetryInterval(retry time.Duration) {
	o.wopt.RetryInterval = retry
}

// SetDialer replaces the default net.Dialer for re-establishing the
// the FrameStreamSockOutput connection. This can be used to set the
// timeout for connection establishment and enable keepalives
// new connections.
//
// FrameStreamSockOutput uses a default dialer with a 30 second
// timeout.
func (o *FrameStreamSockOutput) SetDialer(dialer *net.Dialer) {
	o.wopt.Dialer = dialer
}

// SetLogger configures FrameStreamSockOutput to log through the given
// Logger.
func (o *FrameStreamSockOutput) SetLogger(logger Logger) {
	o.wopt.Logger = logger
}

// GetOutputChannel returns the channel on which the
// FrameStreamSockOutput accepts data.
//
// GetOutputChannel satisifes the dnstap Output interface.
func (o *FrameStreamSockOutput) GetOutputChannel() chan []byte {
	return o.outputChannel
}

// RunOutputLoop reads data from the output channel and sends it over
// a connections to the FrameStreamSockOutput's address, establishing
// the connection as needed.
//
// RunOutputLoop satisifes the dnstap Output interface.
func (o *FrameStreamSockOutput) RunOutputLoop() {
	w := NewSocketWriter(o.address, &o.wopt)

	for b := range o.outputChannel {
		// w is of type *SocketWriter, whose Write implementation
		// handles all errors by retrying the connection.
		w.WriteFrame(b)
	}

	w.Close()
	close(o.wait)
	return
}

// Close shuts down the FrameStreamSockOutput's output channel and returns
// after all pending data has been flushed and the connection has been closed.
//
// Close satisifes the dnstap Output interface
func (o *FrameStreamSockOutput) Close() {
	close(o.outputChannel)
	<-o.wait
}
