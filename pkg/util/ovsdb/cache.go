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
	"sort"
	"sync"

	"yunion.io/x/ovsdb/types"
	"yunion.io/x/pkg/errors"
)

// Cache is a local copy of the rows a monitor covers.  It is safe for
// concurrent use
type Cache struct {
	schema *types.Schema

	mu     sync.RWMutex
	tables map[string]map[string]Row
}

// NewCache returns an empty cache for a database
func NewCache(schema *types.Schema) *Cache {
	return &Cache{
		schema: schema,
		tables: map[string]map[string]Row{},
	}
}

// Schema is the schema of the cached database
func (c *Cache) Schema() *types.Schema {
	return c.schema
}

// Apply folds an update notification into the cache, rfc7047 4.1.6
func (c *Cache) Apply(tus TableUpdates) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for table, tu := range tus {
		for uuid, ru := range tu {
			if ru == nil {
				continue
			}
			switch {
			case ru.New != nil:
				c.insertOrModify(table, uuid, ru.New)
			case ru.Old != nil:
				c.remove(table, uuid)
			}
		}
	}
}

// insertOrModify is called with the lock held.  An update carries only the
// columns that changed, so it is merged into what is already known
func (c *Cache) insertOrModify(table, uuid string, newRow Row) {
	rows, ok := c.tables[table]
	if !ok {
		rows = map[string]Row{}
		c.tables[table] = rows
	}
	row, ok := rows[uuid]
	if !ok {
		row = Row{}
		rows[uuid] = row
	}
	for col, val := range newRow {
		row[col] = val
	}
	// monitor updates do not carry _uuid, it is the key of the update
	row[ColUuid] = RawUuid(uuid)
}

// remove is called with the lock held
func (c *Cache) remove(table, uuid string) {
	rows, ok := c.tables[table]
	if !ok {
		return
	}
	delete(rows, uuid)
	if len(rows) == 0 {
		delete(c.tables, table)
	}
}

// Row returns a copy of one cached row
func (c *Cache) Row(table, uuid string) (Row, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	rows, ok := c.tables[table]
	if !ok {
		return nil, false
	}
	row, ok := rows[uuid]
	if !ok {
		return nil, false
	}
	return row.Copy(), true
}

// Rows returns copies of the cached rows of a table, ordered by uuid so that
// walking them is reproducible
func (c *Cache) Rows(table string) []Row {
	c.mu.RLock()
	defer c.mu.RUnlock()
	rows := c.tables[table]
	uuids := make([]string, 0, len(rows))
	for uuid := range rows {
		uuids = append(uuids, uuid)
	}
	sort.Strings(uuids)
	r := make([]Row, len(uuids))
	for i, uuid := range uuids {
		r[i] = rows[uuid].Copy()
	}
	return r
}

// Len is the number of rows cached for a table
func (c *Cache) Len(table string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.tables[table])
}

// Tables lists the tables that have rows cached
func (c *Cache) Tables() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	r := make([]string, 0, len(c.tables))
	for table := range c.tables {
		r = append(r, table)
	}
	sort.Strings(r)
	return r
}

// FillTable replaces the content of a generated table type with the rows
// cached for it
func (c *Cache) FillTable(itbl types.ITable) error {
	rows := c.Rows(itbl.OvsdbTableName())
	if err := FillITable(itbl, rows); err != nil {
		return errors.Wrapf(err, "fill %s", itbl.OvsdbTableName())
	}
	return nil
}

// FillTables is FillTable for several tables, as in
//
//	db := ovn_nb.OVNNorthbound{}
//	cache.FillTables(&db.LogicalSwitch, &db.LogicalSwitchPort)
func (c *Cache) FillTables(itbls ...types.ITable) error {
	for _, itbl := range itbls {
		if err := c.FillTable(itbl); err != nil {
			return err
		}
	}
	return nil
}
