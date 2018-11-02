package aws

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/compute/options"
)

type Cpu struct {
	Cores int    `json:"cores"`
	Units string `json:"units"`
}

type CpuCredits struct {
	OptimizationSupported bool `json:"optimizationSupported"`
}

type ProcessorFeatures struct {
	AESNI bool `json:"AES-NI"`
	AVX   bool `json:"AVX"`
	Turbo bool `json:"Turbo"`
}

type SInstanceType struct {
	Architectures          []string          `json:"architectures"`
	Cpu                    Cpu               `json:"cpu"`
	CpuCredits             CpuCredits        `json:"cpuCredits"`
	Description            string            `json:"description"`
	EbsEncryptionSupported bool              `json:"ebsEncryptionSupported"`
	EbsOnly                bool              `json:"ebsOnly"`
	Family                 string            `json:"family"`
	FreeTierEligible       bool              `json:"freeTierEligible"`
	Ipv6Support            bool              `json:"ipv6Support"`
	Memory                 float32           `json:"memory"`
	NetworkPerformance     string            `json:"networkPerformance"`
	PhysicalProcessor      string            `json:"physicalProcessor"`
	ProcessorFeatures      ProcessorFeatures `json:"processorFeatures"`
	ProcessorSpeed         float32           `json:"processorSpeed"`
	SpotSupported          bool              `json:"spotSupported"`
	InstanceTypeId         string            `json:"typeName"`
	VirtualizationTypes    []string           `json:"virtualizationTypes"`
	Vpc                    bool              `json:"vpc"`
	VpcOnly                bool              `json:"vpcOnly"`
	Windows                bool              `json:"windows"`
}

func (self *SInstanceType) memoryMB() int {
	return int(self.Memory * 1024)
}

func (self *SRegion) GetInstanceTypes() ([]SInstanceType, error) {
	if self.instanceTypes == nil {
		var GlobalInstanceTyes []SInstanceType
		instanceTypes, err := ioutil.ReadFile(options.Options.DefaultAwsInstanceTypeFile)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal([]byte(instanceTypes), &GlobalInstanceTyes)
		if err != nil {
			log.Errorf("GetInstanceTypes %s", err)
			return nil, err
		}
		return GlobalInstanceTyes, err
	} else {
		return self.instanceTypes, nil
	}
}

func (self *SRegion) GetInstanceType(instanceTypeId string) (*SInstanceType, error) {
	ret, err := self.GetInstanceTypes()
	if err != nil {
		return nil, err
	}

	for _, item := range ret {
		if item.InstanceTypeId == instanceTypeId {
			return &item, nil
		}
	}

	return nil, fmt.Errorf("instancetype %s not found", instanceTypeId)
}

func (self *SRegion) GetMatchInstanceTypes(cpu int, memMB int, gpu int, zoneId string) ([]SInstanceType, error) {
	types, err := self.GetInstanceTypes()
	if err != nil {
		return nil, err
	}

	// 实例类型顺序: 微型实例 -> 通用型 -> 计算优化型 ...
	// todo：部分实例类型 需要启用ena才能正常启动。需要过滤掉。
	// https://docs.aws.amazon.com/zh_cn/AWSEC2/latest/UserGuide/enhanced-networking-ena.html
	ret := []SInstanceType{}
	for _, t := range types {
		// cpu & mem & disk & ena 都匹配才行
		if t.Cpu.Cores == cpu && t.memoryMB() == memMB {
			ret = append(ret, t)
		}
	}

	return ret, nil
}