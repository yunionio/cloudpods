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
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	R(&options.SchedulerTestOptions{}, "scheduler-test", "Emulate schedule process",
		func(s *mcclient.ClientSession, args *options.SchedulerTestOptions) error {
			params, err := args.Params(s)
			if err != nil {
				return err
			}
			listFields := []string{"id", "name", "capacity", "count", "score"}
			if args.Details {
				listFields = append(listFields, "capacity_details", "score_details")
			}
			result, err := modules.SchedManager.Test(s, params)
			if err != nil {
				return err
			}
			printList(modulebase.JSON2ListResult(result), listFields)
			return nil
		})

	R(&options.SchedulerForecastOptions{}, "scheduler-forecast", "Forecat scheduler result",
		func(s *mcclient.ClientSession, args *options.SchedulerForecastOptions) error {
			params, err := args.Params(s)
			if err != nil {
				return err
			}
			result, err := modules.SchedManager.DoForecast(s, params.JSON(params))
			if err != nil {
				return err
			}
			fmt.Println(result.YAMLString())
			return nil
		})

	type SchedulerCandidateListOptions struct {
		Type   string `help:"Sched type filter" choices:"baremetal|host"`
		Region string `help:"Cloud region ID"`
		Zone   string `help:"Zone ID"`
		Limit  int    `default:"50" help:"Page limit"`
		Offset int    `default:"0" help:"Page offset"`
	}
	R(&SchedulerCandidateListOptions{}, "scheduler-candidate-list", "List scheduler candidates",
		func(s *mcclient.ClientSession, args *SchedulerCandidateListOptions) error {
			params := jsonutils.NewDict()
			if args.Limit > 0 {
				params.Add(jsonutils.NewInt(int64(args.Limit)), "limit")
			}
			if args.Offset > 0 {
				params.Add(jsonutils.NewInt(int64(args.Offset)), "offset")
			}
			if len(args.Region) > 0 {
				params.Add(jsonutils.NewString(args.Region), "region")
			}
			if len(args.Zone) > 0 {
				params.Add(jsonutils.NewString(args.Zone), "zone")
			}
			if len(args.Type) > 0 {
				params.Add(jsonutils.NewString(args.Type), "type")
			}
			result, err := modules.SchedManager.CandidateList(s, params)
			if err != nil {
				return err
			}
			printList(modulebase.JSON2ListResult(result), []string{
				"id", "name", "host_type", "cpu(free/reserverd/total)",
				"mem(free/reserverd/total)", "storage(free/reserverd/total)",
				"status", "host_status", "enable_status"})
			return nil
		})

	type SchedulerCandidateShowOptions struct {
		ID string `help:"ID or name of host"`
	}
	R(&SchedulerCandidateShowOptions{}, "scheduler-candidate-show", "Show candidate detail",
		func(s *mcclient.ClientSession, args *SchedulerCandidateShowOptions) error {
			params := jsonutils.NewDict()
			result, err := modules.SchedManager.CandidateDetail(s, args.ID, params)
			if err != nil {
				return err
			}
			fmt.Println(result.YAMLString())
			return nil
		})

	type SchedulerCleanCacheOptions struct {
		HostId    string `help:"ID of host" short-token:"h"`
		SessionId string `help:"Session id" short-token:"s"`
	}
	R(&SchedulerCleanCacheOptions{}, "scheduler-clean-cache", "Clean scheduler hosts cache",
		func(s *mcclient.ClientSession, args *SchedulerCleanCacheOptions) error {
			err := modules.SchedManager.CleanCache(s, args.HostId, args.SessionId)
			if err != nil {
				return err
			}
			return nil
		})

	type SchedulerHistoryListOptions struct {
		Limit  int  `default:"50" help:"Page limit"`
		Offset int  `default:"0" help:"Page offset"`
		All    bool `help:"Show all histories, including scheduler-test"`
	}
	R(&SchedulerHistoryListOptions{}, "scheduler-history-list", "Show scheduler history list",
		func(s *mcclient.ClientSession, args *SchedulerHistoryListOptions) error {
			params := jsonutils.NewDict()
			if args.Limit == 0 {
				params.Add(jsonutils.NewInt(1024), "limit")
			} else {
				params.Add(jsonutils.NewInt(int64(args.Limit)), "limit")
			}
			if args.Offset > 0 {
				params.Add(jsonutils.NewInt(int64(args.Offset)), "offset")
			}
			if args.All {
				params.Add(jsonutils.JSONTrue, "all")
			} else {
				params.Add(jsonutils.JSONFalse, "all")
			}
			result, err := modules.SchedManager.HistoryList(s, params)
			if err != nil {
				return err
			}
			printList(modulebase.JSON2ListResult(result), []string{
				"session_id", "time", "status", "consuming",
			})
			return nil
		})

	type SchedulerHistoryShowOptions struct {
		ID     string `help:"Session or guest ID"`
		Log    bool   `help:"Show schedule process log"`
		Raw    bool   `help:"Show raw data"`
		Format string `help:"Output format" choices:"json|yaml"`
	}
	R(&SchedulerHistoryShowOptions{}, "scheduler-history-show", "Show scheduler history detail",
		func(s *mcclient.ClientSession, args *SchedulerHistoryShowOptions) error {
			params := jsonutils.NewDict()
			if args.Log {
				params.Add(jsonutils.JSONTrue, "log")
			} else {
				params.Add(jsonutils.JSONFalse, "log")
			}
			if args.Raw {
				params.Add(jsonutils.JSONTrue, "raw")
			} else {
				params.Add(jsonutils.JSONFalse, "false")
			}
			result, err := modules.SchedManager.HistoryShow(s, args.ID, params)
			if err != nil {
				return err
			}
			if args.Format == "json" {
				fmt.Println(result.PrettyString())
			} else {
				fmt.Println(result.YAMLString())
			}
			return nil
		})

	type SyncOpt struct {
		Wait bool `help:"wait sync finish"`
	}
	R(&SyncOpt{}, "scheduler-sync-sku", "Sync scheduler SKU cache",
		func(s *mcclient.ClientSession, args *SyncOpt) error {
			result, err := modules.SchedManager.SyncSku(s, args.Wait)
			if err != nil {
				return err
			}
			fmt.Println(result.YAMLString())
			return nil
		})
}
