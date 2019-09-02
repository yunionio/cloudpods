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

package validators

import (
	"testing"
)

func TestSSHKeyValidator(t *testing.T) {
	aKey := `
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAvWB7GIQ5nuffEtS5L5fPvkBP8MjOLuuIkn+G+BS8HDhWkXr4
jYokpPz/mXwdk2VmJONQw28XmAvJTLyT6xZPNOehBENm6VNakY7PJ4fXAoHFNMaF
crkj3KmwjQXQAEX6Ul7vOVl8wRBMWQ3iQiR2OKvnIkrWZF7Y4lEQVJkHTPzp7GLv
LjmiOEnOZdtrM6YtrRtIdfEk/vGDJL86xOAWmM7vMFrH9obuhyXwqtomGn+4II3C
j4ShMWKQkmHoOOmd14At+fhHlKkvtAOtDOlCXB/svKSvqBfzjbYORHCtXXc8WvID
yz2OCPgh9VgxAIcgfEt4cKvtZjz3hlcCgIY0qQIDAQABAoIBAQCPNCQhd+typHhl
bwLSYIQxo8RPmimABY/y6AiSFGvjEx8zR8Aol+v5728BC3/589V304VBJAK9cTw5
kOhx/x7KLNXvuWBa1DNKmqk/hVMrjCIqNGy5QhNCS/c7zMdrTX9rRmqz/V1/SOnS
9dLAnX3ggO15WwogQDDVguNMdaO1rMWtO/DZS53rVbhSxfM5JPq2oxAucDKxlhjT
MGdF/iK3/NyRft36QEbBB9qG+97YvXUmXFF2UVFAro9k+nWzR7BLGhkcDRRc5FsK
R/Vixe3G8pR9/XBs6MyWK/HlGBEvzdlehyEw1duGxEUNdBALcVtZXTZ37L6pn8G9
yHSvzzcBAoGBAOPUGPC77CBR9d0YP+YydkTjayV+p55k1vhrW/uvlSKZO/LwZ5b5
T/2qcmWt2SiXE6A6YUOwq2NJvBPeq+6WJBVqQoPsiVg7mc0j5/nP0TtH2GGgfPBm
JKCG+bdrh5OcVkql5fdX35zLSDkKNnxASS4bEAISdViBelmPN/QH7ayxAoGBANTL
MtnwhKj9Eas7yNfV6H1Va6AUuB3nx+ZDcFq/mPNzPE15Ddv3gqoQqi9doXe61oGx
X4XfMiKBlo5Jzh2u5LcO6fAmV1itCi2pyWcKk8Nc+Aarc36HsOLao4sw1zczCVyt
BF0TlxKeRpqBT/uQfmEThwbN8LUyoBKGl2oBWiV5AoGAHpW0m2y+8D/Qf9PnkCGq
Gulk0u3D1tG2wja3bHxPywtDLwPzBCOIB4fAP8Is6vQNIG916z5mY7fcVdaIwkjJ
o05Wi5tPfNbTeOSfGbw6XHjypXiEDUnJFPvJvkPjOX+9XdwTmTbkwAnSMkYatmdy
64uahIyx0CXhpPBDFLGTyKECgYAO4XLZ6MbuJlxg9BpUdaH/ecS/+hLyDG5fPOIT
hoiEpc9Wv5tngYSCrg2oqEyNWeR8R1Idw4D3BsbnhmPCkaNu5b0YTSYYjmlCzjfG
W+f/ZnX1yXGXLJgDFTUQm8bBFnGWKIdAlwkehTD8xwQ33F/qG/p6UFZ/5V1qTj0y
bYvHSQKBgBwzshyuDA/QSxSVDn2HI3hK1202eAN2PERsBGP2VSAEpIwav2KvBVva
p30+rx5gwUquGpB24gyHlZ0l3eVbONop84wOS8eoA4wUyXBlkgzvZvlFvJCpm8x1
qtvUqlXM7TheLX3gGucB76fmc+wLs06QPHd0sxAlTGcwwBVOUPvH
-----END RSA PRIVATE KEY-----
`
	cases := []*C{
		{
			Name:      "missing non-optional",
			In:        `{}`,
			Out:       `{}`,
			Optional:  false,
			Err:       ERR_MISSING_KEY,
			ValueWant: "",
		},
		{
			Name:      "missing optional",
			In:        `{}`,
			Out:       `{}`,
			Optional:  true,
			ValueWant: "",
		},
		{
			Name:      "missing with default",
			In:        `{}`,
			Out:       `{s: "` + aKey + `"}`,
			Default:   aKey,
			ValueWant: aKey,
		},
		{
			Name:      "good in",
			In:        `{"s": "` + aKey + `"}`,
			Out:       `{"s": "` + aKey + `"}`,
			ValueWant: aKey,
		},
		{
			Name:      "bad in",
			In:        `{"s": "0"}`,
			Out:       `{"s": "0"}`,
			Err:       ERR_INVALID_VALUE,
			ValueWant: "",
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			v := NewSSHKeyValidator("s")
			if c.Default != nil {
				s := c.Default.(string)
				v.Default(s)
			}
			if c.Optional {
				v.Optional(true)
			}
			testS(t, v, c)
		})
	}
}
