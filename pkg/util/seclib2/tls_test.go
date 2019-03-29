// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
