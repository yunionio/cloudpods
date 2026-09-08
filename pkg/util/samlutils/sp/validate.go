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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"hash"
	"strings"
	"time"
	"unicode"

	"github.com/beevik/etree"
	"github.com/ma314smith/signedxml"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/samlutils"
	"yunion.io/x/pkg/util/timeutils"

	"yunion.io/x/onecloud/pkg/httperrors"
)

const defaultSAMLClockSkew = 5 * time.Minute

type SAMLVerifyOptions struct {
	IdpEntityId     string
	SpEntityId      string
	Recipient       string
	Certs           []x509.Certificate
	DecryptKey      *rsa.PrivateKey
	Now             time.Time
	ClockSkew       time.Duration
	VerifySignature bool
}

func ParseCertificates(certStr string) ([]x509.Certificate, error) {
	certStr = strings.TrimSpace(certStr)
	if len(certStr) == 0 {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "empty signing_cert")
	}

	certs := make([]x509.Certificate, 0)
	rest := []byte(certStr)
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, errors.Wrap(err, "parse certificate")
		}
		certs = append(certs, *cert)
	}
	if len(certs) > 0 {
		return certs, nil
	}

	der := []byte(certStr)
	if decoded, err := base64.StdEncoding.DecodeString(stripAllSpace(certStr)); err == nil && len(decoded) > 0 {
		der = decoded
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "invalid signing_cert")
	}
	return []x509.Certificate{*cert}, nil
}

func CertificatesFromIdpDescriptor(desc samlutils.EntityDescriptor) []x509.Certificate {
	if desc.IDPSSODescriptor == nil {
		return nil
	}
	certs := make([]x509.Certificate, 0)
	for _, kd := range desc.IDPSSODescriptor.KeyDescriptors {
		if len(kd.Use) > 0 && kd.Use != samlutils.KEY_USE_SIGNING {
			continue
		}
		if kd.KeyInfo.X509Data == nil {
			continue
		}
		parsed, err := ParseCertificates(kd.KeyInfo.X509Data.X509Certificate.Cert)
		if err != nil {
			continue
		}
		certs = append(certs, parsed...)
	}
	return certs
}

func VerifySAMLResponse(xmlBytes []byte, resp *samlutils.Response, opts SAMLVerifyOptions) error {
	if resp == nil {
		return errors.Wrap(httperrors.ErrInvalidCredential, "missing SAML response")
	}
	if len(strings.TrimSpace(opts.IdpEntityId)) == 0 {
		return errors.Wrap(httperrors.ErrInvalidCredential, "missing IdP entity id")
	}
	if len(strings.TrimSpace(opts.SpEntityId)) == 0 {
		return errors.Wrap(httperrors.ErrInvalidCredential, "missing SP entity id")
	}

	if opts.VerifySignature {
		if len(opts.Certs) == 0 {
			return errors.Wrap(httperrors.ErrInvalidCredential, "missing IdP signing certificate")
		}
		signedXML, err := signedDocument(xmlBytes, resp, opts.DecryptKey)
		if err != nil {
			return err
		}
		referenced, err := validateXMLWithCertificates(signedXML, opts.Certs)
		if err != nil {
			return err
		}
		if err := bindSignedAssertion(resp, referenced); err != nil {
			return err
		}
	}

	if err := checkIssuers(resp, opts.IdpEntityId); err != nil {
		return err
	}
	if err := checkAudience(resp, opts.SpEntityId); err != nil {
		return err
	}
	if err := checkConditions(resp, opts); err != nil {
		return err
	}
	if err := checkDestination(resp, opts.Recipient); err != nil {
		return err
	}
	return nil
}

func signedDocument(xmlBytes []byte, resp *samlutils.Response, decryptKey *rsa.PrivateKey) (string, error) {
	if hasEnvelopedSignature(xmlBytes) {
		return string(xmlBytes), nil
	}
	if resp.EncryptedAssertion == nil {
		return "", errors.Wrap(httperrors.ErrInvalidCredential, "unsigned SAML response")
	}
	if decryptKey == nil {
		return "", errors.Wrap(httperrors.ErrInvalidCredential, "unsigned SAML response")
	}
	plain, err := decryptEncryptedAssertion(resp.EncryptedAssertion, decryptKey)
	if err != nil {
		return "", errors.Wrap(err, "decrypt assertion")
	}
	if !hasEnvelopedSignature(plain) {
		return "", errors.Wrap(httperrors.ErrInvalidCredential, "unsigned SAML assertion")
	}
	return string(plain), nil
}

func hasEnvelopedSignature(xmlBytes []byte) bool {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(xmlBytes); err != nil {
		return false
	}
	return doc.FindElement(".//Signature") != nil
}

func validateXMLWithCertificates(signed string, certs []x509.Certificate) ([]string, error) {
	validator, err := signedxml.NewValidator(signed)
	if err != nil {
		return nil, errors.Wrap(err, "signedxml.NewValidator")
	}
	validator.Certificates = append([]x509.Certificate{}, certs...)
	referenced, err := validator.ValidateReferences()
	if err != nil {
		return nil, errors.Wrap(httperrors.ErrInvalidCredential, "invalid SAML signature")
	}
	if len(referenced) == 0 {
		return nil, errors.Wrap(httperrors.ErrInvalidCredential, "invalid SAML signature")
	}
	return referenced, nil
}

func bindSignedAssertion(resp *samlutils.Response, referenced []string) error {
	if resp.Assertion == nil || len(strings.TrimSpace(resp.Assertion.ID)) == 0 {
		return errors.Wrap(httperrors.ErrInvalidCredential, "missing assertion")
	}
	wantId := strings.TrimSpace(resp.Assertion.ID)
	for _, refXML := range referenced {
		doc := etree.NewDocument()
		if err := doc.ReadFromString(refXML); err != nil {
			continue
		}
		el := findElementByID(doc, wantId)
		if el == nil {
			continue
		}
		if !strings.EqualFold(el.Tag, "Assertion") {
			continue
		}
		frag := etree.NewDocument()
		frag.SetRoot(el.Copy())
		xmlStr, err := frag.WriteToString()
		if err != nil {
			return errors.Wrap(err, "write signed assertion")
		}
		assertion := samlutils.Assertion{}
		if err := xml.Unmarshal([]byte(xmlStr), &assertion); err != nil {
			return errors.Wrap(err, "unmarshal signed assertion")
		}
		resp.Assertion = &assertion
		return nil
	}
	return errors.Wrap(httperrors.ErrInvalidCredential, "assertion is not signed")
}

func findElementByID(doc *etree.Document, id string) *etree.Element {
	if doc.Root() == nil || len(id) == 0 {
		return nil
	}
	if doc.Root().SelectAttrValue("ID", "") == id {
		return doc.Root()
	}
	return doc.FindElement(".//[@ID='" + id + "']")
}

func checkIssuers(resp *samlutils.Response, idpEntityId string) error {
	want := strings.TrimSpace(idpEntityId)
	if resp.Assertion == nil {
		return errors.Wrap(httperrors.ErrInvalidCredential, "missing assertion")
	}
	got := strings.TrimSpace(resp.Assertion.Issuer.Issuer)
	if got != want {
		return errors.Wrap(httperrors.ErrInvalidCredential, "issuer mismatch")
	}
	if len(strings.TrimSpace(resp.Issuer.Issuer)) > 0 && strings.TrimSpace(resp.Issuer.Issuer) != want {
		return errors.Wrap(httperrors.ErrInvalidCredential, "issuer mismatch")
	}
	return nil
}

func checkAudience(resp *samlutils.Response, spEntityId string) error {
	if resp.Assertion == nil {
		return errors.Wrap(httperrors.ErrInvalidCredential, "missing assertion")
	}
	want := strings.TrimSpace(spEntityId)
	restrictions := resp.Assertion.Conditions.AudienceRestrictions
	if len(restrictions) == 0 {
		return errors.Wrap(httperrors.ErrInvalidCredential, "missing audience")
	}
	for _, restriction := range restrictions {
		if strings.TrimSpace(restriction.Audience.Value) == want {
			return nil
		}
	}
	return errors.Wrap(httperrors.ErrInvalidCredential, "audience mismatch")
}

func checkConditions(resp *samlutils.Response, opts SAMLVerifyOptions) error {
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	skew := opts.ClockSkew
	if skew <= 0 {
		skew = defaultSAMLClockSkew
	}
	if resp.Assertion == nil {
		return errors.Wrap(httperrors.ErrInvalidCredential, "missing assertion")
	}

	hasExpiry := false
	cond := resp.Assertion.Conditions
	if err := checkTimeWindow(cond.NotBefore, cond.NotOnOrAfter, now, skew, &hasExpiry); err != nil {
		return err
	}
	scd := resp.Assertion.Subject.SubjectConfirmation.SubjectConfirmationData
	if err := checkTimeWindow(scd.NotBefore, scd.NotOnOrAfter, now, skew, &hasExpiry); err != nil {
		return err
	}
	if !hasExpiry {
		return errors.Wrap(httperrors.ErrInvalidCredential, "missing NotOnOrAfter")
	}
	if len(opts.Recipient) > 0 && len(strings.TrimSpace(scd.Recipient)) > 0 &&
		strings.TrimSpace(scd.Recipient) != strings.TrimSpace(opts.Recipient) {
		return errors.Wrap(httperrors.ErrInvalidCredential, "recipient mismatch")
	}
	return nil
}

func checkDestination(resp *samlutils.Response, recipient string) error {
	if len(recipient) == 0 || len(strings.TrimSpace(resp.Destination)) == 0 {
		return nil
	}
	if strings.TrimSpace(resp.Destination) != strings.TrimSpace(recipient) {
		return errors.Wrap(httperrors.ErrInvalidCredential, "destination mismatch")
	}
	return nil
}

func checkTimeWindow(notBefore *string, notOnOrAfter string, now time.Time, skew time.Duration, hasExpiry *bool) error {
	if notBefore != nil && len(strings.TrimSpace(*notBefore)) > 0 {
		t, err := parseSAMLTime(*notBefore)
		if err != nil {
			return errors.Wrap(httperrors.ErrInvalidCredential, "invalid NotBefore")
		}
		if now.Add(skew).Before(t) {
			return errors.Wrap(httperrors.ErrInvalidCredential, "assertion not yet valid")
		}
	}
	if len(strings.TrimSpace(notOnOrAfter)) == 0 {
		return nil
	}
	t, err := parseSAMLTime(notOnOrAfter)
	if err != nil {
		return errors.Wrap(httperrors.ErrInvalidCredential, "invalid NotOnOrAfter")
	}
	*hasExpiry = true
	if !now.Add(-skew).Before(t) {
		return errors.Wrap(httperrors.ErrInvalidCredential, "assertion expired")
	}
	return nil
}

func parseSAMLTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if t, err := timeutils.ParseTimeStr(s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	return time.Time{}, errors.Wrap(httperrors.ErrInvalidCredential, "invalid time")
}

func stripAllSpace(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}

func decryptEncryptedAssertion(enc *samlutils.EncryptedAssertion, privateKey *rsa.PrivateKey) ([]byte, error) {
	if enc == nil || privateKey == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidCredential, "missing encrypted assertion")
	}
	data := enc.EncryptedData
	cipherText, err := base64.StdEncoding.DecodeString(strings.TrimSpace(data.CipherData.CipherValue.Value))
	if err != nil {
		return nil, errors.Wrap(err, "decode encrypted data")
	}
	if data.KeyInfo.EncryptedKey == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidCredential, "missing encrypted key")
	}
	key, err := decryptEncryptedKey(*data.KeyInfo.EncryptedKey, privateKey)
	if err != nil {
		return nil, err
	}
	switch data.EncryptionMethod.Algorithm {
	case "http://www.w3.org/2001/04/xmlenc#aes128-cbc",
		"http://www.w3.org/2001/04/xmlenc#aes192-cbc",
		"http://www.w3.org/2001/04/xmlenc#aes256-cbc":
		plain, err := decryptAesCbc(key, cipherText)
		if err != nil {
			return nil, err
		}
		return stripPKCS7(plain), nil
	default:
		return nil, errors.Wrap(httperrors.ErrInvalidCredential, "unsupported encryption algorithm")
	}
}

func decryptEncryptedKey(key samlutils.EncryptedKey, privateKey *rsa.PrivateKey) ([]byte, error) {
	cipherText, err := base64.StdEncoding.DecodeString(strings.TrimSpace(key.CipherData.CipherValue.Value))
	if err != nil {
		return nil, errors.Wrap(err, "decode encrypted key")
	}
	if key.EncryptionMethod.Algorithm != "http://www.w3.org/2001/04/xmlenc#rsa-oaep-mgf1p" {
		return nil, errors.Wrap(httperrors.ErrInvalidCredential, "unsupported key encryption")
	}
	var shaAlg hash.Hash = sha1.New()
	if key.EncryptionMethod.DigestMethod != nil &&
		len(key.EncryptionMethod.DigestMethod.Algorithm) > 0 &&
		key.EncryptionMethod.DigestMethod.Algorithm != "http://www.w3.org/2000/09/xmldsig#sha1" {
		return nil, errors.Wrap(httperrors.ErrInvalidCredential, "unsupported key digest")
	}
	plaintext, err := rsa.DecryptOAEP(shaAlg, rand.Reader, privateKey, cipherText, nil)
	if err != nil {
		return nil, errors.Wrap(err, "decrypt key")
	}
	return plaintext, nil
}

func decryptAesCbc(key []byte, secret []byte) ([]byte, error) {
	if len(secret) < aes.BlockSize {
		return nil, errors.Wrap(httperrors.ErrInvalidCredential, "invalid encrypted assertion")
	}
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.Wrap(err, "aes.NewCipher")
	}
	decrypter := cipher.NewCBCDecrypter(c, secret[0:aes.BlockSize])
	data := make([]byte, len(secret)-aes.BlockSize)
	copy(data, secret[aes.BlockSize:])
	decrypter.CryptBlocks(data, data)
	return data, nil
}

func stripPKCS7(data []byte) []byte {
	if len(data) == 0 {
		return data
	}
	pad := int(data[len(data)-1])
	if pad <= 0 || pad > aes.BlockSize || pad > len(data) {
		return bytesTrimRightNull(data)
	}
	for i := 0; i < pad; i++ {
		if int(data[len(data)-1-i]) != pad {
			return bytesTrimRightNull(data)
		}
	}
	return data[:len(data)-pad]
}

func bytesTrimRightNull(data []byte) []byte {
	i := len(data)
	for i > 0 && data[i-1] == 0 {
		i--
	}
	return data[:i]
}
