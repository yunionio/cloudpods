package content_crypt

import (
	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwe/internal/cipher"
	"github.com/lestrrat-go/jwx/jwe/internal/keygen"
)

// Generic encrypts a message by applying all the necessary
// modifications to the keys and the contents
type Generic struct {
	alg     jwa.ContentEncryptionAlgorithm
	keysize int
	tagsize int
	cipher  cipher.ContentCipher
	cekgen  keygen.Generator
}
