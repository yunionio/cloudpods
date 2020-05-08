package types

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/pkg/errors"
)

type Schema struct {
	Name    string
	Version string
	Cksum   string
	Tables  Tables
}

type Tables map[string]Table
type Table struct {
	Name    string
	Columns map[string]Column
	MaxRows int
	IsRoot  bool
	Indexes [][]string
}

type Columns map[string]Column
type Column struct {
	Name      string
	Type      Type
	Ephemeral bool
	Mutable   bool
}

type Type struct {
	Key          BaseType
	Value        BaseType
	Min          int
	Max          int
	MaxUnlimited bool
}

type BaseType struct {
	Type       Atomic
	Enum       interface{} // TODO
	MinInteger int
	MaxInteger int
	MinReal    int
	MaxReal    int
	MinLength  int    // strings only
	MaxLength  int    // strings only
	RefTable   string // uuids only
	RefType    string // only with refTable
}

var (
	errSchema = errors.New("bad schema")
)

func ParseSchema(r io.Reader) (*Schema, error) {
	type (
		PColumn struct {
			Type      interface{}
			Ephemeral bool
			Mutable   bool
		}
		PTable struct {
			Columns map[string]PColumn
			MaxRows int
			IsRoot  bool
			Indexes [][]string
		}
		PSchema struct {
			Name    string
			Version string
			Cksum   string
			Tables  map[string]PTable
		}
	)

	psch := &PSchema{}
	dec := json.NewDecoder(r)
	err := dec.Decode(psch)
	if err != nil {
		return nil, err
	}
	sch := &Schema{
		Name:    psch.Name,
		Version: psch.Version,
		Cksum:   psch.Cksum,
		Tables:  Tables{},
	}
	for tblName, ptbl := range psch.Tables {
		tbl := Table{
			Name:    tblName,
			MaxRows: ptbl.MaxRows,
			IsRoot:  ptbl.IsRoot,
			Indexes: ptbl.Indexes,
			Columns: Columns{
				"_uuid": Column{
					Name: "_uuid",
					Type: Type{
						Key: BaseType{
							Type: Uuid,
						},
						Min: 1,
						Max: 1,
					},
				},
				"_version": Column{
					Name: "_version",
					Type: Type{
						Key: BaseType{
							Type: Uuid,
						},
						Min: 1,
						Max: 1,
					},
				},
			},
		}
		for colName, pcol := range ptbl.Columns {
			col := Column{
				Name:      colName,
				Ephemeral: pcol.Ephemeral,
				Mutable:   pcol.Mutable,
				Type: Type{
					Min: 1,
					Max: 1,
				},
			}
			switch t0 := pcol.Type.(type) {
			case string:
				col.Type.Key.Type = Atomic(t0)
			case map[string]interface{}:
				if kbt, err := parseBaseType(t0["key"]); err != nil {
					return nil, errors.Wrap(err, "bad key type")
				} else {
					col.Type.Key = kbt
				}

				if vbt, err := parseBaseType(t0["value"]); err != nil {
					return nil, errors.Wrap(err, "bad value type")
				} else {
					col.Type.Value = vbt
				}

				if minv, exist := t0["min"]; exist {
					if min, ok := minv.(float64); ok {
						col.Type.Min = int(min)
					} else {
						return nil, errors.Wrapf(errSchema, "bad min: %#v", minv)
					}
				}
				if maxv, exist := t0["max"]; exist {
					if max, ok := maxv.(float64); ok {
						col.Type.Max = int(max)
					} else if unlimited, ok := maxv.(string); ok && unlimited == "unlimited" {
						col.Type.Max = -1
						col.Type.MaxUnlimited = true
					} else {
						return nil, errors.Wrapf(errSchema, "bad max: %#v", maxv)
					}
				}
			default:
				return nil, errors.Wrapf(errSchema, "bad type: %#v", t0)
			}
			tbl.Columns[colName] = col
		}
		sch.Tables[tblName] = tbl
	}
	return sch, nil
}

// rfc7047, 3.2 Schema Format, <base-type>
func parseBaseType(bti interface{}) (BaseType, error) {
	bt := BaseType{}
	switch btv := bti.(type) {
	case string:
		bt.Type = Atomic(btv)
		return bt, nil
	case map[string]interface{}:
		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		if err := enc.Encode(btv); err != nil {
			return bt, errors.Wrapf(errSchema, "encode %v", err)
		}
		dec := json.NewDecoder(buf)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&bt); err != nil {
			return bt, errors.Wrapf(errSchema, "decode %v", err)
		}
		return bt, nil
	case nil:
		return bt, nil
	default:
		return BaseType{}, errors.Wrapf(errSchema, "unexpected base type %#v", bti)
	}
}
