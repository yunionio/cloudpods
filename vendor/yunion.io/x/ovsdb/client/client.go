package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"

	"yunion.io/x/log"
)

type Client struct {
	conn net.Conn
	enc  *json.Encoder
	dec  *json.Decoder

	seq uint64

	pending      map[uint64]chan *jsonRpcResponse
	pendingMutex sync.Mutex

	monitorHandlers      map[string]MonitorHandler
	monitorHandlersMutex sync.Mutex

	stopCh chan struct{}
	done   chan struct{}
}

type MonitorHandler func(tableUpdates *json.RawMessage)

func NewClient(target string) (*Client, error) {
	parts := strings.SplitN(target, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid target %s", target)
	}
	proto := parts[0]
	addr := parts[1]

	conn, err := net.Dial(proto, addr)
	if err != nil {
		return nil, err
	}

	c := &Client{
		conn:            conn,
		enc:             json.NewEncoder(conn),
		dec:             json.NewDecoder(conn),
		pending:         make(map[uint64]chan *jsonRpcResponse),
		monitorHandlers: make(map[string]MonitorHandler),
		stopCh:          make(chan struct{}),
		done:            make(chan struct{}),
	}

	go c.readLoop()

	return c, nil
}

func (c *Client) Close() error {
	close(c.stopCh)
	c.conn.Close()
	<-c.done
	return nil
}

func (c *Client) readLoop() {
	defer close(c.done)
	for {
		select {
		case <-c.stopCh:
			return
		default:
		}
		
		var rawMsg json.RawMessage
		if err := c.dec.Decode(&rawMsg); err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			log.Errorf("decode error: %v", err)
			// try to recover or exit? for now exit
			return
		}

		// Determine if it is a request, response or notification
		// OVSDB server sends responses and notifications
		
		var resp jsonRpcResponse
		if err := json.Unmarshal(rawMsg, &resp); err == nil && resp.Id != nil {
			// It's a response
			idVal, ok := resp.Id.(float64) // json numbers are floats
			if !ok {
				// check if it's string or other type?
				// we use uint64 for id, so we expect number
				// let's handle generic id
			}
			
			c.pendingMutex.Lock()
			var ch chan *jsonRpcResponse
			// Find the channel
			// Simple match for now, need robust ID handling
			// We'll assume ID is integer
			id := uint64(idVal)
			if ch, ok = c.pending[id]; ok {
				delete(c.pending, id)
			}
			c.pendingMutex.Unlock()
			
			if ch != nil {
				ch <- &resp
			}
			continue
		}
		
		var notif jsonRpcNotification
		if err := json.Unmarshal(rawMsg, &notif); err == nil && notif.Method == "update" {
			// handle update
			var params []json.RawMessage
			if err := json.Unmarshal(*notif.Params, &params); err != nil {
				log.Errorf("failed to unmarshal update params: %v", err)
				continue
			}
			if len(params) >= 2 {
				var monId string
				if err := json.Unmarshal(params[0], &monId); err != nil {
					// maybe it is not string? RFC says json-value
					// We will use string as monitor ID
				}
				
				c.monitorHandlersMutex.Lock()
				handler, ok := c.monitorHandlers[monId]
				c.monitorHandlersMutex.Unlock()
				
				if ok {
					handler(&params[1])
				}
			}
			continue
		}
	}
}

func (c *Client) Call(ctx context.Context, method string, params []interface{}) (*json.RawMessage, error) {
	id := atomic.AddUint64(&c.seq, 1)
	req := jsonRpcRequest{
		Method: method,
		Params: params,
		Id:     id,
	}

	ch := make(chan *jsonRpcResponse, 1)
	c.pendingMutex.Lock()
	c.pending[id] = ch
	c.pendingMutex.Unlock()

	defer func() {
		c.pendingMutex.Lock()
		delete(c.pending, id)
		c.pendingMutex.Unlock()
	}()

	if err := c.enc.Encode(req); err != nil {
		return nil, err
	}

	select {
	case resp := <-ch:
		if resp.Error != nil {
			return nil, fmt.Errorf("rpc error: %v", resp.Error)
		}
		return resp.Result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Client) Monitor(ctx context.Context, dbName string, monitorId string, monitorRequests interface{}, handler MonitorHandler) (*json.RawMessage, error) {
	c.monitorHandlersMutex.Lock()
	c.monitorHandlers[monitorId] = handler
	c.monitorHandlersMutex.Unlock()
	
	return c.Call(ctx, "monitor", []interface{}{dbName, monitorId, monitorRequests})
}

func (c *Client) Transact(ctx context.Context, dbName string, operations []interface{}) (*json.RawMessage, error) {
	params := make([]interface{}, 0, 1+len(operations))
	params = append(params, dbName)
	params = append(params, operations...)
	return c.Call(ctx, "transact", params)
}
