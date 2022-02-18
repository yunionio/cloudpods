/*
 * @Author: your name
 * @Date: 2022-02-16 21:11:39
 * @LastEditTime: 2022-02-18 15:41:27
 * @LastEditors: Please set LastEditors
 * @Description: 打开koroFileHeader查看配置 进行设置: https://github.com/OBKoro1/koro1FileHeader/wiki/%E9%85%8D%E7%BD%AE
 * @FilePath: \cloudpods\pkg\multicloud\bingocloud\zone.go
 */
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
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SZone struct {
	// region      *SRegion
	DisplayName string `json:"displayName"`
	ZoneName    string `json:"zoneName"`
	ZoneState   string `json:"zoneState"`
}

func (self *SRegion) GetZones() ([]SZone, error) {
	resp, err := self.invoke("DescribeAvailabilityZones", nil)
	if err != nil {
		return nil, err
	}
	log.Errorf("resp=:%s", resp)
	//resp=:{"-xmlns":"http://ec2.amazonaws.com/doc/2009-08-15/",
	// "availabilityZoneInfo":{"item":{"displayName":"cc1","zoneName":"cc1","zoneState":"available"}}}
	result := struct {
		AvailabilityZoneInfo struct {
			Item []SZone
		}
	}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, err
	}

	return result.AvailabilityZoneInfo.Item, nil
}

func (self *SZone) GetName() string {
	return self.ZoneName
}
func (self *SZone) GetDisplayName() string {
	return self.DisplayName
}
func (self *SZone) GetState() string {
	return self.GetState()
}

// func (self *SZone) GetIRegion() cloudprovider.ICloudRegion {
// 	return self.region
// }

func (self *SZone) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(self.GetName()).CN(self.GetName())
	return table
}
