package modules

var (
	Dynamicschedtags ResourceManager
)

func init() {
	Dynamicschedtags = NewComputeManager("dynamicschedtag", "dynamicschedtags",
		[]string{"ID", "Name", "Description",
			"Condition", "Schedtag", "Schedtag_Id", "Enabled"},
		[]string{})

	registerComputeV2(&Dynamicschedtags)
}
