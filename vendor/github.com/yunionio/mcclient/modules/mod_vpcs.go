package modules

var (
	Vpcs ResourceManager
)

func init() {
	Vpcs = NewComputeManager("vpc", "vpcs",
		[]string{"ID", "Name", "Enabled", "Status", "Cloudregion_Id", "Is_default", "Cidr_Block"},
		[]string{})

	registerCompute(&Vpcs)
}
