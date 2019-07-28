package modules

type LoadbalancerCachedCertificateManager struct {
	ResourceManager
}

var (
	LoadbalancerCachedCertificates LoadbalancerCachedCertificateManager
)

func init() {
	LoadbalancerCachedCertificates = LoadbalancerCachedCertificateManager{
		NewComputeManager(
			"cachedloadbalancercertificate",
			"cachedloadbalancercertificates",
			[]string{
				"id",
				"certificate_id",
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
	registerCompute(&LoadbalancerCachedCertificates)
}
