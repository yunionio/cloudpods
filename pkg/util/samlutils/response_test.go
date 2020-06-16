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

package samlutils

import (
	"encoding/xml"
	"testing"
)

func TestNewResponse(t *testing.T) {
	privateKeyString := `-----BEGIN PRIVATE KEY-----
MIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQDKpaC0XoinzuQO
bt/VsWap0OVqN8O3cBFS41Ss9GGbjn1ZIg+D14OxywZdrRA17MKxfjhO4F+d2n8H
Ih5ZofGURKgOH8KrEIPcii5HLk1ZVnLXXbinExwG26OPAID9rYNSdBhVeR4pqHm7
SBnBXV+s83eIekAuPE6lIOF4irA3kOXxKhInYAygnx2fNR6p98suNda+F1VTO8HV
WoAvjJoQ0iAA6/eHmUv1gPtJ+odjJPH7m6FrF3voJIiwmgFua01qkn6tZH53hBc2
Dkbt5PxqI1KC0C79wq1ufehkk9MR5NTMxIwxbc3PcTbmwduGelKOsloLK/tGu5g8
Kq5UJE4ZAgMBAAECggEBAJmRtNybe1I4Hmm1qlkl7Fgqn4DEK8Sa3/YBowzC0ilx
bRqcDkfqjbmx0uwwl8VV3CFoNsHHlY5po7RDLd7dM9cZxIWXmg3LITKDYRi+RQ27
zqHZO3MZrzafQi6/wgD8ejWFF1/Gvo3xR/ceZ6461aOaie5aPsMLHspSxat05p/k
C4NkBvjHdYt4PiH873PLOI6mpt8N5mnkYmer8S5XX02GyQI4OGnHuCqQsHtlhgj7
XeSB2biDNrcYfL1Kp1E8vEGBYqhohVWVWo+rP5zyoWNxQUXOVDE5zQanLrktjno5
uzHkK1zKGcOlh7s40pV3qgcQFRTWt4RRP5wBasKc9wECgYEA948ikAVBiAiEPDha
hlNboaUKYeB6HxKSUZhqaGY0+7OWSyBhQBALCZRk4ccscysj4VUur+6FwC7lleRj
tG+S7ukyiQEeH0c0NAfnIlSd+LFQREsj4MVLn+Sl/8edNIl/UH+69pDtVU6Az/2u
DF0FloGL5XZ0WLIVcRZdUjt8HfkCgYEA0Y54DPN5PcLKci0Plioc0qLb33kp6F+w
JJhOLJv+tbmeNVA0EM+BwTHlHF8MRsreybwSHoUYX61vkR2AlGLo8D5TlTEA1VSD
k4w5Yic3cjQzOFMWQczpGdBwV37inHOU2phOp2kKs8LyT/gJw5f6b34R7zfRc8ry
uAEXaTg4OSECgYBb37ESBgFV/OMmfjuKUnFVQiziOi7YTUokIg6LhDLxnqqOYwv0
fH+8JGh0Kjji3QXJ4JUdEcZtlnn58PLXyfib1cu9cL6/GOvUy4IKCaE+5H9HeSNt
jYsNYgwBKxG6p7SqKV03mH2cBTBlAF6RlAw42QcUN6viJuUyPPyRQiZD8QKBgQCO
N28X8wC8Pn9gH16tnaTz+pzXvAYJ8y66lzaupaumLvPE4MqFAh7gO3lu2L6fKL0s
EdwGJHOXM0A9LtV9XucRbGsTHC+hl/q33vluuIizk+OS/ShkvakQ4NntN2qZnQNP
mv/+M5aUyt/iD8aonHLUya1oOOyH9hrlb7Aws3vMoQKBgQDNH4xxS7wxnf7GcTnt
96tHvjBVHXa0SK1XkbFjgIEDU+xeEUteYs/9jW4AMA/lqsfBXUwhF3JAVoRUWm0M
d95jkAi6q2icuGogkftyv+LgUvhSUm6N+8eQ+YvQfgsNBqxlSudjpP0V+S2v73vp
8egFWKkL+H0upowTbr0T0OUfog==
-----END PRIVATE KEY-----
`
	certString := `MIIE/DCCA+SgAwIBAgIQQHdQar3faUJLgHcCgO4z3zANBgkqhkiG9w0BAQsFADCB
nzELMAkGA1UEBhMCQ04xEDAOBgNVBAgMB0JlaWppbmcxETAPBgNVBAcMCENoYW95
YW5nMSMwIQYDVQQKDBpZdW5pb24gVGVjaG5vbG9neSBDby4gTHRkLjERMA8GA1UE
CwwIT25lQ2xvdWQxFDASBgNVBAMMC0AxNTkxNzIxNDc5MR0wGwYJKoZIhvcNAQkB
Fg5pbmZvQHl1bmlvbi5jbjAeFw0yMDA2MDkxNjUxMjBaFw0yMjA4MTgxNjUxMjBa
MIGXMQswCQYDVQQGEwJDTjEQMA4GA1UECAwHQmVpamluZzERMA8GA1UEBwwIQ2hh
b3lhbmcxIzAhBgNVBAoMGll1bmlvbiBUZWNobm9sb2d5IENvLiBMdGQuMREwDwYD
VQQLDAhPbmVDbG91ZDEMMAoGA1UEAwwDaWRwMR0wGwYJKoZIhvcNAQkBFg5pbmZv
QHl1bmlvbi5jbjCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAMqloLRe
iKfO5A5u39WxZqnQ5Wo3w7dwEVLjVKz0YZuOfVkiD4PXg7HLBl2tEDXswrF+OE7g
X53afwciHlmh8ZREqA4fwqsQg9yKLkcuTVlWctdduKcTHAbbo48AgP2tg1J0GFV5
HimoebtIGcFdX6zzd4h6QC48TqUg4XiKsDeQ5fEqEidgDKCfHZ81Hqn3yy411r4X
VVM7wdVagC+MmhDSIADr94eZS/WA+0n6h2Mk8fuboWsXe+gkiLCaAW5rTWqSfq1k
fneEFzYORu3k/GojUoLQLv3CrW596GST0xHk1MzEjDFtzc9xNubB24Z6Uo6yWgsr
+0a7mDwqrlQkThkCAwEAAaOCATgwggE0MAkGA1UdEwQCMAAwHQYDVR0OBBYEFFie
B9nsyKQ0A+oUWqiJkkURLJMqMIHUBgNVHSMEgcwwgcmAFISskqtTpDD4otifwvuG
bFwyvZ0zoYGlpIGiMIGfMQswCQYDVQQGEwJDTjEQMA4GA1UECAwHQmVpamluZzER
MA8GA1UEBwwIQ2hhb3lhbmcxIzAhBgNVBAoMGll1bmlvbiBUZWNobm9sb2d5IENv
LiBMdGQuMREwDwYDVQQLDAhPbmVDbG91ZDEUMBIGA1UEAwwLQDE1OTE3MjE0Nzkx
HTAbBgkqhkiG9w0BCQEWDmluZm9AeXVuaW9uLmNuggkA7uFZmhG6rz0wEwYDVR0l
BAwwCgYIKwYBBQUHAwEwCwYDVR0PBAQDAgWgMA8GA1UdEQQIMAaHBH8AAAEwDQYJ
KoZIhvcNAQELBQADggEBAKc7gUyrAMto3O8/Qi/m23aHxYc33GVMRrDPNS7asSWO
l+3IiWc2L61OiWhmK75t1leMRZU5hEuxAb3Rq6TwHVQWD3mAuIrss+Dsfhs0Y/wa
miIYV+v4rXuu3CusBxQo8NRuNW+f/F5cR0WVBDDMm5NDQKPVdeQ5uszqtWN62tPq
zvi1nRqaRKZQbzENGmqLwfMQCm0fsUzw6mdZb137S04MuSnvb1hTyhQ492uqoTlE
KM6qMM9M9PWLW8PiZxiHxtLdO5BIXOXY//xU9xD58sOIAePuc322+TbMBtRZAyqg
OZ+tB0Nzk2RJp4luPEqxoBcDVpcuiyfu3s3Jtnr3Lns=`

	input := SSAMLResponseInput{
		NameId:                      "testUser",
		NameIdFormat:                NAME_ID_FORMAT_TRANSIENT,
		RequestID:                   "_dck4mm08qmdhc8k4nuir07hghetdqqg8umg5",
		RequestEntityId:             "https://auth.huaweicloud.com/",
		IssuerEntityId:              "https://saml.yunion.io/",
		IssuerCertString:            certString,
		AssertionConsumerServiceURL: "https://auth.huaweicloud.com/authui/saml/SAMLAssertionConsumer",
	}
	resp := NewResponse(input)
	resp.AddAudienceRestriction(input.RequestEntityId)
	for k, v := range map[string]string{
		"xUserId":    "userXXXX",
		"xAccountId": "accountXXXX",
		"bpId":       "bpXXXXX",
		"email":      "xxxx@yunion.io",
		"name":       "xxxx",
		"mobile":     "13812341234",
	} {
		resp.AddAttribute(k, k, "urn:oasis:names:tc:SAML:2.0:attrname-format:uri", []string{v})
	}

	respXml, err := xml.MarshalIndent(resp, "", "  ")
	if err != nil {
		t.Fatalf("xml.MarshalIndent fail %s", err)
	}

	privateKey, err := decodePrivateKey([]byte(privateKeyString))
	if err != nil {
		t.Fatalf("decodePrivateKey fail %s", err)
	}

	signed, err := SignXML(string(respXml), privateKey)
	if err != nil {
		t.Fatalf("SignXML fail %s", err)
	}

	t.Logf("signed XML: %s", signed)

	validXMLs, err := ValidateXML(signed)
	if err != nil {
		t.Fatalf("ValidateXML fail %s", err)
	}

	t.Logf("validated xml: %s", validXMLs)
}
