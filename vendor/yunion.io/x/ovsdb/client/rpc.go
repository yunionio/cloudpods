package client

import (
	"encoding/json"
)

type jsonRpcRequest struct {
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
	Id     interface{}   `json:"id"`
}

type jsonRpcResponse struct {
	Result *json.RawMessage `json:"result"`
	Error  interface{}      `json:"error"`
	Id     interface{}      `json:"id"`
}

type jsonRpcNotification struct {
	Method string           `json:"method"`
	Params *json.RawMessage `json:"params"`
	Id     interface{}      `json:"id"` // should be null
}

type Operation struct {
	Op    string `json:"op"`
	Table string `json:"table,omitempty"`
	// Common fields
	Where   []interface{} `json:"where,omitempty"`
	Columns []string      `json:"columns,omitempty"`
	// Insert/Update
	Row      map[string]interface{} `json:"row,omitempty"`
	UuidName string                 `json:"uuid-name,omitempty"`
	// Mutate
	Mutations []interface{} `json:"mutations,omitempty"`
	// Wait
	Timeout *int64         `json:"timeout,omitempty"`
	Until   string         `json:"until,omitempty"`
	Rows    []interface{}  `json:"rows,omitempty"`
	// Comment
	Comment string `json:"comment,omitempty"`
}

// Helpers for conditions
func NewCondition(column string, function string, value interface{}) []interface{} {
	return []interface{}{column, function, value}
}

// Helpers for mutations
func NewMutation(column string, mutator string, value interface{}) []interface{} {
	return []interface{}{column, mutator, value}
}
