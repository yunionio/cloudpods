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
	"crypto/dsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"math/big"

	"golang.org/x/crypto/ssh"

	"yunion.io/x/log"
)

func GenerateRSASSHKeypair() (string, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Errorf("generate rsa key error %s", err)
		return "", "", err
	}

	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	privateStr := string(pem.EncodeToMemory(privateKeyPEM))

	pub, err := exportSshPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", err
	}
	publicStr := string(pub)

	return privateStr, publicStr, nil
}

func GenerateDSASSHKeypair() (string, string, error) {
	var privateKey dsa.PrivateKey

	params := &privateKey.Parameters
	err := dsa.GenerateParameters(params, rand.Reader, dsa.L1024N160)
	if err != nil {
		log.Errorf("generateParameter error %s", err)
		return "", "", err
	}
	err = dsa.GenerateKey(&privateKey, rand.Reader)
	if err != nil {
		log.Errorf("generate key error %s", err)
		return "", "", err
	}

	type DsaASN1 struct {
		Version int
		P       *big.Int
		Q       *big.Int
		G       *big.Int
		Pub     *big.Int
		Priv    *big.Int
	}

	k := DsaASN1{}
	k.P = privateKey.P
	k.Q = privateKey.Q
	k.G = privateKey.G
	k.Pub = privateKey.Y
	k.Priv = privateKey.X

	privBytes, err := asn1.Marshal(k)
	if err != nil {
		log.Errorf("asn1 marshal error %s", err)
		return "", "", err
	}

	privateKeyPEM := &pem.Block{Type: "DSA PRIVATE KEY", Bytes: privBytes}
	privateStr := string(pem.EncodeToMemory(privateKeyPEM))

	pub, err := exportSshPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", err
	}
	publicStr := string(pub)

	return privateStr, publicStr, nil
}

func GetPublicKeyScheme(pubkey ssh.PublicKey) string {
	switch pubkey.Type() {
	case ssh.KeyAlgoRSA:
		return "RSA"
	case ssh.KeyAlgoDSA:
		return "DSA"
	case ssh.KeyAlgoECDSA256, ssh.KeyAlgoECDSA384, ssh.KeyAlgoECDSA521:
		return "ECDSA"
		// case ssh.KeyAlgoED25519:
		//	return "ED"
	}
	return "UNKNOWN"
}
