package certs

import (
	"crypto"
	"crypto/x509"
	"fmt"
	"path/filepath"

	certutil "yunion.io/x/onecloud/pkg/util/tls/cert"
	pkiutil "yunion.io/x/onecloud/pkg/util/tls/pki"
	"yunion.io/x/pkg/errors"
)

type configMutatorsFunc func(*certutil.Config) error

// QemuCert represents a cretificate that qemu required.
type QemuCert struct {
	Name           string
	LongName       string
	BaseName       string
	CAName         string
	configMutators []configMutatorsFunc
	config         certutil.Config
}

// GetConfig returns the definition for the given cert.
func (k *QemuCert) GetConfig() (*certutil.Config, error) {
	for _, f := range k.configMutators {
		if err := f(&k.config); err != nil {
			return nil, err
		}
	}

	return &k.config, nil
}

// CreateFromCA makes and writes a certificate using the given CA cert and key.
func (k *QemuCert) CreateFromCA(dir string, caCert *x509.Certificate, caKey crypto.Signer) error {
	cfg, err := k.GetConfig()
	if err != nil {
		return errors.Wrapf(err, "couldn't create %q certificate", k.Name)
	}
	cert, key, err := pkiutil.NewCertAndKey(
		caCert, caKey,
		&pkiutil.CertConfig{
			Config: *cfg,
		})
	if err != nil {
		return err
	}

	if err := writeCertificateFilesIfNotExist(
		dir,
		k.BaseName,
		caCert,
		cert,
		key,
		cfg,
	); err != nil {
		return errors.Wrapf(err, "failed to write or validate certificate %q", k.Name)
	}

	return nil
}

// CreateAsCA creates a certificate authority, writing the files to disk and also returning the created CA so it can be used to sign child certs.
func (k *QemuCert) CreateAsCA(dir string) (*x509.Certificate, crypto.Signer, error) {
	cfg, err := k.GetConfig()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "couldn't get configuration for %q CA certificate", k.Name)
	}
	caCert, caKey, err := pkiutil.NewCertificateAuthority(&pkiutil.CertConfig{Config: *cfg})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "couldn't generate %q CA certificate", k.Name)
	}

	if err := writeCertificateAuthorithyFilesIfNotExist(
		dir,
		k.BaseName,
		caCert,
		caKey,
	); err != nil {
		return nil, nil, errors.Wrapf(err, "couldn't write out %q CA certificate", k.Name)
	}

	return caCert, caKey, nil
}

// CertificateTree is represents a one-level-deep tree, mapping a CA to the certs that depend on it.
type CertificateTree map[*QemuCert]Certificates

// CreateTree creates the CAs, certs signed by the CAs, and writes them all to disk.
func (t CertificateTree) CreateTree(dir string) error {
	for ca, leaves := range t {
		cfg, err := ca.GetConfig()
		if err != nil {
			return err
		}

		var caKey crypto.Signer

		caCert, err := pkiutil.TryLoadCertFromDisk(dir, ca.BaseName)
		if err == nil {
			// Cert exists already, make sure it's valid
			if !caCert.IsCA {
				return errors.Errorf("certificate %q is not a CA", ca.Name)
			}
			// Try and load a CA Key
			caKey, err = pkiutil.TryLoadKeyFromDisk(dir, ca.BaseName)
			if err != nil {
				// If there's no CA key, make sure every certificate exists.
				for _, leaf := range leaves {
					cl := certKeyLocation{
						pkiDir:   dir,
						baseName: leaf.BaseName,
						uxName:   leaf.Name,
					}
					if err := validateSignedCertWithCA(cl, caCert); err != nil {
						return errors.Wrapf(err, "could not load expected certificate %q or validate the existence of key %q for it", leaf.Name, ca.Name)
					}
				}
				continue
			}
			// CA key exists; just use that to create new certificates.
		} else {
			// CACert doesn't already exist, create a new cert and key.
			caCert, caKey, err = pkiutil.NewCertificateAuthority(&pkiutil.CertConfig{Config: *cfg})
			if err != nil {
				return err
			}

			err = writeCertificateAuthorithyFilesIfNotExist(
				dir,
				ca.BaseName,
				caCert,
				caKey,
			)
			if err != nil {
				return err
			}
		}

		for _, leaf := range leaves {
			if err := leaf.CreateFromCA(dir, caCert, caKey); err != nil {
				return err
			}
		}
	}
	return nil
}

// CertificateMap is a flat map of certificates, keyed by Name.
type CertificateMap map[string]*QemuCert

// CertTree returns a one-level-deep tree, mapping a CA cert to an array of certificates that should be signed by it.
func (m CertificateMap) CertTree() (CertificateTree, error) {
	caMap := make(CertificateTree)

	for _, cert := range m {
		if cert.CAName == "" {
			if _, ok := caMap[cert]; !ok {
				caMap[cert] = []*QemuCert{}
			}
		} else {
			ca, ok := m[cert.CAName]
			if !ok {
				return nil, errors.Errorf("certificate %q references unknown CA %q", cert.Name, cert.CAName)
			}
			caMap[ca] = append(caMap[ca], cert)
		}
	}

	return caMap, nil
}

// Certificates is a list of Certificates that should be created
type Certificates []*QemuCert

func (c Certificates) AsMap() CertificateMap {
	certMap := make(map[string]*QemuCert)
	for _, cert := range c {
		certMap[cert.Name] = cert
	}

	return certMap
}

const (
	CACertAndKeyBaseName     = "ca"
	ServerCertBaseName       = "server"
	QemuServerCertCommonName = "qemu-server"
	ClientCertBaseName       = "client"
	QemuClientCertCommonName = "qemu-client"
)

var (
	QemuCertRootCA = QemuCert{
		Name:     "ca",
		LongName: "self-signed CA to provision identities for other qemu actions",
		BaseName: CACertAndKeyBaseName,
		config: certutil.Config{
			CommonName: "qemu",
		},
	}

	QemuCertServer = QemuCert{
		Name:     "server",
		LongName: "certificate for server",
		BaseName: ServerCertBaseName,
		CAName:   "ca",
		config: certutil.Config{
			CommonName: QemuServerCertCommonName,
			Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		},
	}

	QemuCertClient = QemuCert{
		Name:     "client",
		LongName: "certificate for the server to connect to client",
		BaseName: ClientCertBaseName,
		CAName:   "ca",
		config: certutil.Config{
			CommonName:   QemuClientCertCommonName,
			Organization: []string{"system:host"},
			Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		},
	}
)

func init() {
	pkiutil.SetPathForCert(func(pkiPath, name string) string {
		return filepath.Join(pkiPath, fmt.Sprintf("%s-cert.pem", name))
	})

	pkiutil.SetPathForKey(func(pkiPath, name string) string {
		return filepath.Join(pkiPath, fmt.Sprintf("%s-key.pem", name))
	})
}

// GetDefaultCertList returns all of the certificates qemu requires.
func GetDefaultCertList() Certificates {
	return Certificates{
		&QemuCertRootCA,
		&QemuCertServer,
		&QemuCertClient,
	}
}

func setCommonNameToNodeName(commonName string) configMutatorsFunc {
	return func(cc *certutil.Config) error {
		cc.CommonName = commonName
		return nil
	}
}
