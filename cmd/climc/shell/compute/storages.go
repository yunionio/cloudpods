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

package compute

import (
	"fmt"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/mcclient/options/compute"
	"yunion.io/x/onecloud/pkg/util/cephutils"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.Storages).WithContextManager(&modules.Zones)
	cmd.List(&compute.StorageListOptions{})
	cmd.Update(&compute.StorageUpdateOptions{})
	cmd.Create(&compute.StorageCreateOptions{})
	cmd.Show(&options.BaseShowOptions{})
	cmd.Delete(&options.BaseIdOptions{})
	cmd.Perform("enable", &options.BaseIdOptions{})
	cmd.Perform("disable", &options.BaseIdOptions{})
	cmd.Perform("online", &options.BaseIdOptions{})
	cmd.Perform("offline", &options.BaseIdOptions{})
	cmd.Perform("cache-image", &compute.StorageCacheImageActionOptions{})
	cmd.Perform("uncache-image", &compute.StorageUncacheImageActionOptions{})
	cmd.Perform("change-owner", &options.ChangeOwnerOptions{})
	cmd.Perform("force-detach-host", &compute.StorageForceDetachHost{})
	cmd.Perform("public", &options.BasePublicOptions{})
	cmd.Perform("private", &options.BaseIdOptions{})
	cmd.Get("hardware-info", &options.BaseIdOptions{})
	cmd.Perform("set-hardware-info", &compute.StorageSetHardwareInfoOptions{})
	cmd.Perform("set-commit-bound", &compute.StorageSetCommitBoundOptions{})

	type StorageCephRunOptions struct {
		ID     string `help:"ID or name of ceph storage"`
		SUBCMD string `help:"ceph subcommand"`
	}
	R(&StorageCephRunOptions{}, "storage-ceph-run", "Run ceph command against a ceph storage", func(s *mcclient.ClientSession, args *StorageCephRunOptions) error {
		result, err := modules.Storages.Get(s, args.ID, nil)
		if err != nil {
			return errors.Wrap(err, "Get")
		}
		info := struct {
			StorageType string `json:"storage_type"`
			StorageConf struct {
				Key               string `json:"key"`
				MonHost           string `json:"mon_host"`
				Pool              string `json:"pool"`
				EnableMessengerV2 bool   `json:"enable_messenger_v2"`
			}
		}{}
		err = result.Unmarshal(&info)
		if err != nil {
			return errors.Wrap(err, "Unmarshal")
		}
		if info.StorageType != "rbd" {
			return errors.Errorf("invalid storage_type %s", info.StorageType)
		}
		cli, err := cephutils.NewClient(
			info.StorageConf.MonHost,
			info.StorageConf.Key,
			info.StorageConf.Pool,
			info.StorageConf.EnableMessengerV2,
		)
		if err != nil {
			return errors.Wrap(err, "cephutils.NewClient")
		}
		defer cli.Close()

		if args.SUBCMD == "showconf" {
			cli.ShowConf()
			return nil
		}

		opts := strings.Split(args.SUBCMD, " ")
		if len(opts) == 0 {
			return errors.Errorf("empty command")
		}
		output, err := cli.Output(opts[0], opts[1:])
		if err != nil {
			return errors.Wrap(err, "Run")
		}
		fmt.Println(output.PrettyString())
		return nil
	})
}
