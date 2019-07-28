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

package aws

import "testing"

type testPublicKey struct {
	publickey   string
	fingerprint string
}

func TestMd5Fingerprint(t *testing.T) {
	rsa := testPublicKey{
		"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCBBuv9nsAGNpKVulxNc7zHXEEyiTqYU8J6sTfmB9lmrRea/RO/pUJg1ZGlHKLSbZ5h+d4mquASf8K3s3SQtz/4sBHroRijanO16i0Rk6t5kwcIRzaf11NiImImKgwNiCwZyiK2egAfsjDVEi8H+kSRA0N0PMxRfwOEZ/hNtVaNV7/MwkXylOuWUikGvPpm3sRmelfQoS3Hf055WM1m6POgddbjucq9bjQDW1O4dfDkWuX+385EOtfCBPtfeiAcOBBd+qEjmdfxroQwxHXLkZH7rdoS9jss3fi9P/K0ZpBKswKsed2sxKo9NNYfTDN19Kv8NBOW8W7MxN1po/2gvbd/",
		"4c:ae:76:94:fb:59:66:8c:a6:07:e2:54:2f:14:19:c5",
	}

	testKeys := []testPublicKey{rsa}

	for _, k := range testKeys {
		fingerprint, err := md5Fingerprint(k.publickey)
		if err != nil {
			t.Error(err)
			continue
		}

		if fingerprint != k.fingerprint {
			t.Errorf("ssh-rsa fingerprint is not as expected.%s != %s", fingerprint, k.fingerprint)
			continue
		}
	}
}
