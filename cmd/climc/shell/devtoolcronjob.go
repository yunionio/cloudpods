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
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func paramValidator(param jsonutils.JSONObject) (bool, error) {
	// TODO 移到 server 端
	log.Infof("paramValidator: param: %+v", param)
	interval, err := param.Int("interval")
	if err != nil {
		return true, nil
	}
	day, err := param.Int("day")
	if err != nil {
		return true, nil
	}
	if interval == 0 && day == 0 {
		return false, fmt.Errorf("interval and day can not be 0 at the same time")
	}
	return true, nil
}

func init() {
	type CronjobListOptions struct {
		options.BaseListOptions
		Name string `help:"cloud region ID or Name" json:"-"`
	}

	R(&CronjobListOptions{}, "devtoolcronjob-list", "List Devtool Cronjobs", func(s *mcclient.ClientSession, args *CronjobListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		var result *modulebase.ListResult
		result, err = modules.DevToolCronjobs.List(s, params)
		printList(result, modules.DevToolCronjobs.GetColumns(s))
		return nil
	})

	type CronjobCreateOptions struct {
		NAME     string `help:"Ansible Playbook ID or Name" json:"-"`
		Day      int    `help:"Cronjob runs at given day" default:"0"`
		Hour     int    `help:"Cronjob runs at given hour" default:"0"`
		Min      int    `help:"Cronjob runs at given min" default:"0"`
		Sec      int    `help:"Cronjob runs at given sec" default:"0"`
		Interval int64  `help:"Cronjob runs at given interval" default:"0"`
		Start    bool   `help:"start job when created" default:"false"`
		Enabled  bool   `help:"Set job status enabled" default:"false"`
	}
	R(
		&CronjobCreateOptions{},
		"devtoolcronjob-create",
		"Create a cronjob repo component",
		func(s *mcclient.ClientSession, args *CronjobCreateOptions) error {
			result, err := modules.AnsiblePlaybooks.Get(s, args.NAME, nil)
			if err != nil {
				return err
			}
			ansiblePlaybookName, err := result.GetString("name")
			if err != nil {
				return err
			}

			ansiblePlaybookID, err := result.GetString("id")
			if err != nil {
				return err
			}

			params := jsonutils.NewDict()
			params.Add(jsonutils.NewString(ansiblePlaybookName), "name")

			params.Add(jsonutils.NewString(ansiblePlaybookID), "ansible_playbook_id")

			if args.Start {
				params.Add(jsonutils.JSONTrue, "start")

			}
			if args.Enabled {
				params.Add(jsonutils.JSONTrue, "enabled")
			} else if args.Interval > 0 {
				params.Add(jsonutils.NewInt(int64(args.Interval)), "interval")
			} else {
				params.Add(jsonutils.NewInt(int64(0)), "interval")
				params.Add(jsonutils.NewInt(int64(args.Day)), "day")
				params.Add(jsonutils.NewInt(int64(args.Hour)), "hour")
				params.Add(jsonutils.NewInt(int64(args.Min)), "min")
				params.Add(jsonutils.NewInt(int64(args.Sec)), "sec")
			}
			ok, err := paramValidator(params)
			if err != nil || !ok {
				log.Infof("paramValidator error %s", err)
				return err
			}
			cronjob, err := modules.DevToolCronjobs.Create(s, params)
			if err != nil {
				log.Errorf("modules.DevToolCronjobs.Create error %s", err)
				return err
			}
			printObject(cronjob)
			return nil
		},
	)

	type DevToolCronjobShowOptions struct {
		ID string `help:"ID or Name of the DevToolCronjob to show"`
	}
	R(&DevToolCronjobShowOptions{}, "devtoolcronjob-show", "Show cronjob details", func(s *mcclient.ClientSession, args *DevToolCronjobShowOptions) error {
		result, err := modules.DevToolCronjobs.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type DevToolCronjobUpdateOptions struct {
		ID       string `help:"ID or Name of DevToolCronjob to update"`
		Day      int    `help:"Cronjob runs at given day" json:"-" default:"-1"`
		Hour     int    `help:"Cronjob runs at given hour" json:"-" default:"-1"`
		Min      int    `help:"Cronjob runs at given min" default:"-1"`
		Sec      int    `help:"Cronjob runs at given sec" default:"-1"`
		Interval int    `help:"Cronjob runs at given interval" default:"-1"`
		Start    bool   `help:"start job when created"`
		Stop     bool   `help:"start job when created"`
		Enable   bool   `help:"Set job status enabled"`
		Disable  bool   `help:"Set job status enabled"`
	}
	R(&DevToolCronjobUpdateOptions{}, "devtoolcronjob-update", "Update DevToolCronjob", func(s *mcclient.ClientSession, args *DevToolCronjobUpdateOptions) error {
		result, err := modules.DevToolCronjobs.Get(s, args.ID, nil)
		if err != nil {
			return err
		}

		params := jsonutils.NewDict()
		interval, _ := result.Int("interval")
		day, _ := result.Int("day")
		params.Add(jsonutils.NewString(args.ID), "id")
		params.Add(jsonutils.NewInt(int64(interval)), "interval")
		params.Add(jsonutils.NewInt(int64(day)), "day")

		log.Infof("DevToolCronjobUpdateOptions args: %+v", args)
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

		ok, err := paramValidator(params)
		if err != nil || !ok {
			return err
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

		result, err = modules.DevToolCronjobs.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&DevToolCronjobShowOptions{}, "devtoolcronjob-delete", "Delete DevToolCronjob", func(s *mcclient.ClientSession, args *DevToolCronjobShowOptions) error {
		result, err := modules.DevToolCronjobs.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
