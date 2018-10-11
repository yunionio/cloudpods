package aws

type SInstanceType struct {
	CpuCoreCount         int
	MemorySize           float32
	EniQuantity          int // 实例规格支持网卡数量
	GPUAmount            int
	GPUSpec              string
	InstanceTypeFamily   string
	InstanceTypeId       string
	LocalStorageCategory string
	LocalStorageAmount   int
	LocalStorageCapacity int64
	InstanceBandwidthRx  int
	InstanceBandwidthTx  int
	InstancePpsRx        int
	InstancePpsTx        int
}