package modules

type LoadbalancerCertificateManager struct {
	ResourceManager
}

var (
	LoadbalancerCertificates LoadbalancerCertificateManager
)

func init() {
	LoadbalancerCertificates = LoadbalancerCertificateManager{
		NewComputeManager(
			"loadbalancercertificate",
			"loadbalancercertificates",
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
		),
	}
	registerCompute(&LoadbalancerCertificates)
}
