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

package bingocloud

import (
	"fmt"
	"net/url"
	"time"
)

type SInstance struct {
	ReservationId string `json:"reservationId"`
	OwnerId       string
	GroupSet      struct {
		Item struct {
			GroupId   string
			GroupName string
		}
	}
	InstancesSet struct {
		Item struct {
			InstanceId    string `json:"instanceId"`
			InstanceName  string `json:"instanceName"`
			HostName      string `json:"hostName"`
			ImageId       string `json:"imageId"`
			InstanceState struct {
				Code            int    `json:"code"`
				Name            string `json:"name"`
				PendingProgress string `json:"pendingProgress"`
			} `json:"instanceState"`
			PrivateDNSName     string `json:"privateDnsName"`
			DNSName            string `json:"dnsName"`
			PrivateIPAddress   string `json:"privateIpAddress"`
			PrivateIPAddresses string `json:"privateIpAddresses"`
			IPAddress          string `json:"ipAddress"`
			NifInfo            string `json:"nifInfo"`
			KeyName            string `json:"keyName"`
			AmiLaunchIndex     int    `json:"amiLaunchIndex"`
			ProductCodesSet    struct {
				Item struct {
					ProductCode string `json:"productCode"`
				} `json:"item"`
			} `json:"productCodesSet"`
			InstanceType   string    `json:"instanceType"`
			VmtypeCPU      int       `json:"vmtype_cpu"`
			VmtypeMem      int       `json:"vmtype_mem"`
			VmtypeDisk     int       `json:"vmtype_disk"`
			VmtypeGpu      int       `json:"vmtype_gpu"`
			VmtypeSsd      int       `json:"vmtype_ssd"`
			VmtypeHdd      int       `json:"vmtype_hdd"`
			VmtypeHba      int       `json:"vmtype_hba"`
			VmtypeSriov    int       `json:"vmtype_sriov"`
			LaunchTime     time.Time `json:"launchTime"`
			RootDeviceType string    `json:"rootDeviceType"`
			HostAddress    string    `json:"hostAddress"`
			Platform       string    `json:"platform"`
			UseCompactMode bool      `json:"useCompactMode"`
			ExtendDisk     bool      `json:"extendDisk"`
			Placement      struct {
				AvailabilityZone string `json:"availabilityZone"`
			} `json:"placement"`
			Namespace    string `json:"namespace"`
			KernelId     string `json:"kernelId"`
			RamdiskId    string `json:"ramdiskId"`
			OperName     string `json:"operName"`
			OperProgress int    `json:"operProgress"`
			Features     string `json:"features"`
			Monitoring   struct {
				State string `json:"state"`
			} `json:"monitoring"`
			SubnetId              string    `json:"subnetId"`
			VpcId                 string    `json:"vpcId"`
			StorageId             string    `json:"storageId"`
			DisableAPITermination bool      `json:"disableApiTermination"`
			Vncdisabled           bool      `json:"vncdisabled"`
			StartTime             time.Time `json:"startTime"`
			CustomStatus          string    `json:"customStatus"`
			SystemStatus          int       `json:"systemStatus"`
			NetworkStatus         int       `json:"networkStatus"`
			ScheduleTags          string    `json:"scheduleTags"`
			StorageScheduleTags   string    `json:"storageScheduleTags"`
			IsEncrypt             bool      `json:"isEncrypt"`
			IsImported            bool      `json:"isImported"`
			Ec2Version            string    `json:"ec2Version"`
			Passphrase            string    `json:"passphrase"`
			DrsEnabled            bool      `json:"drs_enabled"`
			LaunchPriority        int       `json:"launchPriority"`
			CPUPriority           int       `json:"cpuPriority"`
			MemPriority           int       `json:"memPriority"`
			CPUQuota              int       `json:"cpuQuota"`
			AutoMigrate           bool      `json:"autoMigrate"`
			DrMirrorId            string    `json:"drMirrorId"`
			BlockDeviceMapping    struct {
				Item struct {
					DeviceName string `json:"deviceName"`
					Ebs        struct {
						AttachTime          time.Time `json:"attachTime"`
						DeleteOnTermination bool      `json:"deleteOnTermination"`
						Status              string    `json:"status"`
						VolumeId            string    `json:"volumeId"`
						Size                int       `json:"size"`
					} `json:"ebs"`
				} `json:"item"`
			} `json:"blockDeviceMapping"`
			EnableLiveScaleup bool   `json:"enableLiveScaleup"`
			ImageBytes        int64  `json:"imageBytes"`
			StatusReason      string `json:"statusReason"`
			Hypervisor        string `json:"hypervisor"`
			Bootloader        string `json:"bootloader"`
			BmMachineId       string `json:"bmMachineId"`
		}
	}
}

func (self *SRegion) DescribeInstances(id string, maxResult int, nextToken string) ([]SInstance, string, error) {
	params := map[string]string{}
	if len(id) > 0 {
		params["instanceId"] = id
	}
	if maxResult > 0 {
		params["maxRecords"] = fmt.Sprintf("%d", maxResult)
	}
	if len(nextToken) > 0 {
		params["nextToken"] = nextToken
	}
	// resp, err := self.invoke("DescribeInstances", params)
	resp, err := self.invoke("DescribeInstanceHosts", params)

	if err != nil {
		return nil, "", err
	}
	result := struct {
		NextToken      string
		ReservationSet struct {
			Item []SInstance
		}
	}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, "", err
	}
	return result.ReservationSet.Item, result.NextToken, nil
}

////////
func (self *SRegion) GetInstance(id string) (*SInstance, error) {
	vm := &SInstance{}
	params := url.Values{}
	params.Set("include_vm_disk_config", "true")
	params.Set("include_vm_nic_config", "true")
	return vm, self.get("vms", id, params, vm)
}
