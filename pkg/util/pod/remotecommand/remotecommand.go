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

package remotecommand

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/httpstream"
	"yunion.io/x/onecloud/pkg/util/pod/remotecommand/spdy"
)

// StreamOptions holds information pertaining to the current streaming session:
// input/output streams, if the client is requesting a TTY, and a terminal size queue to
// support terminal resizing.
type StreamOptions struct {
	Stdin             io.Reader
	Stdout            io.Writer
	Stderr            io.Writer
	Tty               bool
	TerminalSizeQueue TerminalSizeQueue
	Header            http.Header
}

// Executor is an interface for transporting shell-style streams.
type Executor interface {
	// Stream initiates the transport of the standard shell streams. It will transport any
	// non-nil stream to a remote system, and return an error if a problem occurs.
	Stream(options StreamOptions) error
}

type streamCreator interface {
	CreateStream(headers http.Header) (httpstream.Stream, error)
}

type streamProtocolHandler interface {
	stream(conn streamCreator) error
}

// streamExecutor handles transporting standard shell streams over a httpstream connection.
type streamExecutor struct {
	upgrader  spdy.Upgrader
	transport http.RoundTripper

	method    string
	url       *url.URL
	protocols []string
}

// NewSPDYExecutor connects to the provided server and upgrades the connection to
// multiplexed bidirectional streams.
func NewSPDYExecutor(method string, url *url.URL) (Executor, error) {
	wrapper, upgradeRoundTripper, err := spdy.RoundTripperFor()
	if err != nil {
		return nil, err
	}
	return NewSPDYExecutorForTransports(wrapper, upgradeRoundTripper, method, url)
}

// NewSPDYExecutorForTransports connects to the provided server using the given transport,
// upgrades the response using the given upgrader to multiplexed bidirectional streams.
func NewSPDYExecutorForTransports(tranport http.RoundTripper, upgrader spdy.Upgrader, method string, url *url.URL) (Executor, error) {
	return NewSPDYExecutorForProtocols(
		tranport, upgrader, method, url,
		StreamProtocolV4Name,
		StreamProtocolV3Name,
		StreamProtocolV2Name,
		StreamProtocolV1Name)
}

// NewSPDYExecutorForProtocols connects to the provided server and upgrades the connection to
// multiplexed bidirectional streams using only the provided protocols. Exposed for testing, most
// callers should use NewSPDYExecutor or NewSPDYExecutorForTransports.
func NewSPDYExecutorForProtocols(transport http.RoundTripper, upgrader spdy.Upgrader, method string, url *url.URL, protocols ...string) (Executor, error) {
	return &streamExecutor{
		upgrader:  upgrader,
		transport: transport,
		method:    method,
		url:       url,
		protocols: protocols,
	}, nil
}

// Stream opens a protocol streamer to the server and streams until a client closes
// the connection or the server disconnects.
func (e *streamExecutor) Stream(options StreamOptions) error {
	req, err := http.NewRequest(e.method, e.url.String(), nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	if options.Header != nil {
		req.Header = options.Header
	}
	conn, protocol, err := spdy.Negotiate(
		e.upgrader,
		&http.Client{Transport: e.transport},
		req,
		e.protocols...,
	)
	if err != nil {
		return err
	}
	defer conn.Close()

	var streamer streamProtocolHandler
	switch protocol {
	case StreamProtocolV4Name:
		streamer = newStreamProtocolV4(options)
	case StreamProtocolV3Name:
		streamer = newStreamProtocolV3(options)
	case StreamProtocolV2Name:
		streamer = newStreamProtocolV2(options)
	case "":
		log.Infof("The server did not negotiate a streaming protocol version. Falling back to %s", StreamProtocolV1Name)
		fallthrough
	case StreamProtocolV1Name:
		streamer = newStreamProtocolV1(options)
	}

	return streamer.stream(conn)
}
