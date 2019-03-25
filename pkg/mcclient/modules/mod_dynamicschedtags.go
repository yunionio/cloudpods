package modules

var (
	Dynamicschedtags ResourceManager
)

func init() {
	Dynamicschedtags = NewComputeManager("dynamicschedtag", "dynamicschedtags",
		[]string{
			"ID", "Name", "Description", "Condition", "Schedtag",
			"Schedtag_Id", "Resource_Type", "Enabled"},
		[]string{})

	registerComputeV2(&Dynamicschedtags)
}
