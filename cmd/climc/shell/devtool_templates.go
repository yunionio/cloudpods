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
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	printAnsiblePlaybookObject := func(obj jsonutils.JSONObject) {
		dict := obj.(*jsonutils.JSONDict)
		pbJson, err := dict.Get("playbook")
		if err != nil {
			printObject(obj)
			return
		}
		pbStr := pbJson.YAMLString()
		dict.Set("playbook", jsonutils.NewString(pbStr))
		printObject(obj)
	}

	type TemplateListOptions struct {
		options.BaseListOptions
		Name string `help:"cloud region ID or Name" json:"-"`
	}

	R(&TemplateListOptions{}, "devtool-template-list", "List Devtool Templates", func(s *mcclient.ClientSession, args *TemplateListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		var result *modulebase.ListResult
		result, err = modules.DevToolTemplates.List(s, params)
		printList(result, modules.DevToolTemplates.GetColumns(s))
		return nil
	})

	R(
		&options.DevtoolTemplateCreateOptions{},
		"devtool-template-create",
		"Create a template repo component",
		func(s *mcclient.ClientSession, opts *options.DevtoolTemplateCreateOptions) error {

			params, err := opts.Params()
			if err != nil {
				return err
			}
			log.Infof("ansible playbook create opts: %+v", params)
			// apb, err := modules.Devtool.Create(s, params)
			apb, err := modules.DevToolTemplates.Create(s, params)
			if err != nil {
				return err
			}
			printAnsiblePlaybookObject(apb)
			return nil
		},
	)

	R(
		&options.DevtoolTemplateIdOptions{},
		"devtool-template-show",
		"Show devtool template",
		func(s *mcclient.ClientSession, opts *options.DevtoolTemplateIdOptions) error {
			apb, err := modules.DevToolTemplates.Get(s, opts.ID, nil)
			if err != nil {
				return err
			}
			printAnsiblePlaybookObject(apb)
			return nil
		},
	)

	R(
		&options.DevtoolTemplateBindingOptions{},
		"devtool-template-bind",
		"Binding devtool template to a host/vm",
		func(s *mcclient.ClientSession, opts *options.DevtoolTemplateBindingOptions) error {
			params := jsonutils.NewDict()
			params.Set("server_id", jsonutils.NewString(opts.ServerID))
			_, err := modules.DevToolTemplates.PerformAction(s, opts.ID, "bind", params)
			if err != nil {
				return err
			}
			// printAnsiblePlaybookObject(apb)
			return nil
		},
	)

	R(
		&options.DevtoolTemplateBindingOptions{},
		"devtool-template-unbind",
		"Binding devtool template to a host/vm",
		func(s *mcclient.ClientSession, opts *options.DevtoolTemplateBindingOptions) error {
			params := jsonutils.NewDict()
			params.Set("server_id", jsonutils.NewString(opts.ServerID))
			_, err := modules.DevToolTemplates.PerformAction(s, opts.ID, "unbind", params)
			if err != nil {
				return err
			}
			// printAnsiblePlaybookObject(apb)
			return nil
		},
	)

	R(
		&options.DevtoolTemplateIdOptions{},
		"devtool-template-delete",
		"Delete devtool template",
		func(s *mcclient.ClientSession, opts *options.DevtoolTemplateIdOptions) error {
			apb, err := modules.DevToolTemplates.Delete(s, opts.ID, nil)
			if err != nil {
				return err
			}
			printAnsiblePlaybookObject(apb)
			return nil
		},
	)

	/*
		type DevToolTemplateShowOptions struct {
			ID string `help:"ID or Name of the DevToolTemplate to show"`
		}
		R(&DevToolTemplateShowOptions{}, "devtool-template-show", "Show template details", func(s *mcclient.ClientSession, args *DevToolTemplateShowOptions) error {
			result, err := modules.DevToolTemplates.Get(s, args.ID, nil)
			if err != nil {
				return err
			}
			printObject(result)
			return nil
		})

		type DevToolTemplateUpdateOptions struct {
			ID       string `help:"ID or Name of DevToolTemplate to update"`
			Day      int    `help:"Template runs at given day" json:"-" default:"-1"`
			Hour     int    `help:"Template runs at given hour" json:"-" default:"-1"`
			Min      int    `help:"Template runs at given min" default:"-1"`
			Sec      int    `help:"Template runs at given sec" default:"-1"`
			Interval int    `help:"Template runs at given interval" default:"-1"`
			Start    bool   `help:"start job when created"`
			Stop     bool   `help:"start job when created"`
			Enable   bool   `help:"Set job status enabled"`
			Disable  bool   `help:"Set job status enabled"`
		}
		R(&DevToolTemplateUpdateOptions{}, "devtool-template-update", "Update DevToolTemplate", func(s *mcclient.ClientSession, args *DevToolTemplateUpdateOptions) error {
			result, err := modules.DevToolTemplates.Get(s, args.ID, nil)
			if err != nil {
				return err
			}

			params := jsonutils.NewDict()
			interval, _ := result.Int("interval")
			day, _ := result.Int("day")
			params.Add(jsonutils.NewString(args.ID), "id")
			params.Add(jsonutils.NewInt(int64(interval)), "interval")
			params.Add(jsonutils.NewInt(int64(day)), "day")

			log.Infof("DevToolTemplateUpdateOptions args: %+v", args)
			if args.Interval >= 0 {
				params.Add(jsonutils.NewInt(int64(args.Interval)), "interval")
				if args.Interval > 0 {
					params.Add(jsonutils.NewInt(0), "day")
				}
			} else if args.Day >= 0 {
				params.Add(jsonutils.NewInt(int64(args.Day)), "day")
				if args.Day > 0 {
					params.Add(jsonutils.NewInt(0), "interval")
				}
			}
			if args.Hour >= 0 {
				params.Add(jsonutils.NewInt(int64(args.Hour)), "hour")
			}
			if args.Min >= 0 {
				params.Add(jsonutils.NewInt(int64(args.Min)), "min")
			}
			if args.Sec >= 0 {
				params.Add(jsonutils.NewInt(int64(args.Sec)), "sec")
			}

			if args.Start && args.Stop {
				return fmt.Errorf("can not set job start and stop at the same time")
			} else if args.Start {
				params.Add(jsonutils.JSONTrue, "start")
			} else if args.Stop {
				params.Add(jsonutils.JSONFalse, "start")
			}
			if args.Enable && args.Disable {
				return fmt.Errorf("can not set job enabled and disabled at the same time")
			} else if args.Enable {
				params.Add(jsonutils.JSONTrue, "enabled")
			} else if args.Disable {
				params.Add(jsonutils.JSONFalse, "enabled")
			}

			result, err = modules.DevToolTemplates.Update(s, args.ID, params)
			if err != nil {
				return err
			}
			printObject(result)
			return nil
		})

		R(&DevToolTemplateShowOptions{}, "devtool-template-delete", "Delete DevToolTemplate", func(s *mcclient.ClientSession, args *DevToolTemplateShowOptions) error {
			result, err := modules.DevToolTemplates.Delete(s, args.ID, nil)
			if err != nil {
				return err
			}
			printObject(result)
			return nil
		})

	*/
}
