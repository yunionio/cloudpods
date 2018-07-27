package modules

var (
	Wires ResourceManager
)

func init() {
	Wires = NewComputeManager("wire", "wires",
		[]string{"ID", "Name", "Bandwidth", "Zone_ID",
			"Zone", "Networks", "VPC", "VPC_ID"},
		[]string{})

	registerCompute(&Wires)
}
