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

package test

import (
	"os"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/printutils"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func TestShell() {

	type RegionResourceCount struct {
		ZoneCount             int
		VpcCount              int
		NetworkCount          int
		EipCount              int
		VmCount               int
		LbCount               int
		LbCertCount           int
		DiskCount             int
		SnapshotCount         int
		SnapshotPolicyCount   int
		ImageCount            int
		NetworkInterfaceCount int
		BucketCount           int
		RdsCount              int
	}

	type ReadonlyTestOptions struct {
		TestVpc bool `default:"true"`
		TestLb  bool `default:"true"`
	}

	var list = func(parent cloudprovider.ICloudResource, resource string, callback func() (interface{}, error)) interface{} {
		result, err := callback()
		if err != nil {
			if errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
				return result
			}
			log.Errorf("list %s error: %v", resource, err)
			os.Exit(-1)
		}
		log.Debugf("%s(%s) %s:", parent.GetName(), parent.GetGlobalId(), resource)
		printutils.PrintGetterList(result, nil)
		return result
	}

	var show = func(parent cloudprovider.ICloudResource, resource string, callback func() (interface{}, error)) interface{} {
		result, err := callback()
		if err != nil {
			if errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
				return result
			}
			log.Errorf("show %s(%s) %s error: %v", parent.GetName(), parent.GetGlobalId(), resource, err)
			os.Exit(-1)
		}
		log.Debugf("%s(%s) %s:", parent.GetName(), parent.GetGlobalId(), resource)
		printutils.PrintGetterObject(result)
		return result
	}

	shellutils.R(&ReadonlyTestOptions{}, "test-readonly", "Test read", func(cli cloudprovider.ICloudRegion, args *ReadonlyTestOptions) error {
		result := RegionResourceCount{
			ZoneCount:             0,
			VpcCount:              0,
			NetworkCount:          0,
			EipCount:              0,
			VmCount:               0,
			LbCount:               0,
			LbCertCount:           0,
			DiskCount:             0,
			SnapshotCount:         0,
			SnapshotPolicyCount:   0,
			ImageCount:            0,
			NetworkInterfaceCount: 0,
			BucketCount:           0,
		}
		if args.TestVpc {
			_vpcs := list(cli, "vpcs", func() (interface{}, error) {
				return cli.GetIVpcs()
			})
			vpcs := _vpcs.([]cloudprovider.ICloudVpc)
			result.VpcCount += len(vpcs)
			for i := range vpcs {
				_wire := list(vpcs[i], "wire", func() (interface{}, error) {
					return vpcs[i].GetIWires()
				})
				wires := _wire.([]cloudprovider.ICloudWire)
				for j := range wires {
					_networks := list(wires[j], "networks", func() (interface{}, error) {
						return wires[j].GetINetworks()
					})
					networks := _networks.([]cloudprovider.ICloudNetwork)
					result.NetworkCount += len(networks)
				}
			}
		}
		_eips := list(cli, "eips", func() (interface{}, error) {
			return cli.GetIEips()
		})
		eips := _eips.([]cloudprovider.ICloudEIP)
		result.EipCount = len(eips)
		_snapshots := list(cli, "snapshots", func() (interface{}, error) {
			return cli.GetISnapshots()
		})
		snapshots := _snapshots.([]cloudprovider.ICloudSnapshot)
		result.SnapshotCount = len(snapshots)
		_snapshotPolicies := list(cli, "snapshot policies", func() (interface{}, error) {
			return cli.GetISnapshotPolicies()
		})
		snapshotPolicies := _snapshotPolicies.([]cloudprovider.ICloudSnapshotPolicy)
		result.SnapshotPolicyCount = len(snapshotPolicies)
		_nics := list(cli, "network interfaces", func() (interface{}, error) {
			return cli.GetINetworkInterfaces()
		})
		nics := _nics.([]cloudprovider.ICloudNetworkInterface)
		result.NetworkInterfaceCount = len(nics)

		_buckets := list(cli, "buckets", func() (interface{}, error) {
			return cli.GetIBuckets()
		})
		buckets := _buckets.([]cloudprovider.ICloudBucket)
		result.BucketCount = len(buckets)
		for i := range buckets {
			list(buckets[i], "bucket objects", func() (interface{}, error) {
				return buckets[i].ListObjects("", "", "", 10)
			})
		}

		list(cli, "storages", func() (interface{}, error) {
			return cli.GetIStorages()
		})
		_caches := list(cli, "storagecaches", func() (interface{}, error) {
			return cli.GetIStoragecaches()
		})
		caches := _caches.([]cloudprovider.ICloudStoragecache)
		for i := range caches {
			_images := list(caches[i], "images", func() (interface{}, error) {
				return caches[i].GetICloudImages()
			})
			images := _images.([]cloudprovider.ICloudImage)
			result.ImageCount = len(images)
		}

		_zones := list(cli, "zones", func() (interface{}, error) {
			return cli.GetIZones()
		})
		zones := _zones.([]cloudprovider.ICloudZone)
		result.ZoneCount = len(zones)
		for i := range zones {
			_hosts := list(zones[i], "host", func() (interface{}, error) {
				return zones[i].GetIHosts()
			})
			hosts := _hosts.([]cloudprovider.ICloudHost)
			for j := range hosts {
				_vms := list(hosts[j], "vms", func() (interface{}, error) {
					return hosts[j].GetIVMs()
				})
				vms := _vms.([]cloudprovider.ICloudVM)
				result.VmCount += len(vms)
				for k := range vms {
					list(vms[k], "vm disks", func() (interface{}, error) {
						return vms[k].GetIDisks()
					})
					list(vms[k], "vm nics", func() (interface{}, error) {
						return vms[k].GetINics()
					})
					show(vms[k], "vm eip", func() (interface{}, error) {
						return vms[k].GetIEIP()
					})
				}
			}
			_storages := list(zones[i], "storages", func() (interface{}, error) {
				return zones[i].GetIStorages()
			})
			storages := _storages.([]cloudprovider.ICloudStorage)
			for j := range storages {
				_disks := list(storages[j], "disks", func() (interface{}, error) {
					return storages[j].GetIDisks()
				})
				disks := _disks.([]cloudprovider.ICloudDisk)
				result.DiskCount += len(disks)
			}
		}
		if args.TestLb {
			/*
				list(cli, "lb acls", func() (interface{}, error) {
					return cli.GetILoadBalancerAcls()
				})
			*/
			_certs := list(cli, "lb certificates", func() (interface{}, error) {
				return cli.GetILoadBalancerCertificates()
			})
			certs := _certs.([]cloudprovider.ICloudLoadbalancerCertificate)
			result.LbCertCount = len(certs)
			/*
				list(cli, "lb backend groups", func() (interface{}, error) {
					return cli.GetILoadBalancerBackendGroups()
				})
			*/
			_lbs := list(cli, "lbs", func() (interface{}, error) {
				return cli.GetILoadBalancers()
			})
			lbs := _lbs.([]cloudprovider.ICloudLoadbalancer)
			result.LbCount += len(lbs)
			for i := range lbs {
				_listeners := list(lbs[i], "lb listeners", func() (interface{}, error) {
					return lbs[i].GetILoadBalancerListeners()
				})
				listeners := _listeners.([]cloudprovider.ICloudLoadbalancerListener)
				for j := range listeners {
					list(listeners[j], "listener rules", func() (interface{}, error) {
						return listeners[j].GetILoadbalancerListenerRules()
					})
				}
				_groups := list(lbs[i], "lb backend groups", func() (interface{}, error) {
					return lbs[i].GetILoadBalancerBackendGroups()
				})
				groups := _groups.([]cloudprovider.ICloudLoadbalancerBackendGroup)
				for j := range groups {
					list(groups[j], "lb backend", func() (interface{}, error) {
						return groups[j].GetILoadbalancerBackends()
					})
				}
			}
		}

		_rds := list(cli, "rds", func() (interface{}, error) {
			return cli.GetIDBInstances()
		})
		rds := _rds.([]cloudprovider.ICloudDBInstance)
		result.RdsCount = len(rds)
		log.Println("resource sum for ", cli.GetName(), jsonutils.Marshal(result).PrettyString())
		return nil
	})
}
