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

package jdcloud

import (
	commodels "github.com/jdcloud-api/jdcloud-sdk-go/services/common/models"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/vm/apis"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/vm/client"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/vm/models"
)

type SInstanceType struct {
	models.InstanceType
}

func (it *SInstanceType) GetCpu() int {
	return it.Cpu
}

func (it *SInstanceType) GetMemoryMB() int {
	return it.MemoryMB
}

func (r *SRegion) InstanceTypes(instanceTypes ...string) ([]SInstanceType, error) {
	vm := "vm"
	req := apis.NewDescribeInstanceTypesRequestWithAllParams(r.ID, &vm, []commodels.Filter{
		{
			Name:   "instanceTypes",
			Values: instanceTypes,
		},
	})
	client := client.NewVmClient(r.getCredential())
	client.Logger = Logger{debug: r.client.debug}
	resp, err := client.DescribeInstanceTypes(req)
	if err != nil {
		return nil, err
	}
	its := make([]SInstanceType, len(resp.Result.InstanceTypes))
	for i := range resp.Result.InstanceTypes {
		its = append(its, SInstanceType{
			InstanceType: resp.Result.InstanceTypes[i],
		})
	}
	return its, nil
}
