package qcloud

import "yunion.io/x/log"

// "time"

// {"CpuCoreCount":1,"EniQuantity":1,"GPUAmount":0,"GPUSpec":"","InstanceTypeFamily":"ecs.t1","InstanceTypeId":"ecs.t1.xsmall","LocalStorageCategory":"","MemorySize":0.500000}
// InstanceBandwidthRx":26214400,"InstanceBandwidthTx":26214400,"InstancePpsRx":4500000,"InstancePpsTx":4500000

type SInstanceType struct {
	Zone              string //	可用区。
	InstanceType      string //	实例机型。
	InstanceFamily    string //	实例机型系列。
	GPU               int    //	GPU核数，单位：核。
	CPU               int    //	CPU核数，单位：核。
	Memory            int    //	内存容量，单位：GB。
	CbsSupport        string //	是否支持云硬盘。取值范围：TRUE：表示支持云硬盘；FALSE：表示不支持云硬盘。
	InstanceTypeState string //	机型状态。取值范围：AVAILABLE：表示机型可用；UNAVAILABLE：表示机型不可用。
}

func (self *SRegion) GetInstanceTypes() ([]SInstanceType, error) {
	params := make(map[string]string)
	params["Region"] = self.Region

	body, err := self.cvmRequest("DescribeInstanceTypeConfigs", params)
	if err != nil {
		log.Errorf("DescribeInstanceTypeConfigs fail %s", err)
		return nil, err
	}

	instanceTypes := make([]SInstanceType, 0)
	err = body.Unmarshal(&instanceTypes, "InstanceTypeConfigSet")
	if err != nil {
		log.Errorf("Unmarshal instance type details fail %s", err)
		return nil, err
	}
	return instanceTypes, nil
}

func (self *SInstanceType) memoryMB() int {
	return int(self.Memory * 1024)
}
