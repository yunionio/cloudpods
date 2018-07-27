package modules

var (
	SecGroups ResourceManager
)

func init() {
	SecGroups = NewComputeManager("secgroup", "secgroups",
		[]string{"ID", "Name", "Rules",
			"Is_public", "Created_at",
			"Guest_cnt", "Description"},
		[]string{})

	registerCompute(&SecGroups)
}
