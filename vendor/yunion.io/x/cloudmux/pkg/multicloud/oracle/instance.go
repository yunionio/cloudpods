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

package oracle

import (
	"time"

	"yunion.io/x/pkg/errors"
)

type SInstance struct {
	AvailabilityDomain string
	CompartmentId      string
	DefinedTags        struct {
		OracleTags struct {
			CreatedBy string
			CreatedOn time.Time
		}
	}
	DisplayName      string
	ExtendedMetadata struct {
	}
	FaultDomain  string
	FreeformTags struct {
	}
	Id            string
	ImageId       string
	LaunchMode    string
	LaunchOptions struct {
		BootVolumeType                  string
		Firmware                        string
		NetworkType                     string
		RemoteDataVolumeType            string
		IsPvEncryptionInTransitEnabled  bool
		IsConsistentVolumeNamingEnabled bool
	}
	InstanceOptions struct {
		AreLegacyImdsEndpointsDisabled bool
	}
	AvailabilityConfig struct {
		IsLiveMigrationPreferred bool
		RecoveryAction           string
	}
	LifecycleState string
	Metadata       struct {
	}
	Region      string
	Shape       string
	ShapeConfig struct {
		Ocpus                     float64
		MemoryInGBs               float64
		ProcessorDescription      string
		NetworkingBandwidthInGbps float64
		MaxVnicAttachments        int
		Gpus                      int
		LocalDisks                int
		Vcpus                     int
	}
	IsCrossNumaNode bool
	SourceDetails   struct {
		SourceType string
		ImageId    string
	}
	SystemTags struct {
	}
	TimeCreated time.Time
	AgentConfig struct {
		IsMonitoringDisabled  bool
		IsManagementDisabled  bool
		AreAllPluginsDisabled bool
		PluginsConfig         []struct {
			Name         string
			DesiredState string
		}
	}
}

func (self *SRegion) GetInstances() ([]SInstance, error) {
	resp, err := self.list(SERVICE_IAAS, "instances", nil)
	if err != nil {
		return nil, err
	}
	ret := []SInstance{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	return ret, nil
}
