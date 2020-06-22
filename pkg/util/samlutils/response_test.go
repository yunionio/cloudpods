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
	"strings"
	"testing"
)

var (
	privateKeyString = `-----BEGIN PRIVATE KEY-----
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
	certString = `MIIE/DCCA+SgAwIBAgIQQHdQar3faUJLgHcCgO4z3zANBgkqhkiG9w0BAQsFADCB
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
)

func TestNewResponse(t *testing.T) {
	input := SSAMLResponseInput{
		// NameId:                      "testUser",
		// NameIdFormat:                NAME_ID_FORMAT_TRANSIENT,
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

var (
	encryptedResp = `<?xml version="1.0" encoding="UTF-8"?>
<saml2p:Response Destination="https://saml.yunion.io/SAML/sp/acs" ID="_b7af65d18431cf6e58bbe17db99ed3b5" IssueInstant="2020-06-18T19:39:51.753Z" Version="2.0" xmlns:saml2p="urn:oasis:names:tc:SAML:2.0:protocol"><saml2:Issuer xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion">https://samltest.id/saml/idp</saml2:Issuer><ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#"><ds:SignedInfo><ds:CanonicalizationMethod Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/><ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/><ds:Reference URI="#_b7af65d18431cf6e58bbe17db99ed3b5"><ds:Transforms><ds:Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"/><ds:Transform Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/></ds:Transforms><ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/><ds:DigestValue>RE6No535ZL120aHLfko+nmZmphMma53rPfyEbenqxQU=</ds:DigestValue></ds:Reference></ds:SignedInfo><ds:SignatureValue>EjPgOQ4oufcqcTZu9VaOlEcZUmvEZa3fOYAWxvLMMjmx5OcGfhk8Xs9Dsyy4UOcdNXL+4vQx0ZfglBFXCLHPCO209V0oAhwRK9g0VMH5RAvgKJIH0pEU7ThHq66AMYqgKumwAE3APzMzDapI+hLvyH6wWYQNxjqOCYlGQ1Y8CNjPu11XxFzgIG0VgcHHaDL5ovAVZThrgf9M8iO0rnMpo/BmeetjOd624aws9ajZMdhNbeNeswHukvg70YmfbS0PoheYfJneeMtIvywWOfBSArOCSbwAXpCnwKVSNCYNQifVUTnJVt27QEai4LhsGYlZVeu4ilwYbMo9U+iVP27S2g==</ds:SignatureValue><ds:KeyInfo><ds:X509Data><ds:X509Certificate>MIIDEjCCAfqgAwIBAgIVAMECQ1tjghafm5OxWDh9hwZfxthWMA0GCSqGSIb3DQEBCwUAMBYxFDAS
BgNVBAMMC3NhbWx0ZXN0LmlkMB4XDTE4MDgyNDIxMTQwOVoXDTM4MDgyNDIxMTQwOVowFjEUMBIG
A1UEAwwLc2FtbHRlc3QuaWQwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC0Z4QX1NFK
s71ufbQwoQoW7qkNAJRIANGA4iM0ThYghul3pC+FwrGv37aTxWXfA1UG9njKbbDreiDAZKngCgyj
xj0uJ4lArgkr4AOEjj5zXA81uGHARfUBctvQcsZpBIxDOvUUImAl+3NqLgMGF2fktxMG7kX3GEVN
c1klbN3dfYsaw5dUrw25DheL9np7G/+28GwHPvLb4aptOiONbCaVvh9UMHEA9F7c0zfF/cL5fOpd
Va54wTI0u12CsFKt78h6lEGG5jUs/qX9clZncJM7EFkN3imPPy+0HC8nspXiH/MZW8o2cqWRkrw3
MzBZW3Ojk5nQj40V6NUbjb7kfejzAgMBAAGjVzBVMB0GA1UdDgQWBBQT6Y9J3Tw/hOGc8PNV7JEE
4k2ZNTA0BgNVHREELTArggtzYW1sdGVzdC5pZIYcaHR0cHM6Ly9zYW1sdGVzdC5pZC9zYW1sL2lk
cDANBgkqhkiG9w0BAQsFAAOCAQEASk3guKfTkVhEaIVvxEPNR2w3vWt3fwmwJCccW98XXLWgNbu3
YaMb2RSn7Th4p3h+mfyk2don6au7Uyzc1Jd39RNv80TG5iQoxfCgphy1FYmmdaSfO8wvDtHTTNiL
ArAxOYtzfYbzb5QrNNH/gQEN8RJaEf/g/1GTw9x/103dSMK0RXtl+fRs2nblD1JJKSQ3AdhxK/we
P3aUPtLxVVJ9wMOQOfcy02l+hHMb6uAjsPOpOVKqi3M8XmcUZOpx4swtgGdeoSpeRyrtMvRwdcci
NBp9UZome44qZAYH1iqrpmmjsfI9pJItsgWu3kXPjhSfj1AJGR1l9JGvJrHki1iHTA==</ds:X509Certificate></ds:X509Data></ds:KeyInfo></ds:Signature><saml2p:Status xmlns:saml2p="urn:oasis:names:tc:SAML:2.0:protocol"><saml2p:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/></saml2p:Status><saml2:EncryptedAssertion xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion"><xenc:EncryptedData Id="_492394299626c2d83feb08663b41fcb0" Type="http://www.w3.org/2001/04/xmlenc#Element" xmlns:xenc="http://www.w3.org/2001/04/xmlenc#"><xenc:EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#aes128-cbc" xmlns:xenc="http://www.w3.org/2001/04/xmlenc#"/><ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#"><xenc:EncryptedKey Id="_4852b3b33fdfd4ab0e1181fc15b1b1d3" Recipient="https://saml.yunion.io" xmlns:xenc="http://www.w3.org/2001/04/xmlenc#"><xenc:EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#rsa-oaep-mgf1p" xmlns:xenc="http://www.w3.org/2001/04/xmlenc#"><ds:DigestMethod Algorithm="http://www.w3.org/2000/09/xmldsig#sha1" xmlns:ds="http://www.w3.org/2000/09/xmldsig#"/></xenc:EncryptionMethod><ds:KeyInfo><ds:X509Data><ds:X509Certificate>MIIE/DCCA+SgAwIBAgIQQHdQar3faUJLgHcCgO4z3zANBgkqhkiG9w0BAQsFADCBnzELMAkGA1UE
BhMCQ04xEDAOBgNVBAgMB0JlaWppbmcxETAPBgNVBAcMCENoYW95YW5nMSMwIQYDVQQKDBpZdW5p
b24gVGVjaG5vbG9neSBDby4gTHRkLjERMA8GA1UECwwIT25lQ2xvdWQxFDASBgNVBAMMC0AxNTkx
NzIxNDc5MR0wGwYJKoZIhvcNAQkBFg5pbmZvQHl1bmlvbi5jbjAeFw0yMDA2MDkxNjUxMjBaFw0y
MjA4MTgxNjUxMjBaMIGXMQswCQYDVQQGEwJDTjEQMA4GA1UECAwHQmVpamluZzERMA8GA1UEBwwI
Q2hhb3lhbmcxIzAhBgNVBAoMGll1bmlvbiBUZWNobm9sb2d5IENvLiBMdGQuMREwDwYDVQQLDAhP
bmVDbG91ZDEMMAoGA1UEAwwDaWRwMR0wGwYJKoZIhvcNAQkBFg5pbmZvQHl1bmlvbi5jbjCCASIw
DQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAMqloLReiKfO5A5u39WxZqnQ5Wo3w7dwEVLjVKz0
YZuOfVkiD4PXg7HLBl2tEDXswrF+OE7gX53afwciHlmh8ZREqA4fwqsQg9yKLkcuTVlWctdduKcT
HAbbo48AgP2tg1J0GFV5HimoebtIGcFdX6zzd4h6QC48TqUg4XiKsDeQ5fEqEidgDKCfHZ81Hqn3
yy411r4XVVM7wdVagC+MmhDSIADr94eZS/WA+0n6h2Mk8fuboWsXe+gkiLCaAW5rTWqSfq1kfneE
FzYORu3k/GojUoLQLv3CrW596GST0xHk1MzEjDFtzc9xNubB24Z6Uo6yWgsr+0a7mDwqrlQkThkC
AwEAAaOCATgwggE0MAkGA1UdEwQCMAAwHQYDVR0OBBYEFFieB9nsyKQ0A+oUWqiJkkURLJMqMIHU
BgNVHSMEgcwwgcmAFISskqtTpDD4otifwvuGbFwyvZ0zoYGlpIGiMIGfMQswCQYDVQQGEwJDTjEQ
MA4GA1UECAwHQmVpamluZzERMA8GA1UEBwwIQ2hhb3lhbmcxIzAhBgNVBAoMGll1bmlvbiBUZWNo
bm9sb2d5IENvLiBMdGQuMREwDwYDVQQLDAhPbmVDbG91ZDEUMBIGA1UEAwwLQDE1OTE3MjE0Nzkx
HTAbBgkqhkiG9w0BCQEWDmluZm9AeXVuaW9uLmNuggkA7uFZmhG6rz0wEwYDVR0lBAwwCgYIKwYB
BQUHAwEwCwYDVR0PBAQDAgWgMA8GA1UdEQQIMAaHBH8AAAEwDQYJKoZIhvcNAQELBQADggEBAKc7
gUyrAMto3O8/Qi/m23aHxYc33GVMRrDPNS7asSWOl+3IiWc2L61OiWhmK75t1leMRZU5hEuxAb3R
q6TwHVQWD3mAuIrss+Dsfhs0Y/wamiIYV+v4rXuu3CusBxQo8NRuNW+f/F5cR0WVBDDMm5NDQKPV
deQ5uszqtWN62tPqzvi1nRqaRKZQbzENGmqLwfMQCm0fsUzw6mdZb137S04MuSnvb1hTyhQ492uq
oTlEKM6qMM9M9PWLW8PiZxiHxtLdO5BIXOXY//xU9xD58sOIAePuc322+TbMBtRZAyqgOZ+tB0Nz
k2RJp4luPEqxoBcDVpcuiyfu3s3Jtnr3Lns=</ds:X509Certificate></ds:X509Data></ds:KeyInfo><xenc:CipherData xmlns:xenc="http://www.w3.org/2001/04/xmlenc#"><xenc:CipherValue>ZwBIXvZM7t168N7l10Kt6FxGuF9dpHD876iktHrkfx9Tp+bxcRFGizdFV/BA1KD7NF1/04OnvtXbki+hQhDx4wym7JUuWa/tUMLli3Ks1iOhSeCGwW/wM7i5ZPQ1LF6LLtbl4n2bIL9ystpodR/i2OJ7nqgoA1125qm0XDgzrm1424g1uIQlDS/+ONpuuipD4GcTd7bv+Ef7grtIHZhDuptXjWr1pKy2tKjwQLJw0Otx4UaFvvLz/jfGIKdv4/KsWYJXOHBjt8Kegz73WEDRaHeHhhDy1JIaXP6La1+tg0TCzVQ2pfSg+teXj8CJ0oaBM1n5C2zG77g6JMdtGr5waA==</xenc:CipherValue></xenc:CipherData></xenc:EncryptedKey></ds:KeyInfo><xenc:CipherData xmlns:xenc="http://www.w3.org/2001/04/xmlenc#"><xenc:CipherValue>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</xenc:CipherValue></xenc:CipherData></xenc:EncryptedData></saml2:EncryptedAssertion></saml2p:Response>`
)

func TestDecryptResponse(t *testing.T) {

	privateKey, err := decodePrivateKey([]byte(privateKeyString))
	if err != nil {
		t.Fatalf("decodePrivateKey fail %s", err)
	}

	resp := Response{}
	err = xml.Unmarshal([]byte(encryptedResp), &resp)
	if err != nil {
		t.Fatalf("unmarshal xml fail %s", err)
	}

	if resp.EncryptedAssertion != nil {
		asserText, err := resp.EncryptedAssertion.EncryptedData.decryptData(privateKey)
		if err != nil {
			t.Fatalf("EncryptedAssertion.EncryptedData.decryptData fail %s", err)
		}
		lastKey := "</saml2:Assertion>"
		lastIndex := strings.Index(string(asserText), lastKey) + len(lastKey)
		t.Logf("Decrypted assertion: len(assertion)=%d lastIndex=%d", len(asserText), lastIndex)

		assertion := Assertion{}
		err = xml.Unmarshal(asserText, &assertion)
		if err != nil {
			t.Fatalf("xml.Unmarshal fail %s", err)
		}

		encAsset, err := xml.MarshalIndent(assertion, "", "  ")
		if err != nil {
			t.Fatalf("xml.MarshalIndent fail %s", err)
		}

		t.Logf("assertion: %s", encAsset)
	}
}
