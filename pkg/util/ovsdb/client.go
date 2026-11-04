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
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"net"
	"strings"
	"sync"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/ovsdb/types"
	"yunion.io/x/pkg/errors"
)

const (
	// DefaultDialTimeout bounds the time spent connecting to the remote
	DefaultDialTimeout = 10 * time.Second
	// DefaultCallTimeout bounds the time spent waiting for a response
	DefaultCallTimeout = 30 * time.Second
	// DefaultProbeInterval is how often an idle connection is probed with
	// an echo request.  ovsdb-server probes us at 5s by default
	DefaultProbeInterval = 10 * time.Second
)

// ClientOptions tunes the connection.  A nil *ClientOptions is as good as the
// zero value, which asks for the defaults
type ClientOptions struct {
	// TLSConfig is required for ssl: addresses
	TLSConfig *tls.Config
	// DialTimeout defaults to DefaultDialTimeout
	DialTimeout time.Duration
	// CallTimeout bounds calls made without a deadline of their own.  It
	// defaults to DefaultCallTimeout
	CallTimeout time.Duration
	// ProbeInterval is how often an echo request is sent to find out
	// whether the connection is still alive.  Negative disables the probe,
	// zero asks for DefaultProbeInterval
	ProbeInterval time.Duration
}

func (opts *ClientOptions) withDefaults() ClientOptions {
	r := ClientOptions{}
	if opts != nil {
		r = *opts
	}
	if r.DialTimeout <= 0 {
		r.DialTimeout = DefaultDialTimeout
	}
	if r.CallTimeout <= 0 {
		r.CallTimeout = DefaultCallTimeout
	}
	if r.ProbeInterval == 0 {
		r.ProbeInterval = DefaultProbeInterval
	}
	return r
}

// Client speaks rfc7047 with one remote
type Client struct {
	opts ClientOptions
	rpc  *rpcConn

	mu       sync.Mutex
	schemas  map[string]*types.Schema
	monitors map[string]*Monitor
	locks    map[string]bool
	seq      int
}

// Dial connects to addr and returns a ready to use client.
//
// The address takes the form ovs-vsctl and friends accept
//
//	unix:/path/to/db.sock
//	tcp:127.0.0.1:6641
//	tcp:[::1]:6641
//	ssl:127.0.0.1:6641
//
// A bare path is taken for a unix socket, a bare host:port for tcp
func Dial(ctx context.Context, addr string, opts *ClientOptions) (*Client, error) {
	o := opts.withDefaults()
	conn, err := dialAddr(ctx, addr, &o)
	if err != nil {
		return nil, errors.Wrapf(err, "dial %s", addr)
	}
	return NewClient(conn, opts), nil
}

// NewClient runs the protocol over an already established connection
func NewClient(conn net.Conn, opts *ClientOptions) *Client {
	cli := &Client{
		opts:     opts.withDefaults(),
		schemas:  map[string]*types.Schema{},
		monitors: map[string]*Monitor{},
		locks:    map[string]bool{},
	}
	cli.rpc = newRpcConn(conn, cli.handleNotification)
	if cli.opts.ProbeInterval > 0 {
		go cli.probeLoop()
	}
	return cli
}

func dialAddr(ctx context.Context, addr string, opts *ClientOptions) (net.Conn, error) {
	network, address, wantTLS, err := parseAddr(addr)
	if err != nil {
		return nil, err
	}
	dialer := &net.Dialer{Timeout: opts.DialTimeout}
	if wantTLS {
		if opts.TLSConfig == nil {
			return nil, errors.Wrapf(ErrAddress, "%s: tls config is required", addr)
		}
		return tls.DialWithDialer(dialer, network, address, opts.TLSConfig)
	}
	return dialer.DialContext(ctx, network, address)
}

func parseAddr(addr string) (network string, address string, wantTLS bool, err error) {
	switch {
	case strings.HasPrefix(addr, "unix:"):
		return "unix", addr[len("unix:"):], false, nil
	case strings.HasPrefix(addr, "punix:"):
		return "unix", addr[len("punix:"):], false, nil
	case strings.HasPrefix(addr, "tcp:"):
		return "tcp", addr[len("tcp:"):], false, nil
	case strings.HasPrefix(addr, "ptcp:"):
		return "tcp", addr[len("ptcp:"):], false, nil
	case strings.HasPrefix(addr, "ssl:"):
		return "tcp", addr[len("ssl:"):], true, nil
	case strings.HasPrefix(addr, "pssl:"):
		return "tcp", addr[len("pssl:"):], true, nil
	case strings.HasPrefix(addr, "/"), strings.HasPrefix(addr, "@"):
		return "unix", addr, false, nil
	case strings.Contains(addr, ":"):
		return "tcp", addr, false, nil
	}
	return "", "", false, errors.Wrapf(ErrAddress, "%q", addr)
}

// Close drops the connection.  Outstanding calls fail with ErrClosed
func (cli *Client) Close() error {
	return cli.rpc.Close()
}

// Done is closed when the connection is gone, whether because Close was
// called or because the remote went away
func (cli *Client) Done() <-chan struct{} {
	return cli.rpc.Done()
}

// Err tells why the connection is gone
func (cli *Client) Err() error {
	return cli.rpc.Err()
}

func (cli *Client) probeLoop() {
	ticker := time.NewTicker(cli.opts.ProbeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-cli.rpc.Done():
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), cli.opts.ProbeInterval)
			err := cli.Echo(ctx)
			cancel()
			if err != nil {
				log.Errorf("ovsdb: probe failed, closing connection: %v", err)
				cli.rpc.shutdown(errors.Wrap(err, "probe"))
				return
			}
		}
	}
}

func (cli *Client) call(ctx context.Context, result interface{}, method string, params ...interface{}) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cli.opts.CallTimeout)
		defer cancel()
	}
	raw, err := cli.rpc.Call(ctx, method, params)
	if err != nil {
		return err
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(raw, result); err != nil {
		return errors.Wrapf(ErrProtocol, "%s: decode result: %v", method, err)
	}
	return nil
}

// Echo checks that the connection is still good, rfc7047 4.1.11
func (cli *Client) Echo(ctx context.Context) error {
	var result []string
	if err := cli.call(ctx, &result, "echo"); err != nil {
		return err
	}
	return nil
}

// ListDbs returns the databases the remote serves, rfc7047 4.1.1
func (cli *Client) ListDbs(ctx context.Context) ([]string, error) {
	var result []string
	if err := cli.call(ctx, &result, "list_dbs"); err != nil {
		return nil, err
	}
	return result, nil
}

// GetSchema fetches the schema of a database and remembers it, rfc7047 4.1.2.
//
// The schema is what tells this package how to encode the columns of a row,
// so it has to be fetched before rows can be sent or received
func (cli *Client) GetSchema(ctx context.Context, db string) (*types.Schema, error) {
	var raw json.RawMessage
	if err := cli.call(ctx, &raw, "get_schema", db); err != nil {
		return nil, err
	}
	schema, err := types.ParseSchema(bytes.NewReader(raw))
	if err != nil {
		return nil, errors.Wrapf(err, "parse schema of %s", db)
	}
	cli.mu.Lock()
	cli.schemas[db] = schema
	cli.mu.Unlock()
	return schema, nil
}

// Schema returns the schema fetched earlier by GetSchema
func (cli *Client) Schema(db string) *types.Schema {
	cli.mu.Lock()
	defer cli.mu.Unlock()
	return cli.schemas[db]
}

func (cli *Client) schemaOf(db string) (*types.Schema, error) {
	if schema := cli.Schema(db); schema != nil {
		return schema, nil
	}
	return nil, errors.Wrapf(ErrSchema, "%s: call GetSchema first", db)
}

// Transact runs the operations as a single transaction, rfc7047 4.1.3.
//
// The returned results are the ones the remote sent, they are not checked.
// Use CheckOperationResults, or Txn.Commit which does it for you
func (cli *Client) Transact(ctx context.Context, db string, ops []Operation) ([]OperationResult, error) {
	params := make([]interface{}, 0, len(ops)+1)
	params = append(params, db)
	for i := range ops {
		params = append(params, ops[i])
	}
	var results []OperationResult
	if err := cli.call(ctx, &results, "transact", params...); err != nil {
		return nil, err
	}
	return results, nil
}

// Cancel asks the remote to give up on a transaction, rfc7047 4.1.4.  The id
// is the one of the transact request to cancel, which this package does not
// hand out, so this is here for completeness only
func (cli *Client) Cancel(ctx context.Context, id interface{}) error {
	return cli.notify("cancel", id)
}

// Lock asks for a lock, rfc7047 4.1.8.  It returns whether the lock was
// granted right away.  Locks acquired later are reported by Locked
func (cli *Client) Lock(ctx context.Context, id string) (bool, error) {
	return cli.lockCall(ctx, "lock", id)
}

// Steal takes a lock away from whoever holds it, rfc7047 4.1.8
func (cli *Client) Steal(ctx context.Context, id string) (bool, error) {
	return cli.lockCall(ctx, "steal", id)
}

func (cli *Client) lockCall(ctx context.Context, method, id string) (bool, error) {
	var result struct {
		Locked bool `json:"locked"`
	}
	if err := cli.call(ctx, &result, method, id); err != nil {
		return false, err
	}
	cli.mu.Lock()
	cli.locks[id] = result.Locked
	cli.mu.Unlock()
	return result.Locked, nil
}

// Unlock gives a lock back, rfc7047 4.1.8
func (cli *Client) Unlock(ctx context.Context, id string) error {
	if err := cli.call(ctx, nil, "unlock", id); err != nil {
		return err
	}
	cli.mu.Lock()
	delete(cli.locks, id)
	cli.mu.Unlock()
	return nil
}

// Locked tells whether a lock asked for earlier is held now.  The remote
// notifies us with "locked" and "stolen", rfc7047 4.1.9, 4.1.10
func (cli *Client) Locked(id string) bool {
	cli.mu.Lock()
	defer cli.mu.Unlock()
	return cli.locks[id]
}

func (cli *Client) notify(method string, params ...interface{}) error {
	if params == nil {
		params = []interface{}{}
	}
	return cli.rpc.send(map[string]interface{}{
		"method": method,
		"params": params,
		"id":     nil,
	})
}

// handleNotification dispatches the requests the remote makes of us.  Echo is
// answered by rpcConn itself
func (cli *Client) handleNotification(method string, params json.RawMessage) (interface{}, error) {
	switch method {
	case "update":
		return nil, cli.handleUpdate(params)
	case "locked", "stolen":
		var args []string
		if err := json.Unmarshal(params, &args); err != nil {
			return nil, errors.Wrapf(ErrProtocol, "%s: %v", method, err)
		}
		if len(args) != 1 {
			return nil, errors.Wrapf(ErrProtocol, "%s: want 1 param, got %d", method, len(args))
		}
		cli.mu.Lock()
		cli.locks[args[0]] = method == "locked"
		cli.mu.Unlock()
		return nil, nil
	}
	return nil, errors.Wrapf(ErrProtocol, "unexpected method %q", method)
}

// nextSeq hands out the ids this package makes up for monitors
func (cli *Client) nextSeq() int {
	cli.mu.Lock()
	defer cli.mu.Unlock()
	cli.seq++
	return cli.seq
}
