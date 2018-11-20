package modules

var (
	Cloudmeta ResourceManager
)

func init() {
	Cloudmeta = NewCloudmetaManager("cloudmeta", "cloudmetas",
		[]string{},
		[]string{})

	register(&Cloudmeta)
}
