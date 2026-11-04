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
	"context"
	"encoding/json"
	"io"
	"net"
	"strconv"
	"sync"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

// rpcMessage is a JSON-RPC 1.0 message.  Requests, responses and
// notifications are told apart by which members are present, see RFC7047 4.
//
// Note that a JSON-RPC 1.0 response always carries all of result, error and
// id, with the unused ones set to null
type rpcMessage struct {
	Method *string         `json:"method"`
	Params json.RawMessage `json:"params"`
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"`
	Id     json.RawMessage `json:"id"`
}

func (msg *rpcMessage) isRequest() bool {
	return msg.Method != nil
}

func (msg *rpcMessage) isNotification() bool {
	return msg.Method != nil && isNullJson(msg.Id)
}

func isNullJson(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return true
	}
	return string(raw) == "null"
}

// idKey normalizes a JSON-RPC id for use as a map key.  Remotes are expected
// to echo the id back verbatim, but the encoding of it is theirs to pick
func idKey(raw json.RawMessage) string {
	var val interface{}
	if err := json.Unmarshal(raw, &val); err != nil {
		return string(raw)
	}
	switch v := val.(type) {
	case float64:
		return "n" + strconv.FormatInt(int64(v), 10)
	case string:
		return "s" + v
	default:
		return string(raw)
	}
}

// notifyFunc is called for requests and notifications the remote sends us.
// Returning a nil error along with a nil result for a request makes the
// request be answered with a null result
type notifyFunc func(method string, params json.RawMessage) (interface{}, error)

// rpcConn multiplexes JSON-RPC 1.0 calls over a single connection
type rpcConn struct {
	conn net.Conn
	enc  *json.Encoder
	dec  *json.Decoder

	wmu sync.Mutex

	mu      sync.Mutex
	seq     uint64
	pending map[string]chan *rpcMessage
	err     error

	notify notifyFunc

	// replies to requests the remote makes of us are written by a
	// goroutine of their own.  Writing them from the read loop would stop
	// us from reading while the remote is not reading either, which on a
	// connection with little buffering is a deadlock
	replies chan interface{}

	done      chan struct{}
	closeOnce sync.Once
}

func newRpcConn(conn net.Conn, notify notifyFunc) *rpcConn {
	c := &rpcConn{
		conn:    conn,
		enc:     json.NewEncoder(conn),
		dec:     json.NewDecoder(conn),
		pending: map[string]chan *rpcMessage{},
		notify:  notify,
		replies: make(chan interface{}, 16),
		done:    make(chan struct{}),
	}
	go c.readLoop()
	go c.replyLoop()
	return c
}

// replyLoop answers the requests the remote makes of us
func (c *rpcConn) replyLoop() {
	for {
		select {
		case <-c.done:
			return
		case reply := <-c.replies:
			if err := c.send(reply); err != nil {
				log.Warningf("ovsdb: reply: %v", err)
				return
			}
		}
	}
}

// Done is closed when the connection is no longer usable
func (c *rpcConn) Done() <-chan struct{} {
	return c.done
}

// Err returns the error that brought the connection down, if any
func (c *rpcConn) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

func (c *rpcConn) Close() error {
	c.shutdown(ErrClosed)
	return nil
}

func (c *rpcConn) shutdown(err error) {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		if c.err == nil {
			c.err = err
		}
		// pending calls are woken up by done being closed.  The channels
		// themselves are left alone, a response racing with the shutdown
		// still has somewhere to go
		c.pending = map[string]chan *rpcMessage{}
		c.mu.Unlock()

		c.conn.Close()
		close(c.done)
	})
}

func (c *rpcConn) send(msg interface{}) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	if err := c.enc.Encode(msg); err != nil {
		c.shutdown(errors.Wrap(err, "write"))
		return errors.Wrap(err, "send")
	}
	return nil
}

func (c *rpcConn) readLoop() {
	for {
		msg := &rpcMessage{}
		if err := c.dec.Decode(msg); err != nil {
			if err == io.EOF {
				err = ErrClosed
			}
			c.shutdown(errors.Wrap(err, "read"))
			return
		}
		if msg.isRequest() {
			c.handleRequest(msg)
			continue
		}
		c.handleResponse(msg)
	}
}

func (c *rpcConn) handleRequest(msg *rpcMessage) {
	var (
		result interface{}
		err    error
	)
	switch *msg.Method {
	case "echo":
		// rfc7047 4.1.11, the reply must carry the very same params
		if len(msg.Params) > 0 {
			result = json.RawMessage(msg.Params)
		} else {
			result = []interface{}{}
		}
	default:
		if c.notify != nil {
			result, err = c.notify(*msg.Method, msg.Params)
		} else {
			err = errors.Wrapf(ErrProtocol, "unexpected method %q", *msg.Method)
		}
	}
	if msg.isNotification() {
		if err != nil {
			log.Warningf("ovsdb: notification %s: %v", *msg.Method, err)
		}
		return
	}
	reply := map[string]interface{}{
		"id":     json.RawMessage(msg.Id),
		"result": result,
		"error":  nil,
	}
	if err != nil {
		reply["result"] = nil
		reply["error"] = err.Error()
	}
	select {
	case c.replies <- reply:
	case <-c.done:
	default:
		log.Warningf("ovsdb: reply to %s dropped, the remote is not keeping up", *msg.Method)
	}
}

func (c *rpcConn) handleResponse(msg *rpcMessage) {
	if isNullJson(msg.Id) {
		log.Warningf("ovsdb: response without id: %s", string(msg.Result))
		return
	}
	key := idKey(msg.Id)
	c.mu.Lock()
	ch, ok := c.pending[key]
	if ok {
		delete(c.pending, key)
	}
	c.mu.Unlock()
	if !ok {
		log.Warningf("ovsdb: response to unknown request %s", key)
		return
	}
	ch <- msg
}

// Call sends a request and waits for its response.  Errors reported by the
// remote are wrapped in ErrRemote
func (c *rpcConn) Call(ctx context.Context, method string, params []interface{}) (json.RawMessage, error) {
	if params == nil {
		params = []interface{}{}
	}

	c.mu.Lock()
	if c.err != nil {
		err := c.err
		c.mu.Unlock()
		return nil, errors.Wrap(err, method)
	}
	c.seq++
	id := c.seq
	key := "n" + strconv.FormatUint(id, 10)
	ch := make(chan *rpcMessage, 1)
	c.pending[key] = ch
	c.mu.Unlock()

	req := map[string]interface{}{
		"method": method,
		"params": params,
		"id":     id,
	}
	if err := c.send(req); err != nil {
		c.mu.Lock()
		delete(c.pending, key)
		c.mu.Unlock()
		return nil, errors.Wrap(err, method)
	}

	select {
	case msg := <-ch:
		if !isNullJson(msg.Error) {
			return nil, errors.Wrapf(ErrRemote, "%s: %s", method, string(msg.Error))
		}
		return msg.Result, nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, key)
		c.mu.Unlock()
		return nil, errors.Wrap(ctx.Err(), method)
	case <-c.done:
		return nil, errors.Wrap(c.Err(), method)
	}
}
