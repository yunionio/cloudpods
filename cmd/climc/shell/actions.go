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

package shell

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type BaseActionListOptions struct {
	Scope      string   `help:"scope" choices:"project|domain|system"`
	Since      string   `help:"Show logs since specific date" metavar:"DATETIME"`
	Until      string   `help:"Show logs until specific date" metavar:"DATETIME"`
	Limit      int64    `help:"Limit number of logs" default:"20"`
	Offset     int64    `help:"Offset"`
	Ascending  bool     `help:"Ascending order"`
	Descending bool     `help:"Descending order"`
	Action     []string `help:"Log action"`
	Search     string   `help:"Filter action logs by obj_name, using 'like' syntax."`
	Admin      bool     `help:"admin mode"`
	Succ       bool     `help:"Show success action log only"`
	Fail       bool     `help:"Show failed action log only"`

	User    []string `help:"filter by operator user"`
	Project []string `help:"filter by owner project"`

	PagingMarker string `help:"marker for pagination"`
}

type ActionListOptions struct {
	BaseActionListOptions
	Id   string   `help:"" metavar:"OBJ_ID"`
	Type []string `help:"Type of relevant object" metavar:"OBJ_TYPE"`
}

type TypeActionListOptions struct {
	BaseActionListOptions
	ID string `help:"" metavar:"OBJ_ID"`
}

func doActionList(s *mcclient.ClientSession, args *ActionListOptions) error {
	params := jsonutils.NewDict()
	if len(args.Type) > 0 {
		params.Add(jsonutils.NewStringArray(args.Type), "obj_type")
	}
	if len(args.Id) > 0 {
		params.Add(jsonutils.NewString(args.Id), "obj_id")
	}
	if len(args.Search) > 0 {
		params.Add(jsonutils.NewString(args.Search), "search")
	}
	if len(args.User) > 0 {
		params.Add(jsonutils.NewStringArray(args.User), "user")
	}
	if len(args.Project) > 0 {
		params.Add(jsonutils.NewStringArray(args.Project), "project")
	}
	if len(args.Scope) > 0 {
		params.Add(jsonutils.NewString(args.Scope), "scope")
	}
	if len(args.PagingMarker) > 0 {
		params.Add(jsonutils.NewString(args.PagingMarker), "paging_marker")
	}
	if len(args.Since) > 0 {
		params.Add(jsonutils.NewString(args.Since), "since")
	}
	if len(args.Until) > 0 {
		params.Add(jsonutils.NewString(args.Until), "until")
	}
	if args.Limit > 0 {
		params.Add(jsonutils.NewInt(args.Limit), "limit")
	}
	if args.Offset > 0 {
		params.Add(jsonutils.NewInt(args.Offset), "offset")
	}
	if args.Ascending && !args.Descending {
		params.Add(jsonutils.NewString("asc"), "order")
	} else if !args.Ascending && args.Descending {
		params.Add(jsonutils.NewString("desc"), "order")
	}
	if len(args.Action) > 0 {
		params.Add(jsonutils.NewStringArray(args.Action), "action")
	}
	if args.Admin {
		params.Add(jsonutils.JSONTrue, "admin")
	}
	if args.Succ && args.Fail {
		return fmt.Errorf("succ and fail can't go together")
	}
	if args.Succ {
		params.Add(jsonutils.JSONTrue, "success")
	}
	if args.Fail {
		params.Add(jsonutils.JSONFalse, "success")
	}
	logs, err := modules.Actions.List(s, params)
	if err != nil {
		return err
	}
	printList(logs, modules.Actions.GetColumns(s))
	return nil
}

func init() {
	R(&ActionListOptions{}, "action-show", "Show operation action logs", doActionList)

	R(&TypeActionListOptions{}, "server-action", "Show operation action logs of server", func(s *mcclient.ClientSession, args *TypeActionListOptions) error {
		nargs := ActionListOptions{BaseActionListOptions: args.BaseActionListOptions, Id: args.ID, Type: []string{"server"}}
		return doActionList(s, &nargs)
	})

	R(&TypeActionListOptions{}, "host-action", "Show operation action logs of host", func(s *mcclient.ClientSession, args *TypeActionListOptions) error {
		nargs := ActionListOptions{BaseActionListOptions: args.BaseActionListOptions, Id: args.ID, Type: []string{"host"}}
		return doActionList(s, &nargs)
	})

	R(&TypeActionListOptions{}, "vcenter-action", "Show operation action logs of vcenter", func(s *mcclient.ClientSession, args *TypeActionListOptions) error {
		nargs := ActionListOptions{BaseActionListOptions: args.BaseActionListOptions, Id: args.ID, Type: []string{"vcenter"}}
		return doActionList(s, &nargs)
	})
}
