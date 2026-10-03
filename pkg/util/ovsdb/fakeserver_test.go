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

package ovsdb

import (
	"encoding/json"
	"net"
	"sync"
	"testing"
	"time"
)

// testSchema is a cut down OVN_Northbound schema.  It holds the columns the
// tests need of tables the generated code of yunion.io/x/ovsdb knows, so that
// rows can be round tripped through the real row types
const testSchema = `{
  "name": "OVN_Northbound",
  "version": "5.13.0",
  "cksum": "0 0",
  "tables": {
    "Logical_Switch": {
      "columns": {
        "name": {"type": "string"},
        "ports": {"type": {"key": {"type": "uuid", "refTable": "Logical_Switch_Port", "refType": "strong"},
                           "min": 0, "max": "unlimited"}},
        "other_config": {"type": {"key": "string", "value": "string", "min": 0, "max": "unlimited"}},
        "external_ids": {"type": {"key": "string", "value": "string", "min": 0, "max": "unlimited"}}
      },
      "isRoot": true,
      "indexes": [["name"]]
    },
    "Logical_Switch_Port": {
      "columns": {
        "name": {"type": "string"},
        "type": {"type": "string"},
        "addresses": {"type": {"key": "string", "min": 0, "max": "unlimited"}},
        "port_security": {"type": {"key": "string", "min": 0, "max": "unlimited"}},
        "parent_name": {"type": {"key": "string", "min": 0, "max": 1}},
        "tag": {"type": {"key": {"type": "integer", "minInteger": 1, "maxInteger": 4095}, "min": 0, "max": 1}},
        "up": {"type": {"key": "boolean", "min": 0, "max": 1}},
        "enabled": {"type": {"key": "boolean", "min": 0, "max": 1}},
        "dhcpv4_options": {"type": {"key": {"type": "uuid", "refTable": "DHCP_Options", "refType": "weak"},
                                    "min": 0, "max": 1}},
        "options": {"type": {"key": "string", "value": "string", "min": 0, "max": "unlimited"}},
        "external_ids": {"type": {"key": "string", "value": "string", "min": 0, "max": "unlimited"}}
      },
      "indexes": [["name"]],
      "isRoot": false
    },
    "DHCP_Options": {
      "columns": {
        "cidr": {"type": "string"},
        "options": {"type": {"key": "string", "value": "string", "min": 0, "max": "unlimited"}},
        "external_ids": {"type": {"key": "string", "value": "string", "min": 0, "max": "unlimited"}}
      },
      "isRoot": true
    }
  }
}`

// handlerFunc answers one method of the fake remote
type handlerFunc func(params []json.RawMessage) (interface{}, error)

// fakeServer is a rfc7047 remote good enough to drive the client through its
// paces.  It does not implement a database, tests say what each method returns
type fakeServer struct {
	t    *testing.T
	conn net.Conn
	enc  *json.Encoder
	dec  *json.Decoder

	mu        sync.Mutex
	handlers  map[string]handlerFunc
	requests  []fakeRequest
	writeLock sync.Mutex

	// replies holds the answers the client sends to the requests we make
	// of it
	replies chan rpcMessage

	done chan struct{}
}

type fakeRequest struct {
	Method string
	Params []json.RawMessage
}

// newFakeServer returns a client wired to a fake remote over an in memory
// connection
func newFakeServer(t *testing.T, opts *ClientOptions) (*Client, *fakeServer) {
	t.Helper()
	cconn, sconn := net.Pipe()
	srv := &fakeServer{
		t:        t,
		conn:     sconn,
		enc:      json.NewEncoder(sconn),
		dec:      json.NewDecoder(sconn),
		handlers: map[string]handlerFunc{},
		replies:  make(chan rpcMessage, 16),
		done:     make(chan struct{}),
	}
	srv.Handle("echo", func(params []json.RawMessage) (interface{}, error) {
		return []string{}, nil
	})
	srv.Handle("get_schema", func(params []json.RawMessage) (interface{}, error) {
		return json.RawMessage(testSchema), nil
	})
	go srv.serve()

	if opts == nil {
		opts = &ClientOptions{}
	}
	o := *opts
	if o.ProbeInterval == 0 {
		// the tests drive the clock, no background probing
		o.ProbeInterval = -1
	}
	cli := NewClient(cconn, &o)
	t.Cleanup(func() {
		cli.Close()
		srv.Close()
	})
	return cli, srv
}

// Handle registers the answer for a method
func (srv *fakeServer) Handle(method string, h handlerFunc) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	srv.handlers[method] = h
}

// Requests returns the requests seen so far
func (srv *fakeServer) Requests() []fakeRequest {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	r := make([]fakeRequest, len(srv.requests))
	copy(r, srv.requests)
	return r
}

// LastRequest returns the most recent request for a method
func (srv *fakeServer) LastRequest(method string) *fakeRequest {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	for i := len(srv.requests) - 1; i >= 0; i-- {
		if srv.requests[i].Method == method {
			return &srv.requests[i]
		}
	}
	return nil
}

func (srv *fakeServer) Close() {
	srv.conn.Close()
}

func (srv *fakeServer) serve() {
	defer close(srv.done)
	for {
		msg := &rpcMessage{}
		if err := srv.dec.Decode(msg); err != nil {
			return
		}
		if msg.Method == nil {
			select {
			case srv.replies <- *msg:
			default:
			}
			continue
		}
		var params []json.RawMessage
		if len(msg.Params) > 0 {
			if err := json.Unmarshal(msg.Params, &params); err != nil {
				return
			}
		}
		srv.mu.Lock()
		srv.requests = append(srv.requests, fakeRequest{Method: *msg.Method, Params: params})
		h := srv.handlers[*msg.Method]
		srv.mu.Unlock()

		if isNullJson(msg.Id) {
			continue
		}
		reply := map[string]interface{}{
			"id":     json.RawMessage(msg.Id),
			"result": nil,
			"error":  nil,
		}
		if h == nil {
			reply["error"] = "unknown method"
		} else {
			result, err := h(params)
			if err != nil {
				reply["error"] = err.Error()
			} else {
				reply["result"] = result
			}
		}
		if err := srv.write(reply); err != nil {
			return
		}
	}
}

func (srv *fakeServer) write(v interface{}) error {
	srv.writeLock.Lock()
	defer srv.writeLock.Unlock()
	return srv.enc.Encode(v)
}

// Notify sends a notification to the client
func (srv *fakeServer) Notify(method string, params ...interface{}) error {
	return srv.write(map[string]interface{}{
		"method": method,
		"params": params,
		"id":     nil,
	})
}

// WaitReply returns the next reply the client sent us
func (srv *fakeServer) WaitReply(t *testing.T) rpcMessage {
	t.Helper()
	select {
	case msg := <-srv.replies:
		return msg
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for a reply from the client")
		return rpcMessage{}
	}
}

// Request sends a request to the client
func (srv *fakeServer) Request(id int, method string, params ...interface{}) error {
	return srv.write(map[string]interface{}{
		"method": method,
		"params": params,
		"id":     id,
	})
}
