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
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestGenerateRSASSHKeypair(t *testing.T) {
	priv, pub, _ := GenerateRSASSHKeypair()
	t.Logf("%s", priv)
	t.Logf("%s", pub)
}

func TestGenerateDSASSHKeypair(t *testing.T) {
	priv, pub, _ := GenerateDSASSHKeypair()
	t.Logf("%s", priv)
	t.Logf("%s", pub)
}

func getPublicKeyPem(privateKey string) ([]byte, error) {
	block, _ := pem.Decode([]byte(privateKey))
	if block == nil {
		return nil, fmt.Errorf("invalid private key")
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	derPkix, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return nil, err
	}

	block = &pem.Block{Type: "PUBLIC KEY", Bytes: derPkix}
	return pem.EncodeToMemory(block), nil
}

func getRSAPublicKeySsh(privateKey string) ([]byte, error) {
	block, _ := pem.Decode([]byte(privateKey))
	if block == nil {
		return nil, fmt.Errorf("invalid private key")
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return exportSshPublicKey(&priv.PublicKey)
}

func getDSAPublicKeySsh(privateKey string) ([]byte, error) {
	block, _ := pem.Decode([]byte(privateKey))
	if block == nil {
		return nil, fmt.Errorf("invalid private key")
	}
	priv, err := ssh.ParseDSAPrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return exportSshPublicKey(&priv.PublicKey)
}

func TestRsaDecryptEncrypt(t *testing.T) {
	privateKey, publicKey, err := GenerateRSASSHKeypair()
	if err != nil {
		t.Errorf("fail to generate keypair %s", err)
		return
	}
	/* publicKey2, err := getPublicKeyPem(privateKey)
	if err != nil {
		t.Errorf("fail to get public key in pem format %s", err)
		return
	} */
	pub3, err := getRSAPublicKeySsh(privateKey)
	if err != nil {
		t.Errorf("fail to get public key in ssh format %s", err)
		return
	}

	if publicKey != string(pub3) {
		t.Errorf("public key mismatch! %s != %s", publicKey, pub3)
		return
	}

	t.Logf("%s", string(pub3))
	// t.Logf("%s", string(publicKey2))

	secret := "this is a secret string!!!"
	code, err := EncryptBase64(publicKey, secret)
	if err != nil {
		t.Errorf("rsa encrypt error %s", err)
		return
	}
	t.Logf("%s", code)
	secret2, err := DecryptBase64(privateKey, code)
	if err != nil {
		t.Errorf("rsa decrypt error %s", err)
		return
	}
	if secret != secret2 {
		t.Errorf("rsa decrypt/encrypt error! %s != %s", secret2, secret)
		return
	}
}

func TestDsaDecryptEncrypt(t *testing.T) {
	privateKey, publicKey, err := GenerateDSASSHKeypair()
	if err != nil {
		t.Errorf("fail to generate keypair %s", err)
		return
	}
	/* publicKey2, err := getPublicKeyPem(privateKey)
	if err != nil {
		t.Errorf("fail to get public key in pem format %s", err)
		return
	} */
	pub3, err := getDSAPublicKeySsh(privateKey)
	if err != nil {
		t.Errorf("fail to get public key in ssh format %s", err)
		return
	}

	if publicKey != string(pub3) {
		t.Errorf("public key mismatch! %s != %s", publicKey, pub3)
		return
	}

	t.Logf("%s", string(pub3))
	// t.Logf("%s", string(publicKey2))

	secret := "this is a secret string!!!"
	code, err := EncryptBase64(publicKey, secret)
	if err != nil {
		t.Errorf("dsa encrypt error %s", err)
		return
	}
	t.Logf("%s", code)
	secret2, err := DecryptBase64(privateKey, code)
	if err != nil {
		t.Errorf("rsa decrypt error %s", err)
		return
	}
	if secret != secret2 {
		t.Errorf("rsa decrypt/encrypt error! %s != %s", secret2, secret)
		return
	}
}
