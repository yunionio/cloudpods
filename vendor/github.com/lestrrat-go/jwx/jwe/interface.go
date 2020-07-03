package jwe

import (
	"github.com/lestrrat-go/iter/mapiter"
	"github.com/lestrrat-go/jwx/buffer"
	"github.com/lestrrat-go/jwx/internal/iter"
	"github.com/lestrrat-go/jwx/internal/option"
	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwe/internal/keyenc"
	"github.com/lestrrat-go/jwx/jwe/internal/keygen"
)

const (
	optkeyPrettyJSONFormat = "optkeyPrettyJSONFormat"
)

// Recipient holds the encrypted key and hints to decrypt the key
type Recipient interface {
	Headers() Headers
	EncryptedKey() buffer.Buffer
	SetHeaders(Headers) error
	SetEncryptedKey(interface{}) error
}

type stdRecipient struct {
	headers      Headers
	encryptedKey buffer.Buffer
}

// Message contains the entire encrypted JWE message
type Message struct {
	authenticatedData    *buffer.Buffer
	cipherText           *buffer.Buffer
	initializationVector *buffer.Buffer
	protectedHeaders     Headers
	recipients           []Recipient
	tag                  *buffer.Buffer
	unprotectedHeaders   Headers
}

// contentEncrypter encrypts the content using the content using the
// encrypted key
type contentEncrypter interface {
	Algorithm() jwa.ContentEncryptionAlgorithm
	Encrypt([]byte, []byte, []byte) ([]byte, []byte, []byte, error)
}

type encryptCtx struct {
	contentEncrypter contentEncrypter
	generator        keygen.Generator
	keyEncrypters    []keyenc.Encrypter
	compress         jwa.CompressionAlgorithm
}

// populater is an interface for things that may modify the
// JWE header. e.g. ByteWithECPrivateKey
type populater interface {
	Populate(keygen.Setter) error
}

type Visitor = iter.MapVisitor
type VisitorFunc = iter.MapVisitorFunc
type HeaderPair = mapiter.Pair
type Iterator = mapiter.Iterator
type Option = option.Interface
