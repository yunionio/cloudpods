package modules

import "yunion.io/x/onecloud/pkg/mcclient/modulebase"

var (
	BucketOptions modulebase.ResourceManager
)

func init() {
	BucketOptions = NewMeterManager("bucket_option", "bucket_options",
		[]string{"status"},
		[]string{},
	)
	register(&BucketOptions)
}
