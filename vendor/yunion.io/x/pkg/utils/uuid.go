package utils

import (
	"crypto/rand"
	"encoding/hex"

	"yunion.io/x/log"
)

func GenRequestId(bytes int) string {
	b := make([]byte, bytes)
	_, err := rand.Read(b)
	if err != nil {
		log.Errorf("Fail to generate Request Id: %s", err)
		return ""
	}
	return hex.EncodeToString(b)
}
