package modules

import "yunion.io/x/onecloud/pkg/mcclient/modulebase"

var (
	ServiceCertficatesV3 modulebase.ResourceManager
)

func init() {
	ServiceCertficatesV3 = NewIdentityV3Manager(
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

	register(&ServiceCertficatesV3)
}
