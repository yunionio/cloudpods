package modules

var (
	SecGroupRules ResourceManager
)

func init() {
	SecGroupRules = NewComputeManager("secgrouprule", "secgrouprules",
		[]string{"ID", "Name", "Direction",
			"Action", "Protocol", "Ports", "Priority",
			"Cidr", "Description"},
		[]string{"SecGroups"})

	registerCompute(&SecGroupRules)
}
