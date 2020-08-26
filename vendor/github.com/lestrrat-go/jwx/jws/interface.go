package jws

import (
	"github.com/lestrrat-go/iter/mapiter"
	"github.com/lestrrat-go/jwx/internal/iter"
	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwk"
)

type encodedSignature struct {
	Protected string  `json:"protected,omitempty"`
	Headers   Headers `json:"header,omitempty"`
	Signature string  `json:"signature,omitempty"`
}

type encodedMessage struct {
	Payload    string              `json:"payload"`
	Signatures []*encodedSignature `json:"signatures,omitempty"`
}

// PayloadSigner generates signature for the given payload
type PayloadSigner interface {
	Sign([]byte) ([]byte, error)
	Algorithm() jwa.SignatureAlgorithm
	ProtectedHeader() Headers
	PublicHeader() Headers
}

// Message represents a full JWS encoded message. Flattened serialization
// is not supported as a struct, but rather it's represented as a
// Message struct with only one `signature` element.
//
// Do not expect to use the Message object to verify or construct a
// signed payloads with. You should only use this when you want to actually
// want to programmatically view the contents for the full JWS payload.
//
// To sign and verify, use the appropriate `Sign()` nad `Verify()` functions
type Message struct {
	payload    []byte
	signatures []*Signature
}

type Signature struct {
	headers   Headers // Unprotected Headers
	protected Headers // Protected Headers
	signature []byte  // Signature
}

// JWKAcceptor decides which keys can be accepted
// by functions that iterate over a JWK key set.
type JWKAcceptor interface {
	Accept(jwk.Key) bool
}

// JWKAcceptFunc is an implementation of JWKAcceptor
// using a plain function
type JWKAcceptFunc func(jwk.Key) bool

// Accept executes the provided function to determine if the
// given key can be used
func (f JWKAcceptFunc) Accept(key jwk.Key) bool {
	return f(key)
}

// DefaultJWKAcceptor is the default acceptor that is used
// in functions like VerifyWithJWKSet
var DefaultJWKAcceptor = JWKAcceptFunc(func(key jwk.Key) bool {
	if u := key.KeyUsage(); u != "" && u != "enc" && u != "sig" {
		return false
	}
	return true
})

type Visitor = iter.MapVisitor
type VisitorFunc = iter.MapVisitorFunc
type HeaderPair = mapiter.Pair
type Iterator = mapiter.Iterator
