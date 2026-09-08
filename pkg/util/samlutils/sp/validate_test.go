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

package sp

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"encoding/xml"
	"math/big"
	"strings"
	"testing"
	"time"

	"yunion.io/x/pkg/util/samlutils"
)

const (
	testIdpEntityId = "https://idp.example.com/saml"
	testSpEntityId  = "https://sp.example.com"
	testACS         = "https://sp.example.com/acs"
)

func TestParseCertificates(t *testing.T) {
	_, cert, pemStr := mustGenCert(t)
	parsed, err := ParseCertificates(pemStr)
	if err != nil {
		t.Fatalf("ParseCertificates pem: %v", err)
	}
	if len(parsed) != 1 || parsed[0].SerialNumber.Cmp(cert.SerialNumber) != 0 {
		t.Fatalf("unexpected pem cert")
	}

	raw := strings.TrimSpace(strings.ReplaceAll(pemStr, "-----BEGIN CERTIFICATE-----", ""))
	raw = strings.TrimSpace(strings.ReplaceAll(raw, "-----END CERTIFICATE-----", ""))
	parsed, err = ParseCertificates(raw)
	if err != nil {
		t.Fatalf("ParseCertificates raw base64: %v", err)
	}
	if len(parsed) != 1 {
		t.Fatalf("expected 1 cert, got %d", len(parsed))
	}

	if _, err := ParseCertificates(""); err == nil {
		t.Fatalf("empty cert should fail")
	}
	if _, err := ParseCertificates("not-a-cert"); err == nil {
		t.Fatalf("invalid cert should fail")
	}
}

func TestVerifySAMLResponse(t *testing.T) {
	key, cert, pemStr := mustGenCert(t)
	_, otherCert, _ := mustGenCert(t)

	signed, resp := mustSignedResponse(t, key, pemStr, testIdpEntityId, testSpEntityId, time.Now().UTC())
	unsigned, unsignedResp := mustUnsignedResponse(t, testIdpEntityId, testSpEntityId, time.Now().UTC())

	baseOpts := SAMLVerifyOptions{
		IdpEntityId: testIdpEntityId,
		SpEntityId:  testSpEntityId,
		Recipient:   testACS,
		Now:         time.Now().UTC(),
	}
	if err := VerifySAMLResponse([]byte(unsigned), unsignedResp, baseOpts); err != nil {
		t.Fatalf("unsigned response should pass without signature check: %v", err)
	}
	if err := VerifySAMLResponse([]byte(signed), resp, baseOpts); err != nil {
		t.Fatalf("signed response should pass without signature check: %v", err)
	}

	wrongIssuer, wrongIssuerResp := mustUnsignedResponse(t, "https://evil.example.com", testSpEntityId, time.Now().UTC())
	if err := VerifySAMLResponse([]byte(wrongIssuer), wrongIssuerResp, baseOpts); err == nil {
		t.Fatalf("wrong issuer should fail")
	}

	wrongAud, wrongAudResp := mustUnsignedResponse(t, testIdpEntityId, "https://other-sp.example.com", time.Now().UTC())
	if err := VerifySAMLResponse([]byte(wrongAud), wrongAudResp, baseOpts); err == nil {
		t.Fatalf("wrong audience should fail")
	}

	expired, expiredResp := mustUnsignedResponse(t, testIdpEntityId, testSpEntityId, time.Now().UTC())
	if err := VerifySAMLResponse([]byte(expired), expiredResp, SAMLVerifyOptions{
		IdpEntityId: testIdpEntityId,
		SpEntityId:  testSpEntityId,
		Now:         time.Now().UTC().Add(time.Hour),
	}); err == nil {
		t.Fatalf("expired assertion should fail")
	}

	sigOpts := SAMLVerifyOptions{
		IdpEntityId:     testIdpEntityId,
		SpEntityId:      testSpEntityId,
		Recipient:       testACS,
		Certs:           []x509.Certificate{*cert},
		Now:             time.Now().UTC(),
		VerifySignature: true,
	}
	if err := VerifySAMLResponse([]byte(signed), cloneResponse(t, signed), sigOpts); err != nil {
		t.Fatalf("valid signed response: %v", err)
	}
	if err := VerifySAMLResponse([]byte(unsigned), unsignedResp, sigOpts); err == nil {
		t.Fatalf("unsigned response should fail when signature check is enabled")
	}

	if err := VerifySAMLResponse([]byte(signed), cloneResponse(t, signed), SAMLVerifyOptions{
		IdpEntityId:     testIdpEntityId,
		SpEntityId:      testSpEntityId,
		Certs:           []x509.Certificate{*otherCert},
		Now:             time.Now().UTC(),
		VerifySignature: true,
	}); err == nil {
		t.Fatalf("wrong IdP certificate should fail")
	}

	attackerKey, attackerCert, attackerPem := mustGenCert(t)
	attackerSigned, attackerResp := mustSignedResponse(t, attackerKey, attackerPem, testIdpEntityId, testSpEntityId, time.Now().UTC())
	if err := VerifySAMLResponse([]byte(attackerSigned), attackerResp, SAMLVerifyOptions{
		IdpEntityId:     testIdpEntityId,
		SpEntityId:      testSpEntityId,
		Certs:           []x509.Certificate{*cert},
		Now:             time.Now().UTC(),
		VerifySignature: true,
	}); err == nil {
		t.Fatalf("embedded attacker certificate should not be trusted")
	}
	_ = attackerCert
}

func TestVerifySAMLResponseRejectsUnsignedAssertionWithSignedWrapper(t *testing.T) {
	key, cert, pemStr := mustGenCert(t)
	signed, _ := mustSignedResponse(t, key, pemStr, testIdpEntityId, testSpEntityId, time.Now().UTC())

	var signedResp samlutils.Response
	if err := xml.Unmarshal([]byte(signed), &signedResp); err != nil {
		t.Fatalf("unmarshal signed: %v", err)
	}
	unsigned, unsignedResp := mustUnsignedResponse(t, testIdpEntityId, testSpEntityId, time.Now().UTC())
	_ = unsigned
	unsignedResp.Assertion.ID = "_unsigned-assertion"
	signedResp.Assertion = unsignedResp.Assertion

	opts := SAMLVerifyOptions{
		IdpEntityId:     testIdpEntityId,
		SpEntityId:      testSpEntityId,
		Certs:           []x509.Certificate{*cert},
		Now:             time.Now().UTC(),
		VerifySignature: true,
	}
	if err := VerifySAMLResponse([]byte(signed), &signedResp, opts); err == nil {
		t.Fatalf("unsigned substituted assertion should fail")
	}
}

func mustGenCert(t *testing.T) (*rsa.PrivateKey, *x509.Certificate, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: "idp.example.com"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	pemStr := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	return key, cert, pemStr
}

func mustUnsignedResponse(t *testing.T, idpEntityId, spEntityId string, now time.Time) (string, *samlutils.Response) {
	t.Helper()
	resp := samlutils.NewResponse(samlutils.SSAMLResponseInput{
		IssuerEntityId:              idpEntityId,
		RequestEntityId:             spEntityId,
		AssertionConsumerServiceURL: testACS,
		SSAMLSpInitiatedLoginData: samlutils.SSAMLSpInitiatedLoginData{
			NameId:              "user1",
			NameIdFormat:        samlutils.NAME_ID_FORMAT_UNSPEC,
			AudienceRestriction: spEntityId,
			Attributes: []samlutils.SSAMLResponseAttribute{
				{Name: "uid", Values: []string{"user1"}},
			},
		},
	})
	xmlBytes, err := xml.Marshal(&resp)
	if err != nil {
		t.Fatalf("xml.Marshal: %v", err)
	}
	out := samlutils.Response{}
	if err := xml.Unmarshal(xmlBytes, &out); err != nil {
		t.Fatalf("xml.Unmarshal: %v", err)
	}
	return string(xmlBytes), &out
}

func mustSignedResponse(t *testing.T, key *rsa.PrivateKey, certPEM, idpEntityId, spEntityId string, now time.Time) (string, *samlutils.Response) {
	t.Helper()
	_ = now
	block, _ := pem.Decode([]byte(certPEM))
	certB64 := ""
	if block != nil {
		certB64 = string(pem.EncodeToMemory(block))
		certB64 = strings.TrimPrefix(certB64, "-----BEGIN CERTIFICATE-----")
		certB64 = strings.TrimSuffix(strings.TrimSpace(certB64), "-----END CERTIFICATE-----")
		certB64 = strings.TrimSpace(certB64)
	}
	resp := samlutils.NewResponse(samlutils.SSAMLResponseInput{
		IssuerEntityId:              idpEntityId,
		RequestEntityId:             spEntityId,
		AssertionConsumerServiceURL: testACS,
		IssuerCertString:            certB64,
		SSAMLSpInitiatedLoginData: samlutils.SSAMLSpInitiatedLoginData{
			NameId:              "user1",
			NameIdFormat:        samlutils.NAME_ID_FORMAT_UNSPEC,
			AudienceRestriction: spEntityId,
			Attributes: []samlutils.SSAMLResponseAttribute{
				{Name: "uid", Values: []string{"user1"}},
			},
		},
	})
	xmlBytes, err := xml.Marshal(&resp)
	if err != nil {
		t.Fatalf("xml.Marshal: %v", err)
	}
	signed, err := samlutils.SignXML(string(xmlBytes), key)
	if err != nil {
		t.Fatalf("SignXML: %v", err)
	}
	out := samlutils.Response{}
	if err := xml.Unmarshal([]byte(signed), &out); err != nil {
		t.Fatalf("xml.Unmarshal signed: %v", err)
	}
	return signed, &out
}

func cloneResponse(t *testing.T, xmlStr string) *samlutils.Response {
	t.Helper()
	out := samlutils.Response{}
	if err := xml.Unmarshal([]byte(xmlStr), &out); err != nil {
		t.Fatalf("xml.Unmarshal: %v", err)
	}
	return &out
}
