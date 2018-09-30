package options

import (
	"fmt"
	"reflect"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/reflectutils"
)

// Int returns a pointer to int type with the same value as the argument.  This
// is intended to be used for literal initialization of options
func Int(v int) *int {
	return &v
}

// Bool returns a pointer to bool type with the same value as the argument.
// This is intended to be used for literal initialization of options
func Bool(v bool) *bool {
	return &v
}

// IntV returns the integer value as pointed to by the argument if it's
// non-nil, return 0 otherwise
func IntV(p *int) int {
	if p != nil {
		return *p
	}
	return 0
}

// BoolV returns the bool value as pointed to by the argument if it's non-nil,
// return false otherwise
func BoolV(p *bool) bool {
	if p != nil {
		return *p
	}
	return false
}

type IParamsOptions interface {
	Params() (*jsonutils.JSONDict, error)
}

var BaseListOptionsType = reflect.TypeOf((*BaseListOptions)(nil)).Elem()

func optionsStructRvToParams(rv reflect.Value) (*jsonutils.JSONDict, error) {
	p := jsonutils.NewDict()
	rvType := rv.Type()
	for i := 0; i < rvType.NumField(); i++ {
		ft := rvType.Field(i)
		name := reflectutils.GetStructFieldName(&ft)
		if name == "" {
			continue
		}
		f := rv.Field(i)
	begin:
		switch f.Kind() {
		case reflect.Ptr:
			if f.IsNil() {
				continue
			}
			f = f.Elem()
			goto begin
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			rv64 := f.Convert(gotypes.Int64Type)
			i64 := rv64.Interface().(int64)
			p.Set(name, jsonutils.NewInt(i64))
		case reflect.Bool:
			b := f.Interface().(bool)
			p.Set(name, jsonutils.NewBool(b))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			// NOTE uint64 converted to int64
			rv64 := f.Convert(gotypes.Uint64Type)
			i64 := rv64.Interface().(uint64)
			p.Set(name, jsonutils.NewInt(int64(i64)))
		case reflect.Float32, reflect.Float64:
			rv64 := f.Convert(gotypes.Float64Type)
			f64 := rv64.Interface().(float64)
			p.Set(name, jsonutils.NewFloat(f64))
		case reflect.String:
			s := f.Interface().(string)
			if len(s) > 0 {
				p.Set(name, jsonutils.NewString(s))
			}
		case reflect.Struct:
			if ft.Anonymous {
				continue
			}
			if f.Type() == gotypes.TimeType {
				t := f.Interface().(time.Time)
				p.Set(name, jsonutils.NewTimeString(t))
				continue
			}
			// TODO
			msg := fmt.Sprintf("do not know what to do with non-anonymous struct field: %s", ft.Name)
			panic(msg)
		case reflect.Slice, reflect.Array:
			l := f.Len()
			for i := 0; i < l; i++ {
				namei := fmt.Sprintf("%s.%d", name, i)
				vali := jsonutils.Marshal(f.Index(i).Interface())
				p.Set(namei, vali)
			}
		default:
			msg := fmt.Sprintf("unsupported field type %s: %s", ft.Name, ft.Type)
			panic(msg)
		}
	}
	return p, nil
}

func optionsStructToParams(v interface{}) (*jsonutils.JSONDict, error) {
	rv := reflect.ValueOf(v).Elem()
	return optionsStructRvToParams(rv)
}

// StructToParams converts the struct as pointed to by the argument to JSON
// dict params, ignoring any embedded in struct
func StructToParams(v interface{}) (*jsonutils.JSONDict, error) {
	return optionsStructToParams(v)
}

// ListStructToParams converts the struct as pointed to by the argument to JSON
// dict params, taking into account .BaseListOptions.Params() if it exists
func ListStructToParams(v interface{}) (*jsonutils.JSONDict, error) {
	rv := reflect.ValueOf(v).Elem()
	params, err := optionsStructRvToParams(rv)
	if err != nil {
		return nil, err
	}
	{
		f := rv.FieldByName("BaseListOptions")
		if f.IsValid() {
			listOpts, ok := f.Addr().Interface().(*BaseListOptions)
			if ok {
				listParams, err := listOpts.Params()
				if err != nil {
					return nil, err
				}
				params.Update(listParams)
			}
		}
	}
	return params, nil
}

const (
	ListOrderAsc  = "asc"
	ListOrderDesc = "desc"
)

type BaseListOptions struct {
	Limit            *int     `default:"20" help:"Page limit"`
	Offset           *int     `default:"0" help:"Page offset"`
	OrderBy          []string `help:"Name of the field to be ordered by"`
	Order            string   `help:"List order" choices:"desc|asc"`
	Details          *bool    `help:"Show more details" default:"false"`
	Search           string   `help:"Filter results by a simple keyword search"`
	Meta             *bool    `help:"Piggyback metadata information"`
	Filter           []string `help:"Filters"`
	JointFilter      []string `help:"Filters with joint table col; joint_tbl.related_key(origin_key).filter_col.filter_cond(filters)"`
	FilterAny        *bool    `help:"If true, match if any of the filters matches; otherwise, match if all of the filters match"`
	Admin            *bool    `help:"Is an admin call?"`
	Tenant           string   `help:"Tenant ID or Name"`
	User             string   `help:"User ID or Name"`
	System           *bool    `help:"Show system resource"`
	PendingDelete    *bool    `help:"Show only pending deleted resource"`
	PendingDeleteAll *bool    `help:"Show all resources including pending deleted" json:"-"`
	Field            []string `help:"Show only specified fields"`
	ShowEmulated     *bool    `help:"Show all resources including the emulated resources"`
	ExportFile       string   `help:"Export to file" metavar:"<EXPORT_FILE_PATH>" json:"-"`
	ExportKeys       string   `help:"Export field keys"`
	ExportTexts      string   `help:"Export field displayname texts" json:"-"`
}

func (opts *BaseListOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := optionsStructToParams(opts)
	if err != nil {
		return nil, err
	}
	if len(opts.Filter) == 0 {
		params.Remove("filter_any")
	}
	if BoolV(opts.PendingDeleteAll) {
		params.Set("pending_delete", jsonutils.NewString("all"))
		params.Set("details", jsonutils.JSONTrue) // required to get pending_deleted field
	}
	if opts.Admin == nil {
		requiresSystem := len(opts.Tenant) > 0 ||
			BoolV(opts.System) ||
			BoolV(opts.PendingDelete) ||
			BoolV(opts.PendingDeleteAll)
		if requiresSystem {
			params.Set("admin", jsonutils.JSONTrue)
		}
	}
	return params, nil
}
