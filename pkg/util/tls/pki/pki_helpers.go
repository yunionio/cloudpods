/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pki

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	certutil "yunion.io/x/onecloud/pkg/util/tls/cert"
	keyutil "yunion.io/x/onecloud/pkg/util/tls/key"
)

const (
	// PrivateKeyBlockType is a possible value for pem.Block.Type.
	PrivateKeyBlockType = "PRIVATE KEY"
	// PublicKeyBlockType is a possible value for pem.Block.Type.
	PublicKeyBlockType = "PUBLIC KEY"
	// CertificateBlockType is a possible value for pem.Block.Type.
	CertificateBlockType = "CERTIFICATE"
	// RSAPrivateKeyBlockType is a possible value for pem.BlockType.
	RSAPrivateKeyBlockType = "RSA PRIVATE KEY"
	rsaKeySize             = 2048
)

var (
	CertificateValidity = time.Hour * 24 * 365 * 100
)

// CertConfig is a wrapper around certutil.Config extending it with PublicKeyAlgorithm.
type CertConfig struct {
	certutil.Config
	PublicKeyAlgorithm x509.PublicKeyAlgorithm
}

// NewCertificateAuthority creates new certificate and private key for the certificate authority
func NewCertificateAuthority(config *CertConfig) (*x509.Certificate, crypto.Signer, error) {
	key, err := NewPrivateKey(config.PublicKeyAlgorithm)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to create private key while generating CA certificate")
	}

	cert, err := certutil.NewSelfSignedCACert(config.Config, key)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to create self-signed CA certificate")
	}

	return cert, key, nil
}

// NewCertAndKey creates new certificate and key by passing the certificate authority certificate and key
func NewCertAndKey(caCert *x509.Certificate, caKey crypto.Signer, config *CertConfig) (*x509.Certificate, crypto.Signer, error) {
	key, err := NewPrivateKey(config.PublicKeyAlgorithm)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to create private key")
	}

	cert, err := NewSignedCert(config, key, caCert, caKey)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to sign certificate")
	}

	return cert, key, nil
}

// NewCSR creates a new CSR
func NewCSR(cfg CertConfig, key crypto.Signer) (*x509.CertificateRequest, error) {
	template := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: cfg.Organization,
		},
		DNSNames:    cfg.AltNames.DNSNames,
		IPAddresses: cfg.AltNames.IPs,
	}

	csrBytes, err := x509.CreateCertificateRequest(cryptorand.Reader, template, key)

	if err != nil {
		return nil, errors.Wrap(err, "failed to create a CSR")
	}

	return x509.ParseCertificateRequest(csrBytes)
}

// NewCSRAndKey generates a new key and CSR and that could be signed to create the given certificate
func NewCSRAndKey(config *CertConfig) (*x509.CertificateRequest, crypto.Signer, error) {
	key, err := NewPrivateKey(config.PublicKeyAlgorithm)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to create private key")
	}

	csr, err := NewCSR(*config, key)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to generate CSR")
	}

	return csr, key, nil
}

// HasServerAuth returns true if the given certificate is a ServerAuth
func HasServerAuth(cert *x509.Certificate) bool {
	for i := range cert.ExtKeyUsage {
		if cert.ExtKeyUsage[i] == x509.ExtKeyUsageServerAuth {
			return true
		}
	}
	return false
}

// WriteCertAndKey stores certificate and key at the specified location
func WriteCertAndKey(pkiPath string, name string, cert *x509.Certificate, key crypto.Signer) error {
	if err := WriteKey(pkiPath, name, key); err != nil {
		return errors.Wrap(err, "couldn't write key")
	}

	return WriteCert(pkiPath, name, cert)
}

// WriteCert stores the given certificate at the given location
func WriteCert(pkiPath, name string, cert *x509.Certificate) error {
	if cert == nil {
		return errors.Error("certificate cannot be nil when writing to file")
	}

	certificatePath := pathForCert(pkiPath, name)
	if err := certutil.WriteCert(certificatePath, EncodeCertPEM(cert)); err != nil {
		return errors.Wrapf(err, "unable to write certificate to file %s", certificatePath)
	}

	return nil
}

// WriteKey stores the given key at the given location
func WriteKey(pkiPath, name string, key crypto.Signer) error {
	if key == nil {
		return errors.Error("private key cannot be nil when writing to file")
	}

	privateKeyPath := pathForKey(pkiPath, name)
	encoded, err := keyutil.MarshalPrivateKeyToPEM(key)
	if err != nil {
		return errors.Wrapf(err, "unable to marshal private key to PEM")
	}
	if err := keyutil.WriteKey(privateKeyPath, encoded); err != nil {
		return errors.Wrapf(err, "unable to write private key to file %s", privateKeyPath)
	}

	return nil
}

// WriteCSR writes the pem-encoded CSR data to csrPath.
// The CSR file will be created with file mode 0600.
// If the CSR file already exists, it will be overwritten.
// The parent directory of the csrPath will be created as needed with file mode 0700.
func WriteCSR(csrDir, name string, csr *x509.CertificateRequest) error {
	if csr == nil {
		return errors.Error("certificate request cannot be nil when writing to file")
	}

	csrPath := pathForCSR(csrDir, name)
	if err := os.MkdirAll(filepath.Dir(csrPath), os.FileMode(0700)); err != nil {
		return errors.Wrapf(err, "failed to make directory %s", filepath.Dir(csrPath))
	}

	if err := ioutil.WriteFile(csrPath, EncodeCSRPEM(csr), os.FileMode(0600)); err != nil {
		return errors.Wrapf(err, "unable to write CSR to file %s", csrPath)
	}

	return nil
}

// WritePublicKey stores the given public key at the given location
func WritePublicKey(pkiPath, name string, key crypto.PublicKey) error {
	if key == nil {
		return errors.Error("public key cannot be nil when writing to file")
	}

	publicKeyBytes, err := EncodePublicKeyPEM(key)
	if err != nil {
		return err
	}
	publicKeyPath := pathForPublicKey(pkiPath, name)
	if err := keyutil.WriteKey(publicKeyPath, publicKeyBytes); err != nil {
		return errors.Wrapf(err, "unable to write public key to file %s", publicKeyPath)
	}

	return nil
}

// TryLoadCertAndKeyFromDisk tries to load a cert and a key from the disk and validates that they are valid
func TryLoadCertAndKeyFromDisk(pkiPath, name string) (*x509.Certificate, crypto.Signer, error) {
	cert, err := TryLoadCertFromDisk(pkiPath, name)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to load certificate")
	}

	key, err := TryLoadKeyFromDisk(pkiPath, name)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to load key")
	}

	return cert, key, nil
}

// TryLoadCertFromDisk tries to load the cert from the disk and validates that it is valid
func TryLoadCertFromDisk(pkiPath, name string) (*x509.Certificate, error) {
	certificatePath := pathForCert(pkiPath, name)

	certs, err := certutil.CertsFromFile(certificatePath)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't load the certificate file %s", certificatePath)
	}

	// We are only putting one certificate in the certificate pem file, so it's safe to just pick the first one
	// TODO: Support multiple certs here in order to be able to rotate certs
	cert := certs[0]

	// Check so that the certificate is valid now
	now := time.Now()
	if now.Before(cert.NotBefore) {
		return nil, errors.Error("the certificate is not valid yet")
	}
	if now.After(cert.NotAfter) {
		return nil, errors.Error("the certificate has expired")
	}

	return cert, nil
}

// TryLoadKeyFromDisk tries to load the key from the disk and validates that it is valid
func TryLoadKeyFromDisk(pkiPath, name string) (crypto.Signer, error) {
	privateKeyPath := pathForKey(pkiPath, name)

	// Parse the private key from a file
	privKey, err := keyutil.PrivateKeyFromFile(privateKeyPath)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't load the private key file %s", privateKeyPath)
	}

	// Allow RSA and ECDSA formats only
	var key crypto.Signer
	switch k := privKey.(type) {
	case *rsa.PrivateKey:
		key = k
	case *ecdsa.PrivateKey:
		key = k
	default:
		return nil, errors.Errorf("the private key file %s is neither in RSA nor ECDSA format", privateKeyPath)
	}

	return key, nil
}

// TryLoadCSRAndKeyFromDisk tries to load the CSR and key from the disk
func TryLoadCSRAndKeyFromDisk(pkiPath, name string) (*x509.CertificateRequest, crypto.Signer, error) {
	csr, err := TryLoadCSRFromDisk(pkiPath, name)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not load CSR file")
	}

	key, err := TryLoadKeyFromDisk(pkiPath, name)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not load key file")
	}

	return csr, key, nil
}

// TryLoadPrivatePublicKeyFromDisk tries to load the key from the disk and validates that it is valid
func TryLoadPrivatePublicKeyFromDisk(pkiPath, name string) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKeyPath := pathForKey(pkiPath, name)

	// Parse the private key from a file
	privKey, err := keyutil.PrivateKeyFromFile(privateKeyPath)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "couldn't load the private key file %s", privateKeyPath)
	}

	publicKeyPath := pathForPublicKey(pkiPath, name)

	// Parse the public key from a file
	pubKeys, err := keyutil.PublicKeysFromFile(publicKeyPath)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "couldn't load the public key file %s", publicKeyPath)
	}

	// Allow RSA format only
	k, ok := privKey.(*rsa.PrivateKey)
	if !ok {
		return nil, nil, errors.Errorf("the private key file %s isn't in RSA format", privateKeyPath)
	}

	p := pubKeys[0].(*rsa.PublicKey)

	return k, p, nil
}

// TryLoadCSRFromDisk tries to load the CSR from the disk
func TryLoadCSRFromDisk(pkiPath, name string) (*x509.CertificateRequest, error) {
	csrPath := pathForCSR(pkiPath, name)

	csr, err := CertificateRequestFromFile(csrPath)
	if err != nil {
		return nil, errors.Wrapf(err, "could not load the CSR %s", csrPath)
	}

	return csr, nil
}

// CertificateRequestFromFile returns the CertificateRequest from a given PEM-encoded file.
// Returns an error if the file could not be read or if the CSR could not be parsed.
func CertificateRequestFromFile(file string) (*x509.CertificateRequest, error) {
	pemBlock, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read file")
	}

	csr, err := parseCSRPEM(pemBlock)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading certificate request file %s", file)
	}
	return csr, nil
}

func parseCSRPEM(pemCSR []byte) (*x509.CertificateRequest, error) {
	block, _ := pem.Decode(pemCSR)
	if block == nil {
		return nil, errors.Error("data doesn't contain a valid certificate request")
	}

	if block.Type != certutil.CertificateRequestBlockType {
		return nil, errors.Errorf("expected block type %q, but PEM had type %q", certutil.CertificateRequestBlockType, block.Type)
	}

	return x509.ParseCertificateRequest(block.Bytes)
}

// CertOrKeyExist returns a boolean whether the cert or the key exists
func CertOrKeyExist(pkiPath, name string) bool {
	certificatePath, privateKeyPath := PathsForCertAndKey(pkiPath, name)

	_, certErr := os.Stat(certificatePath)
	_, keyErr := os.Stat(privateKeyPath)
	if os.IsNotExist(certErr) && os.IsNotExist(keyErr) {
		// The cert and the key do not exist
		return false
	}

	// Both files exist or one of them
	return true
}

// CSROrKeyExist returns true if one of the CSR or key exists
func CSROrKeyExist(csrDir, name string) bool {
	csrPath := pathForCSR(csrDir, name)
	keyPath := pathForKey(csrDir, name)

	_, csrErr := os.Stat(csrPath)
	_, keyErr := os.Stat(keyPath)

	return !(os.IsNotExist(csrErr) && os.IsNotExist(keyErr))
}

// EncodePublicKeyPEM returns PEM-encoded public data
func EncodePublicKeyPEM(key crypto.PublicKey) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return []byte{}, err
	}
	block := pem.Block{
		Type:  PublicKeyBlockType,
		Bytes: der,
	}
	return pem.EncodeToMemory(&block), nil
}

// EncodeCSRPEM returns PEM-encoded CSR data
func EncodeCSRPEM(csr *x509.CertificateRequest) []byte {
	block := pem.Block{
		Type:  certutil.CertificateRequestBlockType,
		Bytes: csr.Raw,
	}
	return pem.EncodeToMemory(&block)
}

// EncodeCertPEM returns PEM-endcoded certificate data
func EncodeCertPEM(cert *x509.Certificate) []byte {
	block := pem.Block{
		Type:  CertificateBlockType,
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(&block)
}

// NewPrivateKey creates an RSA private key
func NewPrivateKey(keyType x509.PublicKeyAlgorithm) (crypto.Signer, error) {
	if keyType == x509.ECDSA {
		return ecdsa.GenerateKey(elliptic.P256(), cryptorand.Reader)
	}

	return rsa.GenerateKey(cryptorand.Reader, rsaKeySize)
}

// PathsForCertAndKey returns the paths for the certificate and key given the path and basename.
func PathsForCertAndKey(pkiPath, name string) (string, string) {
	return pathForCert(pkiPath, name), pathForKey(pkiPath, name)
}

var (
	pathForCert = func(pkiPath, name string) string {
		return filepath.Join(pkiPath, fmt.Sprintf("%s.crt", name))
	}

	pathForKey = func(pkiPath, name string) string {
		return filepath.Join(pkiPath, fmt.Sprintf("%s.key", name))
	}

	pathForPublicKey = func(pkiPath, name string) string {
		return filepath.Join(pkiPath, fmt.Sprintf("%s.pub", name))
	}

	pathForCSR = func(pkiPath, name string) string {
		return filepath.Join(pkiPath, fmt.Sprintf("%s.csr", name))
	}
)

func SetPathForCert(sf func(pkiPath, name string) string) {
	pathForCert = sf
}

func SetPathForKey(sf func(pkiPath, name string) string) {
	pathForKey = sf
}

func SetPathForPublicKey(sf func(pkiPath, name string) string) {
	pathForPublicKey = sf
}

func SetPathForCSR(sf func(pkiPath, name string) string) {
	pathForCSR = sf
}

// NewSignedCert creates a signed certificate using the given CA certificate and key
func NewSignedCert(cfg *CertConfig, key crypto.Signer, caCert *x509.Certificate, caKey crypto.Signer) (*x509.Certificate, error) {
	serial, err := cryptorand.Int(cryptorand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	if len(cfg.CommonName) == 0 {
		return nil, errors.Error("must specify a CommonName")
	}
	if len(cfg.Usages) == 0 {
		return nil, errors.Error("must specify at least one ExtKeyUsage")
	}

	RemoveDuplicateAltNames(&cfg.AltNames)

	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: cfg.Organization,
		},
		DNSNames:     cfg.AltNames.DNSNames,
		IPAddresses:  cfg.AltNames.IPs,
		SerialNumber: serial,
		NotBefore:    caCert.NotBefore,
		NotAfter:     time.Now().Add(CertificateValidity).UTC(),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  cfg.Usages,
	}
	certDERBytes, err := x509.CreateCertificate(cryptorand.Reader, &certTmpl, caCert, key.Public(), caKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

// RemoveDuplicateAltNames removes duplicate items in altNames.
func RemoveDuplicateAltNames(altNames *certutil.AltNames) {
	if altNames == nil {
		return
	}

	if altNames.DNSNames != nil {
		altNames.DNSNames = sets.NewString(altNames.DNSNames...).List()
	}

	ipsKeys := make(map[string]struct{})
	var ips []net.IP
	for _, one := range altNames.IPs {
		if _, ok := ipsKeys[one.String()]; !ok {
			ipsKeys[one.String()] = struct{}{}
			ips = append(ips, one)
		}
	}
	altNames.IPs = ips
}
