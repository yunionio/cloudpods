package verify

import (
	"crypto"
	"crypto/ecdsa"

	"github.com/lestrrat-go/jwx/internal/pool"
	"github.com/lestrrat-go/jwx/jwa"
	"github.com/pkg/errors"
)

var ecdsaVerifyFuncs = map[jwa.SignatureAlgorithm]ecdsaVerifyFunc{}

func init() {
	algs := map[jwa.SignatureAlgorithm]crypto.Hash{
		jwa.ES256: crypto.SHA256,
		jwa.ES384: crypto.SHA384,
		jwa.ES512: crypto.SHA512,
	}

	for alg, h := range algs {
		ecdsaVerifyFuncs[alg] = makeECDSAVerifyFunc(h)
	}
}

func makeECDSAVerifyFunc(hash crypto.Hash) ecdsaVerifyFunc {
	return func(payload []byte, signature []byte, key *ecdsa.PublicKey) error {
		r := pool.GetBigInt()
		s := pool.GetBigInt()
		defer pool.ReleaseBigInt(r)
		defer pool.ReleaseBigInt(s)

		n := len(signature) / 2
		r.SetBytes(signature[:n])
		s.SetBytes(signature[n:])

		h := hash.New()
		if _, err := h.Write(payload); err != nil {
			return errors.Wrap(err, "failed to write payload using ecdsa")
		}

		if !ecdsa.Verify(key, h.Sum(nil), r, s) {
			return errors.New(`failed to verify signature using ecdsa`)
		}
		return nil
	}
}

func newECDSA(alg jwa.SignatureAlgorithm) (*ECDSAVerifier, error) {
	verifyfn, ok := ecdsaVerifyFuncs[alg]
	if !ok {
		return nil, errors.Errorf(`unsupported algorithm while trying to create ECDSA verifier: %s`, alg)
	}

	return &ECDSAVerifier{
		verify: verifyfn,
	}, nil
}

func (v ECDSAVerifier) Verify(payload []byte, signature []byte, key interface{}) error {
	if key == nil {
		return errors.New(`missing public key while verifying payload`)
	}

	var pubkey *ecdsa.PublicKey
	switch v := key.(type) {
	case ecdsa.PublicKey:
		pubkey = &v
	case *ecdsa.PublicKey:
		pubkey = v
	default:
		return errors.Errorf(`invalid key type %T. *ecdsa.PublicKey is required`, key)
	}

	return v.verify(payload, signature, pubkey)
}
