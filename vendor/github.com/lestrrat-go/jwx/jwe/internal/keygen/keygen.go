package keygen

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/binary"
	"io"

	"github.com/lestrrat-go/jwx/internal/concatkdf"
	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/pkg/errors"
)

// Bytes returns the byte from this ByteKey
func (k ByteKey) Bytes() []byte {
	return []byte(k)
}

// Size returns the size of the key
func (g Static) Size() int {
	return len(g)
}

// Generate returns the key
func (g Static) Generate() (ByteSource, error) {
	buf := make([]byte, g.Size())
	copy(buf, g)
	return ByteKey(buf), nil
}

// NewRandom creates a new Generator that returns
// random bytes
func NewRandom(n int) Random {
	return Random{keysize: n}
}

// Size returns the key size
func (g Random) Size() int {
	return g.keysize
}

// Generate generates a random new key
func (g Random) Generate() (ByteSource, error) {
	buf := make([]byte, g.keysize)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return nil, errors.Wrap(err, "failed to read from rand.Reader")
	}
	return ByteKey(buf), nil
}

// NewEcdhes creates a new key generator using ECDH-ES
func NewEcdhes(alg jwa.KeyEncryptionAlgorithm, pubkey *ecdsa.PublicKey) (*Ecdhes, error) {
	var keysize int
	switch alg {
	case jwa.ECDH_ES:
		return nil, errors.New("unimplemented")
	case jwa.ECDH_ES_A128KW:
		keysize = 16
	case jwa.ECDH_ES_A192KW:
		keysize = 24
	case jwa.ECDH_ES_A256KW:
		keysize = 32
	default:
		return nil, errors.Errorf("invalid ECDH-ES key generation algorithm (%s)", alg)
	}

	return &Ecdhes{
		algorithm: alg,
		keysize:   keysize,
		pubkey:    pubkey,
	}, nil
}

// Size returns the key size associated with this generator
func (g Ecdhes) Size() int {
	return g.keysize
}

// Generate generates new keys using ECDH-ES
func (g Ecdhes) Generate() (ByteSource, error) {
	priv, err := ecdsa.GenerateKey(g.pubkey.Curve, rand.Reader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate key for ECDH-ES")
	}

	pubinfo := make([]byte, 4)
	binary.BigEndian.PutUint32(pubinfo, uint32(g.keysize)*8)

	z, _ := priv.PublicKey.Curve.ScalarMult(g.pubkey.X, g.pubkey.Y, priv.D.Bytes())
	kdf := concatkdf.New(crypto.SHA256, []byte(g.algorithm.String()), z.Bytes(), []byte{}, []byte{}, pubinfo, []byte{})
	kek := make([]byte, g.keysize)
	if _, err := kdf.Read(kek); err != nil {
		return nil, errors.Wrap(err, "failed to read kdf")
	}

	return ByteWithECPrivateKey{
		PrivateKey: priv,
		ByteKey:    ByteKey(kek),
	}, nil
}

// HeaderPopulate populates the header with the required EC-DSA public key
// information ('epk' key)
func (k ByteWithECPrivateKey) Populate(h Setter) error {
	key, err := jwk.New(&k.PrivateKey.PublicKey)
	if err != nil {
		return errors.Wrap(err, "failed to create JWK")
	}

	if err := h.Set("epk", key); err != nil {
		return errors.Wrap(err, "failed to write header")
	}
	return nil
}
