package seclib2

import (
	"io/ioutil"
	"crypto/tls"
	"crypto/x509"
	"bytes"

	"yunion.io/x/log"
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
