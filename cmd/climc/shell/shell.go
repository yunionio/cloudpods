package shell

import (
	"fmt"
	"github.com/yunionio/jsonutils"
)

type CMD struct {
	Options  interface{}
	Command  string
	Desc     string
	Callback interface{}
}

var CommandTable []CMD = make([]CMD, 0)

func R(options interface{}, command string, desc string, callback interface{}) {
	CommandTable = append(CommandTable, CMD{options, command, desc, callback})
}

type BaseListOptions struct {
	Limit         int      `default:"20" help:"Page limit"`
	Offset        int      `default:"0" help:"Page offset"`
	OrderBy       string   `help:"Name of the field to be ordered by"`
	Order         string   `help:"List order" choices:"desc|asc"`
	Details       bool     `help:"Show more details"`
	Search        string   `help:"Filter results by a simple keyword search"`
	Meta          bool     `help:"Piggyback metadata information"`
	Filter        []string `help:"Filters"`
	FilterAny     bool     `help:"If true, match if any of the filters matches; otherwise, match if all of the filters match"`
	Admin         bool     `help:"Is an admin call?"`
	Tenant        string   `help:"Tenant ID or Name"`
	User          string   `help:"User ID or Name"`
	System        bool     `help:"Show system resource"`
	PendingDelete bool     `help:"Show pending deleted resource"`
	Field         []string `help:"Show only specified fields"`
}

func FetchPagingParams(options BaseListOptions) *jsonutils.JSONDict {
	params := jsonutils.NewDict()
	if options.Limit > 0 {
		params.Add(jsonutils.NewInt(int64(options.Limit)), "limit")
	}
	if options.Offset > 0 {
		params.Add(jsonutils.NewInt(int64(options.Offset)), "offset")
	}
	if len(options.OrderBy) > 0 {
		params.Add(jsonutils.NewString(options.OrderBy), "order_by")
	}
	if len(options.Order) > 0 {
		params.Add(jsonutils.NewString(options.Order), "order")
	}
	if options.Details {
		params.Add(jsonutils.JSONTrue, "details")
	} else {
		params.Add(jsonutils.JSONFalse, "details")
	}
	if len(options.Search) > 0 {
		params.Add(jsonutils.NewString(options.Search), "search")
	}
	if options.Meta {
		params.Add(jsonutils.JSONTrue, "with_meta")
	}
	if len(options.Filter) > 0 {
		arr := jsonutils.NewArray()
		for _, f := range options.Filter {
			arr.Add(jsonutils.NewString(f))
		}
		params.Add(arr, "filter")
		if options.FilterAny {
			params.Add(jsonutils.JSONTrue, "filter_any")
		}
	}
	if options.Admin {
		params.Add(jsonutils.JSONTrue, "admin")
	}
	if len(options.Tenant) > 0 {
		params.Add(jsonutils.NewString(options.Tenant), "tenant")
		if !options.Admin {
			params.Add(jsonutils.JSONTrue, "admin")
		}
	}
	if len(options.User) > 0 {
		params.Add(jsonutils.NewString(options.User), "user")
	}
	if options.System {
		params.Add(jsonutils.JSONTrue, "system")
		if !options.Admin {
			params.Add(jsonutils.JSONTrue, "admin")
		}
	}
	if options.PendingDelete {
		params.Add(jsonutils.JSONTrue, "pending_delete")
		if !options.Admin {
			params.Add(jsonutils.JSONTrue, "admin")
		}
	}
	if len(options.Field) > 0 {
		arr := jsonutils.NewArray()
		for _, f := range options.Field {
			arr.Add(jsonutils.NewString(f))
		}
		params.Add(arr, "field")
	}
	return params
}

func InvalidUpdateError() error {
	return fmt.Errorf("No valid update data")
}
