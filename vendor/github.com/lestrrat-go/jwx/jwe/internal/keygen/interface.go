package keygen

import (
	"crypto/ecdsa"

	"github.com/lestrrat-go/jwx/jwa"
)

type Generator interface {
	Size() int
	Generate() (ByteSource, error)
}

// StaticKeyGenerate uses a static byte buffer to provide keys.
type Static []byte

// RandomKeyGenerate generates random keys
type Random struct {
	keysize int
}

// EcdhesKeyGenerate generates keys using ECDH-ES algorithm
type Ecdhes struct {
	algorithm jwa.KeyEncryptionAlgorithm
	keysize   int
	pubkey    *ecdsa.PublicKey
}

// ByteKey is a generated key that only has the key's byte buffer
// as its instance data. If a ke needs to do more, such as providing
// values to be set in a JWE header, that key type wraps a ByteKey
type ByteKey []byte

// ByteWithECPrivateKey holds the EC-DSA private key that generated
// the key along with the key itself. This is required to set the
// proper values in the JWE headers
type ByteWithECPrivateKey struct {
	ByteKey
	PrivateKey *ecdsa.PrivateKey
}

// ByteSource is an interface for things that return a byte sequence.
// This is used for KeyGenerator so that the result of computations can
// carry more than just the generate byte sequence.
type ByteSource interface {
	Bytes() []byte
}

type Setter interface {
	Set(string, interface{}) error
}
