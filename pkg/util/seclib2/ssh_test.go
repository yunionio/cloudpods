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
	"crypto/ed25519"
	"crypto/rand"
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

func TestGenerateECDSASHAP521SSHKeypair(t *testing.T) {
	priv, pub, _ := GenerateECDSASHAP521SSHKeypair()
	t.Logf("%s", priv)
	t.Logf("%s", pub)
}

func TestGenerateED25519SSHKeypair(t *testing.T) {
	priv, pub, _ := GenerateED25519SSHKeypair()
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

func TestEd25519DecryptEncrypt(t *testing.T) {
	privateKey, publicKey, err := GenerateED25519SSHKeypair()
	if err != nil {
		t.Fatalf("fail to generate keypair %s", err)
	}

	secret := "this is a secret string!!!"
	code, err := EncryptBase64(publicKey, secret)
	if err != nil {
		t.Fatalf("ed25519 encrypt error %s", err)
	}
	decrypted, err := DecryptBase64(privateKey, code)
	if err != nil {
		t.Fatalf("ed25519 decrypt error %s", err)
	}
	if secret != decrypted {
		t.Fatalf("ed25519 decrypt/encrypt error! %s != %s", decrypted, secret)
	}
}

func TestEd25519PKCS8DecryptEncrypt(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("fail to generate keypair %s", err)
	}
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("fail to marshal private key %s", err)
	}
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyBytes})
	publicKeySSH, err := exportSshPublicKey(publicKey)
	if err != nil {
		t.Fatalf("fail to export public key %s", err)
	}

	secret := "this is a secret string!!!"
	code, err := EncryptBase64(string(publicKeySSH), secret)
	if err != nil {
		t.Fatalf("ed25519 encrypt error %s", err)
	}
	decrypted, err := DecryptBase64(string(privateKeyPEM), code)
	if err != nil {
		t.Fatalf("ed25519 decrypt error %s", err)
	}
	if secret != decrypted {
		t.Fatalf("ed25519 decrypt/encrypt error! %s != %s", decrypted, secret)
	}
}
