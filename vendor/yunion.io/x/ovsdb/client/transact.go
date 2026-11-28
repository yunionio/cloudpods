package client

import (
	"context"
	"encoding/json"
	"reflect"

	"yunion.io/x/ovsdb/types"
)

func (c *Client) TransactOps(ctx context.Context, dbName string, ops ...Operation) ([]OperationResult, error) {
	opInterfaces := make([]interface{}, len(ops))
	for i, op := range ops {
		opInterfaces[i] = op
	}
	
	rawResult, err := c.Transact(ctx, dbName, opInterfaces)
	if err != nil {
		return nil, err
	}
	
	// Parse result
	var results []OperationResult
	if err := json.Unmarshal(*rawResult, &results); err != nil {
		return nil, err
	}
	
	return results, nil
}

type OperationResult struct {
	Count   int           `json:"count,omitempty"`
	Uuid    []interface{} `json:"uuid,omitempty"` // ["uuid", "new-uuid"]
	Rows    []interface{} `json:"rows,omitempty"`
	Error   string        `json:"error,omitempty"`
	Details string        `json:"details,omitempty"`
}

func NewConditionUuid(uuid string) []interface{} {
	return []interface{}{"_uuid", "==", []string{"uuid", uuid}}
}

// ToOvsdbJSONRow converts an IRow to a map suitable for OVSDB JSON-RPC
func ToOvsdbJSONRow(row types.IRow) map[string]interface{} {
	ret := make(map[string]interface{})
	
	val := reflect.ValueOf(row).Elem()
	typ := val.Type()
	
	tableName := row.OvsdbTableName()
	
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		
		tag := fieldType.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		
		// Handle "name,omitempty"
		// tag is `json:"name"`
		colName := tag
		if idx := reflect.ValueOf(tag).String(); len(idx) > 0 {
			// manually parse?
			// standard lib does it
		}
		// Just take until comma
		commaIdx := 0
		for j := 0; j < len(tag); j++ {
			if tag[j] == ',' {
				commaIdx = j
				break
			}
		}
		if commaIdx > 0 {
			colName = tag[:commaIdx]
		}
		
		if colName == "_uuid" || colName == "_version" {
			continue
		}
		
		// Check if field should be skipped
		if shouldSkip(field) {
			continue
		}
		
		// Get schema type
		schemaType := GetSchemaType(tableName, colName)
		
		jsonVal := formatValue(field, schemaType)
		ret[colName] = jsonVal
	}
	
	return ret
}

func shouldSkip(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Map:
		return v.IsNil() || v.Len() == 0
	}
	// Keep primitives (Int, String, Bool, etc.) even if zero value
	return false
}

func formatValue(v reflect.Value, schemaType string) interface{} {
	switch schemaType {
	case "uuid":
		// Value should be string or *string
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		return []string{"uuid", v.String()}
	case "setUuid":
		// Value should be []string
		// OVSDB set format: ["set", [["uuid", "u1"], ["uuid", "u2"]]]
		// Wait, RFC says set is ["set", [e1, e2, ...]]
		// e1 is ["uuid", "u1"]
		
		sl := v
		if sl.Len() == 0 {
			return []interface{}{"set", []interface{}{}}
		}
		elems := make([]interface{}, 0, sl.Len())
		for i := 0; i < sl.Len(); i++ {
			uuidStr := sl.Index(i).String()
			elems = append(elems, []string{"uuid", uuidStr})
		}
		return []interface{}{"set", elems}
	case "setString":
		sl := v
		if sl.Len() == 0 {
			return []interface{}{"set", []interface{}{}}
		}
		// If length is 1, can we send just the string?
		// RFC 7047: "For a column with min 0 or 1 and max 1, <value> is the value...
		// For a column with max > 1, <value> is ["set", [e1, ...]]"
		// We assumed setString corresponds to max=unlimited.
		// Let's adhere to set format always for set types.
		elems := make([]interface{}, 0, sl.Len())
		for i := 0; i < sl.Len(); i++ {
			elems = append(elems, sl.Index(i).Interface())
		}
		return []interface{}{"set", elems}
	case "mapStringString", "mapStringInteger", "mapStringIntegerMap", "mapStringIntegerReal", "mapStringBoolean":
		// Map format: ["map", [[k1, v1], [k2, v2]]]
		m := v
		if m.Len() == 0 {
			return []interface{}{"map", []interface{}{}}
		}
		elems := make([]interface{}, 0, m.Len())
		iter := m.MapRange()
		for iter.Next() {
			k := iter.Key().Interface()
			val := iter.Value().Interface()
			elems = append(elems, []interface{}{k, val})
		}
		return []interface{}{"map", elems}
	default:
		// atomic types
		if v.Kind() == reflect.Ptr {
			return v.Elem().Interface()
		}
		return v.Interface()
	}
}

func OvsdbCreateOp(row types.IRow, uuidName string) Operation {
	return Operation{
		Op:       "insert",
		Table:    row.OvsdbTableName(),
		Row:      ToOvsdbJSONRow(row),
		UuidName: uuidName,
	}
}

// For delete, we usually delete by UUID or condition
func OvsdbDeleteOp(table string, conditions []interface{}) Operation {
	return Operation{
		Op:    "delete",
		Table: table,
		Where: conditions,
	}
}

// For update, we update by condition (usually UUID)
func OvsdbUpdateOp(table string, conditions []interface{}, row types.IRow) Operation {
	return Operation{
		Op:    "update",
		Table: table,
		Where: conditions,
		Row:   ToOvsdbJSONRow(row),
	}
}

// Mutate is useful for adding/removing from set/map
func OvsdbMutateOp(table string, conditions []interface{}, mutations []interface{}) Operation {
	return Operation{
		Op:        "mutate",
		Table:     table,
		Where:     conditions,
		Mutations: mutations,
	}
}

// Wait
func OvsdbWaitOp(table string, conditions []interface{}, timeout int64) Operation {
	return Operation{
		Op:      "wait",
		Table:   table,
		Timeout: &timeout,
		Where:   conditions,
		Until:   "==",
		Rows:    []interface{}{}, // empty rows means wait until empty? No.
		// wait until table has rows matching where?
		// RFC: "wait": "waits until <condition> is true"
		// "rows" member of result? No.
		// We need "columns", "rows", "until".
		// Default until is "==".
		// columns default all.
		// rows default empty?
		// "If 'rows' is omitted, it defaults to an empty array."
		// "waits until the query ... would return the result specified by 'rows'".
		// So if we want to wait until a row exists, we should specify it?
		// Usually we wait until specific condition matches nothing (deleted) or something (created).
	}
}
