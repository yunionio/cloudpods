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
	"fmt"
	"sync"

	"yunion.io/x/log"
	"yunion.io/x/ovsdb/types"
	"yunion.io/x/pkg/errors"
)

// MonitorSelect says which kinds of updates are wanted, rfc7047 4.1.5.  A nil
// member means the remote default, which is true for all four
type MonitorSelect struct {
	Initial *bool `json:"initial,omitempty"`
	Insert  *bool `json:"insert,omitempty"`
	Delete  *bool `json:"delete,omitempty"`
	Modify  *bool `json:"modify,omitempty"`
}

// MonitorRequest asks for updates of the columns of one table.  No columns
// means every column of the table
type MonitorRequest struct {
	Columns []string       `json:"columns,omitempty"`
	Select  *MonitorSelect `json:"select,omitempty"`
}

// MonitorRequests maps table names to what is wanted of them
type MonitorRequests map[string]MonitorRequest

// RowUpdate is one row's worth of an update notification, rfc7047 4.1.6
//
//	New set, Old unset      the row was inserted
//	New set, Old set        the row was modified, Old holds the previous
//	                        values of the columns that changed
//	New unset, Old set      the row was deleted
type RowUpdate struct {
	New Row `json:"new,omitempty"`
	Old Row `json:"old,omitempty"`
}

// TableUpdate maps row uuids to their updates
type TableUpdate map[string]*RowUpdate

// TableUpdates maps table names to their updates
type TableUpdates map[string]TableUpdate

// UpdateHandler is called after an update has been folded into the cache
type UpdateHandler func(tus TableUpdates)

// Monitor keeps a local copy of part of a remote database in sync
type Monitor struct {
	cli *Client
	db  string
	id  string

	cache *Cache

	mu        sync.Mutex
	handlers  []UpdateHandler
	cancelled bool
}

// Db is the database being monitored
func (m *Monitor) Db() string { return m.db }

// Id is the monitor id this monitor was registered under
func (m *Monitor) Id() string { return m.id }

// Cache is the local copy of the monitored rows
func (m *Monitor) Cache() *Cache { return m.cache }

// OnUpdate registers a callback for later updates.  It is called from the
// connection's read loop, so it should not block and must not call back into
// the client
func (m *Monitor) OnUpdate(h UpdateHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = append(m.handlers, h)
}

func (m *Monitor) apply(tus TableUpdates) {
	m.cache.Apply(tus)
	m.mu.Lock()
	handlers := make([]UpdateHandler, len(m.handlers))
	copy(handlers, m.handlers)
	m.mu.Unlock()
	for _, h := range handlers {
		h(tus)
	}
}

// Cancel stops the monitor, rfc7047 4.1.7
func (m *Monitor) Cancel(ctx context.Context) error {
	m.mu.Lock()
	if m.cancelled {
		m.mu.Unlock()
		return nil
	}
	m.cancelled = true
	m.mu.Unlock()

	m.cli.mu.Lock()
	delete(m.cli.monitors, m.id)
	m.cli.mu.Unlock()

	return m.cli.call(ctx, nil, "monitor_cancel", m.id)
}

// Monitor asks to be kept up to date with part of a database, rfc7047 4.1.5.
//
// The rows the remote sends back right away are already in the cache of the
// returned monitor by the time this returns
func (cli *Client) Monitor(ctx context.Context, db string, reqs MonitorRequests) (*Monitor, error) {
	schema, err := cli.schemaOf(db)
	if err != nil {
		return nil, err
	}
	for table := range reqs {
		if _, ok := schema.Tables[table]; !ok {
			return nil, errors.Wrapf(ErrSchema, "%s: unknown table %s", db, table)
		}
	}

	m := &Monitor{
		cli:   cli,
		db:    db,
		id:    fmt.Sprintf("%s-%d", db, cli.nextSeq()),
		cache: NewCache(schema),
	}

	// register before the call so that updates arriving on the heels of
	// the reply are not dropped
	cli.mu.Lock()
	cli.monitors[m.id] = m
	cli.mu.Unlock()

	var initial TableUpdates
	if err := cli.call(ctx, &initial, "monitor", db, m.id, reqs); err != nil {
		cli.mu.Lock()
		delete(cli.monitors, m.id)
		cli.mu.Unlock()
		return nil, err
	}
	m.apply(initial)
	return m, nil
}

// MonitorAll monitors every column of every table of a database
func (cli *Client) MonitorAll(ctx context.Context, db string) (*Monitor, error) {
	schema, err := cli.schemaOf(db)
	if err != nil {
		return nil, err
	}
	reqs := MonitorRequests{}
	for table := range schema.Tables {
		reqs[table] = MonitorRequest{}
	}
	return cli.Monitor(ctx, db, reqs)
}

// MonitorTables monitors every column of the tables given
func (cli *Client) MonitorTables(ctx context.Context, db string, itbls ...types.ITable) (*Monitor, error) {
	reqs := MonitorRequests{}
	for _, itbl := range itbls {
		reqs[itbl.OvsdbTableName()] = MonitorRequest{}
	}
	return cli.Monitor(ctx, db, reqs)
}

func (cli *Client) handleUpdate(params json.RawMessage) error {
	var args []json.RawMessage
	if err := json.Unmarshal(params, &args); err != nil {
		return errors.Wrapf(ErrProtocol, "update: %v", err)
	}
	if len(args) != 2 {
		return errors.Wrapf(ErrProtocol, "update: want 2 params, got %d", len(args))
	}
	var id string
	if err := json.Unmarshal(args[0], &id); err != nil {
		return errors.Wrapf(ErrProtocol, "update: monitor id: %v", err)
	}
	cli.mu.Lock()
	m := cli.monitors[id]
	cli.mu.Unlock()
	if m == nil {
		// a monitor we cancelled, updates in flight are expected
		log.Debugf("ovsdb: update for unknown monitor %s", id)
		return nil
	}
	var tus TableUpdates
	if err := json.Unmarshal(args[1], &tus); err != nil {
		return errors.Wrapf(ErrProtocol, "update: table updates: %v", err)
	}
	m.apply(tus)
	return nil
}
