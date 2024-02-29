/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package security

import (
	"crypto/tls"
	"errors"
	"flag"
	"net"
	"os"
	"sync"
)

var dmHome = flag.String("DM_HOME", "", "Where DMDB installed")
var flagLock = sync.Mutex{}

func NewTLSFromTCP(conn net.Conn, sslCertPath string, sslKeyPath string, user string) (*tls.Conn, error) {
	if sslCertPath == "" && sslKeyPath == "" {
		// 为什么从os.getEnv改为flag? 参照JDBC，它通过System.getProperty()获取命令中的-DDM_HOME=值
		// flag非协程安全，内部存在并发写map的操作
		func () {
			flagLock.Lock()
			defer flagLock.Unlock()
			flag.Parse()
		}()
		separator := string(os.PathSeparator)
		if *dmHome != "" {
			sslCertPath = *dmHome + separator + "bin" + separator + "client_ssl" + separator +
				user + separator + "client-cert.pem"
			sslKeyPath = *dmHome + separator + "bin" + separator + "client_ssl" + separator +
				user + separator + "client-key.pem"
		} else {
			return nil, errors.New("sslCertPath and sslKeyPath can not be empty!")
		}
	}
	cer, err := tls.LoadX509KeyPair(sslCertPath, sslKeyPath)
	if err != nil {
		return nil, err
	}
	conf := &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cer},
	}
	tlsConn := tls.Client(conn, conf)
	if err := tlsConn.Handshake(); err != nil {
		return nil, err
	}
	return tlsConn, nil
}
