package seclib2

import "testing"

func TestSplitCert(t *testing.T) {
	PEM := `-----BEGIN CERTIFICATE-----
MIIFADCCA+igAwIBAgIRAOMlOS6MEmLdT29AN1e8XfgwDQYJKoZIhvcNAQELBQAw
6vetSmRT35g6Tf/bZyPtPLnBOw4bpZtN/9KWJ5pJtKN80hgc
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIE2jCCA8KgAwIBAgIJAJlb8KChCsV+MA0GCSqGSIb3DQEBCwUAMIGfMQswCQYD
Fc5CzfQAhw57y6LmnPVoKAE/TFHvvSFNxjwSaBCGQ46FfnZMjs48a2xwHaqwAw==
-----END CERTIFICATE-----
`
	pems := splitCert([]byte(PEM))
	for i := 0; i < len(pems); i += 1 {
		t.Logf("\n%s\n", string(pems[i]))
	}
}
