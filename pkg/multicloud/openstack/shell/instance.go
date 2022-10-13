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
	"yunion.io/x/onecloud/pkg/multicloud/openstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type InstanceListOptions struct {
		Host string `help:"Host name for filter instance list"`
	}
	shellutils.R(&InstanceListOptions{}, "instance-list", "List instances", func(cli *openstack.SRegion, args *InstanceListOptions) error {
		instances, err := cli.GetInstances(args.Host)
		if err != nil {
			return err
		}
		printList(instances, 0, 0, 0, nil)
		return nil
	})

	type InstanceOptions struct {
		ID string `help:"Instance ID"`
	}

	shellutils.R(&InstanceOptions{}, "instance-network-list", "List instance network", func(cli *openstack.SRegion, args *InstanceOptions) error {
		ports, err := cli.GetInstancePorts(args.ID)
		if err != nil {
			return err
		}
		printObject(ports)
		return nil
	})

	shellutils.R(&InstanceOptions{}, "instance-show", "Show instance", func(cli *openstack.SRegion, args *InstanceOptions) error {
		instance, err := cli.GetInstance(args.ID)
		if err != nil {
			return err
		}
		printObject(instance)
		return nil
	})

	shellutils.R(&InstanceOptions{}, "instance-metadata", "Show instance metadata", func(cli *openstack.SRegion, args *InstanceOptions) error {
		metadata, err := cli.GetInstanceMetadata(args.ID)
		if err != nil {
			return err
		}
		printObject(metadata)
		return nil
	})

	shellutils.R(&InstanceOptions{}, "instance-vnc", "Show instance vnc url", func(cli *openstack.SRegion, args *InstanceOptions) error {
		ret, err := cli.GetInstanceVNCUrl(args.ID, true)
		if err != nil {
			ret, err = cli.GetInstanceVNC(args.ID, true)
		}
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	type InstanceDeployOptions struct {
		ID       string `help:"Instance ID"`
		Password string `help:"Instance password"`
		Name     string `help:"Instance name"`
	}

	shellutils.R(&InstanceDeployOptions{}, "instance-deploy", "Deploy instance", func(cli *openstack.SRegion, args *InstanceDeployOptions) error {
		return cli.DeployVM(args.ID, args.Name, args.Password, "", false, "")
	})

	type InstanceChangeConfigOptions struct {
		ID        string `help:"Instance ID"`
		FLAVOR_ID string `help:"Flavor ID"`
	}

	shellutils.R(&InstanceChangeConfigOptions{}, "instance-change-config", "Change instance config", func(cli *openstack.SRegion, args *InstanceChangeConfigOptions) error {
		instance, err := cli.GetInstance(args.ID)
		if err != nil {
			return err
		}
		return cli.ChangeConfig(instance, args.FLAVOR_ID)
	})

	type InstanceDiskOptions struct {
		ID   string `help:"Instance ID"`
		DISK string `help:"DiskId"`
	}

	shellutils.R(&InstanceDiskOptions{}, "instance-detach-disk", "Detach instance disk", func(cli *openstack.SRegion, args *InstanceDiskOptions) error {
		return cli.DetachDisk(args.ID, args.DISK)
	})

	type InstanceMigrateOptions struct {
		ID   string `help:"Instance ID"`
		Host string `help:"host id"`
	}
	type MigrationOptions struct {
		ID          string `help:"Instance ID"`
		Type        string `help:"migration choices:evacuation|live-migration|migration|resize"`
		Migrationid string `help:"migration ID"`
	}
	shellutils.R(&InstanceMigrateOptions{}, "instance-migrate", "migrate instance", func(cli *openstack.SRegion, args *InstanceMigrateOptions) error {
		return cli.MigrateVM(args.ID, args.Host)
	})

	shellutils.R(&InstanceMigrateOptions{}, "instance-live-migrate", "migrate instance", func(cli *openstack.SRegion, args *InstanceMigrateOptions) error {
		return cli.LiveMigrateVM(args.ID, args.Host)
	})

	shellutils.R(&MigrationOptions{}, "instance-migration-list", "list live migration", func(cli *openstack.SRegion, args *MigrationOptions) error {
		return cli.ListServerMigration(args.ID)
	})
	shellutils.R(&MigrationOptions{}, "instance-migration-delete", "delet migration", func(cli *openstack.SRegion, args *MigrationOptions) error {
		return cli.DeleteMigration(args.ID, args.Migrationid)
	})
	shellutils.R(&MigrationOptions{}, "instance-migration-forceComplete", "deletmigration", func(cli *openstack.SRegion, args *MigrationOptions) error {
		return cli.ForceCompleteMigration(args.ID, args.Migrationid)
	})

	shellutils.R(&MigrationOptions{}, "migrations-show", "show migrations", func(cli *openstack.SRegion, args *MigrationOptions) error {
		_, err := cli.GetMigrations(args.ID, args.Type)
		return err
	})
}
