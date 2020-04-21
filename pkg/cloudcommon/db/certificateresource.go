package db

import (
	"time"
)

type SCertificateResourceBase struct {
	Certificate string `create:"required" list:"user" update:"user"`
	PrivateKey  string `create:"required" list:"admin" update:"user"`

	// derived attributes
	PublicKeyAlgorithm      string    `create:"optional" list:"user" update:"user"`
	PublicKeyBitLen         int       `create:"optional" list:"user" update:"user"`
	SignatureAlgorithm      string    `create:"optional" list:"user" update:"user"`
	Fingerprint             string    `create:"optional" list:"user" update:"user"`
	NotBefore               time.Time `create:"optional" list:"user" update:"user"`
	NotAfter                time.Time `create:"optional" list:"user" update:"user"`
	CommonName              string    `create:"optional" list:"user" update:"user"`
	SubjectAlternativeNames string    `create:"optional" list:"user" update:"user"`
}
