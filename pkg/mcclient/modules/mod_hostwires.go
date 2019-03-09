package modules

var (
	Hostwires JointResourceManager
)

func init() {
	Hostwires = NewJointComputeManager("hostwire", "hostwires",
		[]string{"Host_ID", "Host", "Wire_ID", "Wire",
			"Bridge", "Interface", "Mac_addr", "is_master"},
		[]string{},
		&Hosts,
		&Wires)
	registerCompute(&Hostwires)
}
