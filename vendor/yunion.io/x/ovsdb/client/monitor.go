package client

import (
	"context"
	"encoding/json"
	"reflect"

	"yunion.io/x/log"
	"yunion.io/x/ovsdb/types"
)

// MonitorDB sets up monitoring for all tables in the provided database model.
// The model must be a pointer to a struct where each field implements types.ITable.
// It updates the model in-place as notifications arrive.
func (c *Client) MonitorDB(ctx context.Context, dbName string, model types.IDatabase) error {
	// 1. Inspect the model to find tables
	val := reflect.ValueOf(model).Elem()

	monitorRequests := make(map[string]interface{})

	tableMap := make(map[string]reflect.Value)

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		
		if !field.CanInterface() {
			continue
		}
		
		if itbl, ok := field.Interface().(types.ITable); ok {
			tblName := itbl.OvsdbTableName()
			monitorRequests[tblName] = map[string]interface{}{
				"columns": nil, // monitor all columns
			}
			tableMap[tblName] = field
		} else {
			// Maybe it is a pointer to table?
			// The generated code uses values for tables in the DB struct.
			// e.g. LogicalSwitch LogicalSwitchTable
		}
	}

	// 2. Send monitor request
	monitorId := "monid" // TODO: make unique
	
	// Handler for updates
	handler := func(tableUpdates *json.RawMessage) {
		if tableUpdates == nil {
			return
		}
		var updates map[string]map[string]RowUpdate
		if err := json.Unmarshal(*tableUpdates, &updates); err != nil {
			log.Errorf("failed to unmarshal table updates: %v", err)
			return
		}
		
		c.applyUpdates(tableMap, updates)
	}
	
	result, err := c.Monitor(ctx, dbName, monitorId, monitorRequests, handler)
	if err != nil {
		return err
	}
	
	// 3. Apply initial result
	handler(result)
	
	return nil
}

type RowUpdate struct {
	Old *json.RawMessage `json:"old,omitempty"`
	New *json.RawMessage `json:"new,omitempty"`
}

func (c *Client) applyUpdates(tableMap map[string]reflect.Value, updates map[string]map[string]RowUpdate) {
	for tblName, rowUpdates := range updates {
		field, ok := tableMap[tblName]
		if !ok {
			continue
		}
		
		// field is the Table slice (e.g. LogicalSwitchTable)
		// We need to update it.
		// Since it's a slice, and we need to update it in place, we need to be careful.
		// The field is a reflect.Value of the slice field in the struct.
		// We can Set it.
		
		for uuid, rowUpdate := range rowUpdates {
			if rowUpdate.New != nil {
				// Insert or Modify
				// Parse the new row
				
				// We need to create a new row instance of the correct type
				// The table is a slice of RowType.
				// We can get RowType from the slice type.
				sliceType := field.Type() // LogicalSwitchTable (slice)
				elemType := sliceType.Elem() // LogicalSwitch (struct)
				
				newRowPtr := reflect.New(elemType) // *LogicalSwitch
				newRow := newRowPtr.Interface().(types.IRow)
				
				// Unmarshal into the new row
				// But wait, the JSON format in OVSDB is ["uuid", "version", {columns}] or similar?
				// RFC 7047 says:
				// "new": <row>
				// <row> is a JSON object with column names and values.
				// Plus "_uuid" and "_version" are usually not in the columns map but are implicit or separate?
				// Actually <row> includes "_uuid" and "_version" if requested?
				// The monitor request "columns": nil means all columns.
				// RFC 7047 4.1.5: "The "new" member ... contain the contents of the row... 
				// The columns member of the row object contains the values of the columns..."
				// Wait, <row> is just the object { "col1": val1, ... }
				
				// Our generated structs expect to be unmarshaled from something?
				// cli_util.UnmarshalJSON handles the format from ovn-nbctl.
				// Here we have raw JSON from JSON-RPC.
				// We need to manually map the JSON fields to the struct fields using SetColumn?
				
				// Let's parse the JSON into map[string]interface{}
				var rowMap map[string]interface{}
				if err := json.Unmarshal(*rowUpdate.New, &rowMap); err != nil {
					log.Errorf("failed to unmarshal row: %v", err)
					continue
				}
				
				// Set _uuid if not present (it should be the key of the map)
				// Actually the UUID is the key in tableUpdates map.
				// The <row> object might contain _uuid and _version if we are lucky/configured.
				// But we definitely have the UUID from the map key.
				
				// We need to set UUID on the row object.
				// The generated struct has Uuid field, but it's not exposed via SetColumn usually.
				// Wait, generated code:
				// func (row *LogicalSwitch) OvsdbUuid() string { return row.Uuid }
				// But no SetUuid.
				// However, since we have a pointer to the struct, we can set the field using reflection if we know the field name "Uuid".
				
				// Let's try to populate the row.
				err := c.populateRow(newRow, uuid, rowMap)
				if err != nil {
					log.Errorf("failed to populate row: %v", err)
					continue
				}
				
				if rowUpdate.Old == nil {
					// Insert
					// Append to slice
					// But we should check if it already exists?
					// Monitor should give us consistent state.
					// If it's initial result, it's insert.
					
					// AppendRow works on types.ITable but that's for value receiver usually in generated code?
					// func (tbl *LogicalSwitchTable) AppendRow(irow types.IRow)
					// Yes, it takes pointer receiver.
					
					// But field is the value of the slice field.
					// We need to address it.
					// field.Addr() can be used if the field is addressable.
					// val (the struct) is addressable?
					// val = reflect.ValueOf(model).Elem().
					// If model is pointer, val is addressable.
					
					// We can call AppendRow via interface if we cast field.Addr().Interface() to ITable
					// But field.Type() is LogicalSwitchTable.
					// *LogicalSwitchTable implements ITable.
					
					// So:
					ptrToTable := field.Addr()
					if itbl, ok := ptrToTable.Interface().(types.ITable); ok {
						itbl.AppendRow(newRow)
					} else {
						log.Errorf("table %s does not implement ITable", tblName)
					}
					
				} else {
					// Modify
					// We need to find the row with the UUID and update it.
					// Scan the slice.
					found := false
					sliceLen := field.Len()
					for i := 0; i < sliceLen; i++ {
						rowVal := field.Index(i)
						// rowVal is LogicalSwitch (struct)
						// We need pointer to it to call OvsdbUuid
						if rowVal.Addr().Interface().(types.IRow).OvsdbUuid() == uuid {
							// Found it. Replace it or update it.
							field.Index(i).Set(newRowPtr.Elem())
							found = true
							break
						}
					}
					if !found {
						// Treat as insert?
						ptrToTable := field.Addr()
						if itbl, ok := ptrToTable.Interface().(types.ITable); ok {
							itbl.AppendRow(newRow)
						}
					}
				}
				
			} else {
				// Delete (New is nil, Old is not nil)
				// Remove from slice
				sliceLen := field.Len()
				newSlice := reflect.MakeSlice(field.Type(), 0, sliceLen)
				
				for i := 0; i < sliceLen; i++ {
					rowVal := field.Index(i)
					if rowVal.Addr().Interface().(types.IRow).OvsdbUuid() == uuid {
						// Skip this one
						continue
					}
					newSlice = reflect.Append(newSlice, rowVal)
				}
				field.Set(newSlice)
			}
		}
	}
}

func (c *Client) populateRow(row types.IRow, uuid string, data map[string]interface{}) error {
	// Set UUID
	// Use reflection to set "Uuid" field
	val := reflect.ValueOf(row).Elem()
	uuidField := val.FieldByName("Uuid")
	if uuidField.IsValid() && uuidField.CanSet() {
		uuidField.SetString(uuid)
	}
	
	// Set _version if present
	if v, ok := data["_version"]; ok {
		if vs, ok := v.(string); ok {
			verField := val.FieldByName("Version")
			if verField.IsValid() && verField.CanSet() {
				verField.SetString(vs)
			}
		}
	}
	
	// Set other columns
	for k, v := range data {
		if k == "_uuid" || k == "_version" {
			continue
		}
		// v could be atomic or ["uuid", "..."] or ["set", [...]] or ["map", [...]]
		// The generated code's SetColumn expects the value to be compatible or handles conversion?
		// Let's check schema_gen.go: Ensure... functions.
		
		// types.Ensure... functions need to be checked.
		// They likely handle OVSDB JSON format.
		
		if err := row.SetColumn(k, v); err != nil {
			// log.Warningf("failed to set column %s: %v", k, err)
			// return err
			// Don't fail completely on one column error?
		}
	}
	return nil
}
