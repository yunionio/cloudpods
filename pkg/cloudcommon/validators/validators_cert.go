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
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"reflect"
	"strings"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type ValidatorPEM struct {
	Validator
	Blocks []*pem.Block
}

func NewPEMValidator(key string) *ValidatorPEM {
	v := &ValidatorPEM{
		Validator: Validator{Key: key},
	}
	v.SetParent(v)
	return v
}

func (v *ValidatorPEM) getValue() interface{} {
	return v.Blocks
}

func (v *ValidatorPEM) parseFromString(s string) []*pem.Block {
	blocks := []*pem.Block{}
	for d := []byte(s); ; {
		block, rest := pem.Decode(d)
		if block == nil {
			if len(rest) > 0 {
				return nil
			}
			break
		}
		blocks = append(blocks, block)
		d = rest
	}
	return blocks
}

func (v *ValidatorPEM) setDefault(data *jsonutils.JSONDict) bool {
	if v.defaultVal == nil {
		return false
	}
	s, ok := v.defaultVal.(string)
	if !ok {
		return false
	}
	blocks := v.parseFromString(s)
	if blocks != nil {
		value := jsonutils.NewString(s)
		v.value = value
		data.Set(v.Key, value)
		v.Blocks = blocks
		return true
	}
	return false
}

func (v *ValidatorPEM) Validate(data *jsonutils.JSONDict) error {
	if err, isSet := v.Validator.validateEx(data); err != nil || !isSet {
		return err
	}
	s, err := v.value.GetString()
	if err != nil {
		return newInvalidTypeError(v.Key, "pem", err)
	}
	v.Blocks = v.parseFromString(s)
	return nil
}

type ValidatorCertificate struct {
	ValidatorPEM
	Certificates []*x509.Certificate
}

func NewCertificateValidator(key string) *ValidatorCertificate {
	v := &ValidatorCertificate{
		ValidatorPEM: *NewPEMValidator(key),
	}
	v.SetParent(v)
	return v
}

func (v *ValidatorCertificate) getValue() interface{} {
	return v.Certificates
}

func (v *ValidatorCertificate) parseFromBlocks(blocks []*pem.Block) []*x509.Certificate {
	certs := []*x509.Certificate{}
	for i, block := range blocks {
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil
		}
		if i > 0 {
			certSub := certs[i-1]
			if err := certSub.CheckSignatureFrom(cert); err != nil {
				return nil
			}
		}
		certs = append(certs, cert)
	}
	return certs
}

func (v *ValidatorCertificate) parseFromString(s string) []*x509.Certificate {
	blocks := v.ValidatorPEM.parseFromString(s)
	if blocks == nil {
		return nil
	}
	v.Blocks = blocks
	certs := v.parseFromBlocks(blocks)
	return certs
}

func (v *ValidatorCertificate) setDefault(data *jsonutils.JSONDict) bool {
	if v.defaultVal == nil {
		return false
	}
	s, ok := v.defaultVal.(string)
	if !ok {
		return false
	}
	certs := v.parseFromString(s)
	if certs != nil {
		v.setCertificates(certs, data)
		return true
	}
	return false
}

func (v *ValidatorCertificate) setCertificates(certs []*x509.Certificate, data *jsonutils.JSONDict) {
	pems := []byte{}
	for _, block := range v.Blocks {
		d := pem.EncodeToMemory(block)
		pems = append(pems, d...)
	}
	value := jsonutils.NewString(string(pems))
	v.value = value
	data.Set(v.Key, value)
	v.Certificates = certs
}

func (v *ValidatorCertificate) Validate(data *jsonutils.JSONDict) error {
	if err, isSet := v.Validator.validateEx(data); err != nil || !isSet {
		return err
	}
	s, err := v.value.GetString()
	if err != nil {
		return newInvalidTypeError(v.Key, "certificate", err)
	}
	certs := v.parseFromString(s)
	if !v.optional && len(certs) == 0 {
		return newInvalidValueError(v.Key, "empty certificate chain")
	}
	if len(certs) > 0 {
		for _, block := range v.Blocks {
			if block.Type != "CERTIFICATE" {
				err := fmt.Errorf("wrong PEM type: %s", block.Type)
				return newInvalidTypeError(v.Key, "certificate", err)
			}
		}
		for i := 0; i < len(certs)-1; i++ {
			cert := certs[i]
			certP := certs[i+1]
			err := cert.CheckSignatureFrom(certP)
			if err != nil {
				msg := fmt.Sprintf("cannot verify signature of certificate %d in the chain", i+1)
				return newInvalidValueError(v.Key, msg)
			}
		}
		v.setCertificates(certs, data)
		return nil
	}
	return nil
}

func (v *ValidatorCertificate) FingerprintSha256() []byte {
	csum := sha256.Sum256(v.Certificates[0].Raw)
	return csum[:]
}

func (v *ValidatorCertificate) FingerprintSha256String() string {
	fp := v.FingerprintSha256()
	s := hex.EncodeToString(fp)
	return s
}

func (v *ValidatorCertificate) PublicKeyBitLen() int {
	cert := v.Certificates[0]
	pubkey := cert.PublicKey
	switch pub := pubkey.(type) {
	case *rsa.PublicKey:
		return pub.N.BitLen()
	case *ecdsa.PublicKey:
		return pub.X.BitLen()
	default:
		return 0
	}
}

type ValidatorPrivateKey struct {
	ValidatorPEM
	PrivateKey crypto.PrivateKey
}

func NewPrivateKeyValidator(key string) *ValidatorPrivateKey {
	v := &ValidatorPrivateKey{
		ValidatorPEM: *NewPEMValidator(key),
	}
	v.SetParent(v)
	return v
}

func (v *ValidatorPrivateKey) getValue() interface{} {
	return v.PrivateKey
}

func (v *ValidatorPrivateKey) parseFromBlock(block *pem.Block) (crypto.PrivateKey, error) {
	const RSA_PRIVATE_KEY = "RSA PRIVATE KEY"
	const EC_PRIVATE_KEY = "EC PRIVATE KEY"
	const PKCS8_PRIVATE_KEY = "PRIVATE KEY"

	var fuzz bool
	switch block.Type {
	case PKCS8_PRIVATE_KEY:
		goto parsePkcs8
	case RSA_PRIVATE_KEY:
		goto parseRsa
	case EC_PRIVATE_KEY:
		goto parseEc
	default:
	}
	fuzz = true
parsePkcs8:
	{
		pkey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err == nil {
			switch pkey1 := pkey.(type) {
			case *rsa.PrivateKey:
				block.Type = RSA_PRIVATE_KEY
				block.Bytes = x509.MarshalPKCS1PrivateKey(pkey1)
				return pkey1, nil
			case *ecdsa.PrivateKey:
				block.Type = EC_PRIVATE_KEY
				block.Bytes, _ = x509.MarshalECPrivateKey(pkey1)
				return pkey1, nil
			default:
				return nil, newInvalidValueError(v.Key, "unknown private key type")
			}
		}
		if !fuzz {
			return nil, newInvalidValueError(v.Key, fmt.Sprintf("invalid pkcs8 private key: %s", err))
		}
	}
parseRsa:
	{
		pkey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err == nil {
			block.Type = RSA_PRIVATE_KEY
			return pkey, nil
		}
		if !fuzz {
			return nil, newInvalidValueError(v.Key, fmt.Sprintf("invalid rsa private key: %s", err))
		}
	}
parseEc:
	{
		pkey, err := x509.ParseECPrivateKey(block.Bytes)
		if err == nil {
			block.Type = EC_PRIVATE_KEY
			return pkey, nil
		}
		if !fuzz {
			return nil, newInvalidValueError(v.Key, fmt.Sprintf("invalid ec private key: %s", err))
		}
	}
	return nil, newInvalidValueError(v.Key, "invalid private key")
}

func (v *ValidatorPrivateKey) parseFromString(s string) (crypto.PrivateKey, error) {
	blocks := v.ValidatorPEM.parseFromString(s)
	if len(blocks) != 1 {
		return nil, newInvalidValueError(v.Key, fmt.Sprintf("found %d pem blocks, expecting 1", len(blocks)))
	}
	v.Blocks = blocks
	pkey, err := v.parseFromBlock(blocks[0])
	return pkey, err
}

func (v *ValidatorPrivateKey) setDefault(data *jsonutils.JSONDict) bool {
	if v.defaultVal == nil {
		return false
	}
	s, ok := v.defaultVal.(string)
	if !ok {
		return false
	}
	pkey, err := v.parseFromString(s)
	if err != nil {
		return false
	}
	v.setPrivateKey(pkey, data)
	return true
}

func (v *ValidatorPrivateKey) setPrivateKey(pkey crypto.PrivateKey, data *jsonutils.JSONDict) {
	d := pem.EncodeToMemory(v.Blocks[0])
	value := jsonutils.NewString(string(d))
	v.value = value
	data.Set(v.Key, value)
	v.PrivateKey = pkey
}

func (v *ValidatorPrivateKey) Validate(data *jsonutils.JSONDict) error {
	if err, isSet := v.Validator.validateEx(data); err != nil || !isSet {
		return err
	}
	s, err := v.value.GetString()
	if err != nil {
		return newInvalidTypeError(v.Key, "privateKey", err)
	}
	pkey, err := v.parseFromString(s)
	if err != nil {
		return err
	}
	v.setPrivateKey(pkey, data)
	return nil
}

func (v *ValidatorPrivateKey) MatchCertificate(cert *x509.Certificate) error {
	switch pkey := v.PrivateKey.(type) {
	case *rsa.PrivateKey:
		pubkey0 := pkey.Public()
		pubkey1, _ := cert.PublicKey.(crypto.PublicKey) // safe conversion
		if !reflect.DeepEqual(pubkey0, pubkey1) {
			return newInvalidValueError(v.Key, "certificate and rsa key do not match")
		}
	case *ecdsa.PrivateKey:
		pubkey0 := pkey.Public()
		pubkey1, _ := cert.PublicKey.(crypto.PublicKey)
		if !reflect.DeepEqual(pubkey0, pubkey1) {
			return newInvalidValueError(v.Key, "certificate and ec key do not match")
		}
	default:
		// should never happen
		return newInvalidValueError(v.Key, "unknown private key type")
	}
	return nil
}

type ValidatorCertKey struct {
	*ValidatorCertificate
	*ValidatorPrivateKey

	certPubKeyAlgo string
}

func NewCertKeyValidator(cert, key string) *ValidatorCertKey {
	return &ValidatorCertKey{
		ValidatorCertificate: NewCertificateValidator(cert),
		ValidatorPrivateKey:  NewPrivateKeyValidator(key),
	}
}

func (v *ValidatorCertKey) Validate(data *jsonutils.JSONDict) error {
	keyV := map[string]IValidator{
		"certificate": v.ValidatorCertificate,
		"private_key": v.ValidatorPrivateKey,
	}
	for _, v := range keyV {
		if err := v.Validate(data); err != nil {
			return err
		}
	}
	cert := v.ValidatorCertificate.Certificates[0]
	var certPubKeyAlgo string
	{
		// x509.PublicKeyAlgorithm.String() is only available since go1.10
		switch cert.PublicKeyAlgorithm {
		case x509.RSA:
			certPubKeyAlgo = api.LB_TLS_CERT_PUBKEY_ALGO_RSA
		case x509.ECDSA:
			certPubKeyAlgo = api.LB_TLS_CERT_PUBKEY_ALGO_ECDSA
		default:
			certPubKeyAlgo = fmt.Sprintf("algo %#v", cert.PublicKeyAlgorithm)
		}
		if !api.LB_TLS_CERT_PUBKEY_ALGOS.Has(certPubKeyAlgo) {
			return httperrors.NewInputParameterError("invalid cert pubkey algorithm: %s, want %s",
				certPubKeyAlgo, api.LB_TLS_CERT_PUBKEY_ALGOS.String())
		}
	}
	v.certPubKeyAlgo = certPubKeyAlgo
	if err := v.ValidatorPrivateKey.MatchCertificate(cert); err != nil {
		return err
	}
	return nil
}

func (v *ValidatorCertKey) UpdateCertKeyInfo(ctx context.Context, data *jsonutils.JSONDict) *jsonutils.JSONDict {
	cert := v.ValidatorCertificate.Certificates[0]
	// NOTE subject alternative names also includes email, url, ip addresses,
	// but we ignore them here.
	//
	// NOTE we use white space to separate names
	data.Set("common_name", jsonutils.NewString(cert.Subject.CommonName))
	data.Set("subject_alternative_names", jsonutils.NewString(strings.Join(cert.DNSNames, " ")))

	data.Set("not_before", jsonutils.NewTimeString(cert.NotBefore))
	data.Set("not_after", jsonutils.NewTimeString(cert.NotAfter))
	data.Set("public_key_algorithm", jsonutils.NewString(v.certPubKeyAlgo))
	data.Set("public_key_bit_len", jsonutils.NewInt(int64(v.ValidatorCertificate.PublicKeyBitLen())))
	data.Set("signature_algorithm", jsonutils.NewString(cert.SignatureAlgorithm.String()))
	data.Set("fingerprint", jsonutils.NewString(api.LB_TLS_CERT_FINGERPRINT_ALGO_SHA256+":"+v.ValidatorCertificate.FingerprintSha256String()))
	return data
}
