package sign

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"

	"github.com/lestrrat-go/jwx/jwa"
	"github.com/pkg/errors"
)

var ecdsaSignFuncs = map[jwa.SignatureAlgorithm]ecdsaSignFunc{}

func init() {
	algs := map[jwa.SignatureAlgorithm]crypto.Hash{
		jwa.ES256: crypto.SHA256,
		jwa.ES384: crypto.SHA384,
		jwa.ES512: crypto.SHA512,
	}

	for alg, h := range algs {
		ecdsaSignFuncs[alg] = makeECDSASignFunc(h)
	}
}

func makeECDSASignFunc(hash crypto.Hash) ecdsaSignFunc {
	return func(payload []byte, key *ecdsa.PrivateKey) ([]byte, error) {
		curveBits := key.Curve.Params().BitSize
		keyBytes := curveBits / 8
		// Curve bits do not need to be a multiple of 8.
		if curveBits%8 > 0 {
			keyBytes++
		}
		h := hash.New()
		if _, err := h.Write(payload); err != nil {
			return nil, errors.Wrap(err, "failed to write payload using ecdsa")
		}
		r, s, err := ecdsa.Sign(rand.Reader, key, h.Sum(nil))
		if err != nil {
			return nil, errors.Wrap(err, "failed to sign payload using ecdsa")
		}

		rBytes := r.Bytes()
		rBytesPadded := make([]byte, keyBytes)
		copy(rBytesPadded[keyBytes-len(rBytes):], rBytes)

		sBytes := s.Bytes()
		sBytesPadded := make([]byte, keyBytes)
		copy(sBytesPadded[keyBytes-len(sBytes):], sBytes)

		out := append(rBytesPadded, sBytesPadded...)
		return out, nil
	}
}

func newECDSA(alg jwa.SignatureAlgorithm) (*ECDSASigner, error) {
	signfn, ok := ecdsaSignFuncs[alg]
	if !ok {
		return nil, errors.Errorf(`unsupported algorithm while trying to create ECDSA signer: %s`, alg)
	}

	return &ECDSASigner{
		alg:  alg,
		sign: signfn,
	}, nil
}

func (s ECDSASigner) Algorithm() jwa.SignatureAlgorithm {
	return s.alg
}

func (s ECDSASigner) Sign(payload []byte, key interface{}) ([]byte, error) {
	if key == nil {
		return nil, errors.New(`missing private key while signing payload`)
	}

	var pubkey *ecdsa.PrivateKey
	switch v := key.(type) {
	case ecdsa.PrivateKey:
		pubkey = &v
	case *ecdsa.PrivateKey:
		pubkey = v
	default:
		return nil, errors.Errorf(`invalid key type %T. *ecdsa.PrivateKey is required`, key)
	}

	return s.sign(payload, pubkey)
}
