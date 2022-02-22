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
)

type SRouteTable struct {
	RouteTableId   string `json:"routeTableId"`
	RouteTableName string `json:"routeTableName"`
	VpcId          string `json:"vpcId"`
	RouteSet       string `json:"routeSet"`
	AssociationSet struct {
		Item struct {
			RouteTableAssociationId string `json:"routeTableAssociationId"`
			RouteTableId            string `json:"routeTableId"`
			SubnetId                string `json:"subnetId"`
			Main                    string `json:"main"`
		} `json:"item"`
	} `json:"associationSet"`
}

func (self *SRegion) GetRouterTables() ([]SRouteTable, error) {
	resp, err := self.invoke("DescribeRouteTables", nil)
	if err != nil {
		return nil, err
	}
	log.Errorf("resp=:%s", resp)
	result := struct {
		RouteTableSet struct {
			Item []SRouteTable
		}
	}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, err
	}
	return result.RouteTableSet.Item, nil

}
