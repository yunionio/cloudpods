package bingocloud

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"unicode/utf8"
)

type SAccount struct {
	Id          string `json:"Id"`
	AccessKeyId string `json:"AccessKeyId"`
	Arn         string `json:"Arn"`
	FullName    string `json:"FullName"`
	IsAdmin     string `json:"IsAdmin"`
	IsEncrypted string `json:"IsEncrypted"`
	SecurityKey string `json:"SecurityKey"`
	Status      string `json:"Status"`
	Type        string `json:"Type"`
	UserId      string `json:"UserId"`
	UserName    string `json:"UserName"`
}

func (self *SAccount) decryptKeys(masterSecretKey string) (string, string) {
	if len(self.SecurityKey) == len(masterSecretKey) {
		return self.AccessKeyId, self.SecurityKey
	}

	secretKeyBytes, err := base64.StdEncoding.DecodeString(self.SecurityKey)
	if err != nil {
		return "", ""
	}
	var adminSecretKey = ""
	if len(masterSecretKey) >= 32 {
		adminSecretKey = masterSecretKey[0:32]
	} else {
		adminSecretKey = fmt.Sprintf("%s%032s", masterSecretKey, "")[0:32]
	}
	decryptVal, err := aesCrtCrypt([]byte(secretKeyBytes), []byte(adminSecretKey), make([]byte, 16))
	if err != nil {
		return "", ""
	}

	decryptSecret := fmt.Sprintf("%s", decryptVal)

	if !utf8.ValidString(decryptSecret) {
		return self.AccessKeyId, self.SecurityKey
	}

	return self.AccessKeyId, decryptSecret
}

func aesCrtCrypt(val, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockMode := cipher.NewCTR(block, iv)
	body := make([]byte, len(val))
	blockMode.XORKeyStream(body, val)
	return body, nil
}
