package modules

var (
	ResTagDetails ResourceManager
)

func init() {
	ResTagDetails = NewMeterManager("res_tag_detail", "res_tag_details",
		[]string{"key", "value"},
		[]string{},
	)
	register(&ResTagDetails)
}
