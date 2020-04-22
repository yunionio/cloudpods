package modules

import "yunion.io/x/onecloud/pkg/mcclient/modulebase"

var (
	ServiceCertificatesV3 modulebase.ResourceManager
)

func init() {
	ServiceCertificatesV3 = NewIdentityV3Manager(
		"servicecertificate", "servicecertificates",
		[]string{
			"id",
			"name",
			"algorithm",
			"fingerprint",
			"not_before",
			"not_after",
			"common_name",
			"subject_alternative_names",
		},
		[]string{"tenant"},
	)

	register(&ServiceCertificatesV3)
}
