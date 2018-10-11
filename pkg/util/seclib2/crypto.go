package seclib2

import (
	"crypto/rsa"
	"crypto/rand"
	"crypto/sha1"

	"golang.org/x/crypto/ssh"

	"yunion.io/x/log"
	"fmt"
	"crypto/dsa"
	"crypto/ecdsa"
	"encoding/base64"
	"crypto"
)

func exportSshPublicKey(pubkey interface{}) ([]byte, error) {
	pub, err:= ssh.NewPublicKey(pubkey)
	if err != nil {
		return nil, err
	}
	return ssh.MarshalAuthorizedKey(pub), nil
}

func ssh2CryptoPublicKey(key ssh.PublicKey) crypto.PublicKey {
	cryptoPub := key.(ssh.CryptoPublicKey)
	return cryptoPub.CryptoPublicKey()
}

func ssh2rsaPublicKey(key ssh.PublicKey) *rsa.PublicKey {
	cryptoKey := ssh2CryptoPublicKey(key)
	return cryptoKey.(*rsa.PublicKey)
}

func ssh2dsaPublicKey(key ssh.PublicKey) *dsa.PublicKey {
	cryptoKey := ssh2CryptoPublicKey(key)
	return cryptoKey.(*dsa.PublicKey)
}

func ssh2ecdsaPublicKey(key ssh.PublicKey) *ecdsa.PublicKey {
	cryptoKey := ssh2CryptoPublicKey(key)
	return cryptoKey.(*ecdsa.PublicKey)
}


func Encrypt(publicKey, origData []byte) ([]byte, error) {
	pub, _, _, _, err := ssh.ParseAuthorizedKey(publicKey)
	if err != nil {
		log.Errorf("parse authorized key error %s", err)
		return nil, err
	}
	if pub.Type() == ssh.KeyAlgoRSA {
		return rsa.EncryptOAEP(sha1.New(), rand.Reader, ssh2rsaPublicKey(pub), origData, nil)
	} else {
		var pubInf interface{}
		switch pub.Type() {
		case ssh.KeyAlgoDSA:
			pubInf = ssh2dsaPublicKey(pub)
		case ssh.KeyAlgoECDSA256, ssh.KeyAlgoECDSA384, ssh.KeyAlgoECDSA521:
			pubInf = ssh2ecdsaPublicKey(pub)
		default:
			return nil, fmt.Errorf("unsupported key type %s", pub.Type())
		}
		pubStr, err := exportSshPublicKey(pubInf)
		if err != nil {
			return nil, err
		}
		return encryptAES(pubStr, origData)
	}
}

func Decrypt(privateKey, secret []byte) ([]byte, error) {
	priv, err := ssh.ParseRawPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}
	switch priv.(type) {
	case *rsa.PrivateKey:
		rsaPriv := priv.(*rsa.PrivateKey)
		return rsa.DecryptOAEP(sha1.New(), rand.Reader, rsaPriv, secret, nil)
	case *dsa.PrivateKey:
		dsaPriv := priv.(*dsa.PrivateKey)
		dsaPub, err := exportSshPublicKey(&dsaPriv.PublicKey)
		if err != nil {
			return nil, err
		}
		return decryptAES(dsaPub, secret)
	case *ecdsa.PrivateKey:
		ecdsaPriv := priv.(*ecdsa.PrivateKey)
		ecdsaPub, err := exportSshPublicKey(&ecdsaPriv.PublicKey)
		if err != nil {
			return nil, err
		}
		return decryptAES(ecdsaPub, secret)
	}
	return nil, fmt.Errorf("unsupported")
}

func EncryptBase64(publicKey string, message string) (string, error) {
	secretBytes, err := Encrypt([]byte(publicKey), []byte(message))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(secretBytes), nil
}

func DecryptBase64(privateKey string, secret string) (string, error) {
	secretBytes, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		return "", err
	}
	msgBytes, err := Decrypt([]byte(privateKey), secretBytes)
	if err != nil {
		return "", err
	}
	return string(msgBytes), nil
}