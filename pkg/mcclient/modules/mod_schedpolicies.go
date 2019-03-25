package modules

var (
	Schedpolicies ResourceManager
)

func init() {
	Schedpolicies = NewComputeManager("schedpolicy", "schedpolicies",
		[]string{
			"ID", "Name", "Description", "Condition", "Schedtag",
			"Resource_Type", "Schedtag_Id", "Strategy", "Enabled",
		},
		[]string{})

	registerComputeV2(&Schedpolicies)
}
