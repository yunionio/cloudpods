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

package seclib2

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

var CERT_SEP = []byte("-END CERTIFICATE-")

func findCertEndIndex(certBytes []byte) int {
	endpos := bytes.Index(certBytes, CERT_SEP)
	if endpos < 0 {
		return endpos
	}
	endpos += len(CERT_SEP)
	for endpos < len(certBytes) && certBytes[endpos] != '\n' {
		endpos += 1
	}
	return endpos
}

func splitCert(certBytes []byte) [][]byte {
	ret := make([][]byte, 0)
	for {
		endpos := findCertEndIndex(certBytes)
		if endpos > 0 {
			ret = append(ret, certBytes[:endpos])
			for endpos < len(certBytes) && certBytes[endpos] != '-' {
				endpos += 1
			}
			if endpos < len(certBytes) {
				certBytes = certBytes[endpos:]
			} else {
				break
			}
		}
	}
	return ret
}

func InitTLSConfigWithCA(certFile, keyFile, caCertFile string) (*tls.Config, error) {
	cert, err := NewCert(certFile, keyFile, nil)
	if err != nil {
		return nil, err
	}
	cfg := &tls.Config{}

	cfg.RootCAs, err = NewCertPool([]string{caCertFile})
	if err != nil {
		return nil, err
	}
	cfg.Certificates = []tls.Certificate{*cert}
	return cfg, nil
}

// NewCertPool creates x509 certPool with provided CA files.
func NewCertPool(CAFiles []string) (*x509.CertPool, error) {
	certPool := x509.NewCertPool()

	for _, CAFile := range CAFiles {
		pemByte, err := ioutil.ReadFile(CAFile)
		if err != nil {
			return nil, err
		}

		for {
			var block *pem.Block
			block, pemByte = pem.Decode(pemByte)
			if block == nil {
				break
			}
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, err
			}
			certPool.AddCert(cert)
		}
	}

	return certPool, nil
}

// NewCert generates TLS cert by using the given cert,key and parse function.
func NewCert(certfile, keyfile string, parseFunc func([]byte, []byte) (tls.Certificate, error)) (*tls.Certificate, error) {
	cert, err := ioutil.ReadFile(certfile)
	if err != nil {
		return nil, err
	}

	key, err := ioutil.ReadFile(keyfile)
	if err != nil {
		return nil, err
	}

	if parseFunc == nil {
		parseFunc = tls.X509KeyPair
	}

	tlsCert, err := parseFunc(cert, key)
	if err != nil {
		return nil, err
	}
	return &tlsCert, nil
}

func InitTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	allCertPEM, err := ioutil.ReadFile(certFile)
	if err != nil {
		log.Errorf("read tls certfile fail %s", err)
		return nil, err
	}
	certPEMs := splitCert(allCertPEM)
	keyPEM, err := ioutil.ReadFile(keyFile)
	if err != nil {
		log.Errorf("read tls keyfile fail %s", err)
		return nil, err
	}
	cert, err := tls.X509KeyPair(certPEMs[0], keyPEM)
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	for i := 1; i < len(certPEMs); i += 1 {
		caCertPool.AppendCertsFromPEM(certPEMs[i])
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}
	// tlsConfig.ServerName = "CN=*"
	tlsConfig.BuildNameToCertificate()
	return tlsConfig, nil
}

func InitTLSConfigByData(caCertBlock, certPEMBlock, keyPEMBlock []byte) (*tls.Config, error) {
	cert, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	for {
		var block *pem.Block
		block, caCertBlock = pem.Decode(caCertBlock)
		if block == nil {
			break
		}
		caCert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, errors.Wrap(err, "parse caCert data")
		}
		caCertPool.AddCert(caCert)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}
	tlsConfig.BuildNameToCertificate()
	return tlsConfig, nil
}
