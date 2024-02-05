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
	"context"
	"testing"

	"yunion.io/x/jsonutils"
)

func TestValidateCertKey_Validate(t *testing.T) {
	type fields struct {
		ValidatorCertificate *ValidatorCertificate
		ValidatorPrivateKey  *ValidatorPrivateKey
		certPubKeyAlgo       string
	}
	type args struct {
		data *jsonutils.JSONDict
	}
	cert := `-----BEGIN CERTIFICATE-----
MIIDEDCCAfigAwIBAgIIPfkszEMuuikwDQYJKoZIhvcNAQELBQAwEzERMA8GA1UE
AxMIb25lY2xvdWQwHhcNMjAwMTAzMTEwMzA3WhcNMjIwMzEzMTEwMzA3WjASMRAw
DgYDVQQDEwdzZXJ2aWNlMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA
phzvJJ3grosTiPBmmip6OpLkOezhbyuYbDGhynDH9N3+BWOQsPjzkigMhQjoygzH
G3Ac4f6yUJEzThTlkEckEVvxKftXTX+u2/F4SnaF61wiLA8SD2nXtYFNUhaOQ9Bo
vIcUlI/6idcxUZiktbAHv9FnawfNk1n6ryEy8LXPWS5RP+GhQfQG0Zko9APvLr/4
OB9zqgzc8+LUppxYtLMoy42FJ/OZhnGo31BWB9RW2WML+oO6J+uxt0JQGJwp8vr0
8OpSp5rgbu8G5jDlCOj4hEctbflJus/ES2jaR0etOcVSt+GYeTrKPmX9pPZ4TS9A
C1Lok0RR1xiDTS7Gw6pGnwIDAQABo2kwZzAOBgNVHQ8BAf8EBAMCBaAwEwYDVR0l
BAwwCgYIKwYBBQUHAwEwQAYDVR0RBDkwN4IHc2VydmljZYIQc2VydmljZS5vbmVj
bG91ZIIUc2VydmljZS5vbmVjbG91ZC5zdmOHBAqo3pYwDQYJKoZIhvcNAQELBQAD
ggEBAHYS4p2UrJ977SYFYYpsrE6Q01XSG6qt9EDTT9iB5GA/viuLURVHUMQDxKnf
2hSMq/UV+pGfGBw0Ki2sd+Mylb9qi59c26Ogqe8N/v+c219bYdnN7IzAgLQsEOd8
3iEJ7Ypb5pgf3B/dBPyWzxmKjZQ7vIfLYWgbmigPtf29yCWd3AlZrhI9zQUEcq7D
EAqtcvpU5/y7QsBNXo0QJa1WeeAOzYnKHUPQBJU4qLPm305BDHhKFyY23jsRSQMO
3CTjx9YsN9qGnv+oaqleA/ua/4f0QoPEQMXUsN1FGsAsC+vcKwDodY6vIvFnAk6o
pYTlCl+Ls6/Bu/Oml8AvrlaEyuc=
-----END CERTIFICATE-----`
	pkey := `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAphzvJJ3grosTiPBmmip6OpLkOezhbyuYbDGhynDH9N3+BWOQ
sPjzkigMhQjoygzHG3Ac4f6yUJEzThTlkEckEVvxKftXTX+u2/F4SnaF61wiLA8S
D2nXtYFNUhaOQ9BovIcUlI/6idcxUZiktbAHv9FnawfNk1n6ryEy8LXPWS5RP+Gh
QfQG0Zko9APvLr/4OB9zqgzc8+LUppxYtLMoy42FJ/OZhnGo31BWB9RW2WML+oO6
J+uxt0JQGJwp8vr08OpSp5rgbu8G5jDlCOj4hEctbflJus/ES2jaR0etOcVSt+GY
eTrKPmX9pPZ4TS9AC1Lok0RR1xiDTS7Gw6pGnwIDAQABAoIBAHqbvrQDSBTtGIUq
FEFUexWC2KwcuSSqQ/4QAECBUEXgGR/3JpRJnNbTcrI7KkAAgHIzJU52BT3Mftby
O6NrryaU+4OmPgE47mLvb39ezmgzgBGPKiBwWkRhZSXi+iz5xmTpO3qQbzeQu5lj
lqd4f6/Iq5Hnl4hckNj1IzliqOJEShiYQV69NADbUsnmruHL3Mt3f3SQI9yv+k48
5QMGol0eIuFkur6+I0mryvfx36gqHOsZIRweAMPGmGLr+aatrsXfV1pHON/4FhPY
dOSEpVz2hOPt9HMMy8AJXjMDvl8RfaW/T7NY+2lJoE4IMcxuGOqQRGnnBRioOuP9
lnuTbnECgYEAxq79lGJb5oasmk277Z+8h7BF9lHMDo6jah8cEGL5lTS4u7SYcLHp
uL9WBDoAhdVX2XULF13D+NPXQObySr96oGXJxlAtxSpRjURAGlLIn4Y3rth+RUT/
k+N7QkS78wfUCf71icnXr71SqctVC6tJnqgvxF5JP+Cy4P1BuSYcPEkCgYEA1giS
f17Q3t1iWmNGLRb+J6JQeYoQyFDPX/ROz0Ko7Nf1svzYLTWrjJGjXXw8/vUOs0pP
M7yV2TFzLT7nGuDBfwFVU4Kv3JNU5x+Z6++g7tv3/3JiKG0QXQvwNPbYVHrWj1cf
T15HyVNZwibydEl0Q6HdZpzZP2cxg3Jm9kVJW6cCgYAdOMmVFG5d1nr2au50AaVx
84wmsVso3PPN/OtcwaHhvxJYkTRGhvRQNtwI3RsMlBdKpXtPIXxcUZP8OLt0IPuB
MddecpZ4xEOgWmRvOrPFOrFf5vmTaJWKg8+yLHfUQ9d87OHiNSyi7V6GGKDWiYfX
bPcxk4iEe6DzlGwhNii6+QKBgQDF38T4poL6F7gvEmq1kvVDVSeLRd6AI12lO2uE
5/7egEXxxRpiqaTA34AmFI8bsxl1HjUdArOSycnOwcHNMo8RSP1GqKLHjRpIVwnp
e2/QhGLBslEXSMWBEGFxxeh4KdylRol2yhYaBcoM2g76/VHUmRfkHwwmNtQqzyBr
e+D3LwKBgE2fGOe5rqZ8/mCJcx7Wlt8gqD66WUhGqMh3pd4Om6ZGFVLvhPWEu3Md
57oFd4cQZ6FSIOKb+cFIpCIW8sXG+c3vFjIC5PDMnSPbasF5KHN/Kg3C+p2ilviv
Oycn3Dy2jpdE7SpoBCt3HsIhra8a6h7BcCQ87UwObqTsdLe+7/oD
-----END RSA PRIVATE KEY-----`
	var data = jsonutils.NewDict()
	data.Set("certificate", jsonutils.NewString(cert))
	data.Set("private_key", jsonutils.NewString(pkey))
	var data2 = jsonutils.NewDict()
	data2.Set("certificate", jsonutils.NewString(cert+"error"))
	data2.Set("private_key", jsonutils.NewString(pkey+"error"))
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "good in",
			fields: fields{
				ValidatorCertificate: NewCertificateValidator("certificate"),
				ValidatorPrivateKey:  NewPrivateKeyValidator("private_key"),
			},
			args:    args{data},
			wantErr: false,
		},
		{
			name: "bad in",
			fields: fields{
				ValidatorCertificate: NewCertificateValidator("certificate"),
				ValidatorPrivateKey:  NewPrivateKeyValidator("private_key"),
			},
			args:    args{data2},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &ValidatorCertKey{
				ValidatorCertificate: tt.fields.ValidatorCertificate,
				ValidatorPrivateKey:  tt.fields.ValidatorPrivateKey,
				certPubKeyAlgo:       tt.fields.certPubKeyAlgo,
			}
			if err := v.Validate(context.Background(), tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("ValidateCertKey.Validator() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
