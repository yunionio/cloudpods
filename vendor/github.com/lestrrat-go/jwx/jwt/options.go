package jwt

import (
	"github.com/lestrrat-go/jwx/internal/option"
	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwt/openid"
)

type Option = option.Interface

const (
	optkeyVerify = `verify`
	optkeyToken  = `token`
)

type VerifyParameters interface {
	Algorithm() jwa.SignatureAlgorithm
	Key() interface{}
}

type verifyParams struct {
	alg jwa.SignatureAlgorithm
	key interface{}
}

func (p *verifyParams) Algorithm() jwa.SignatureAlgorithm {
	return p.alg
}

func (p *verifyParams) Key() interface{} {
	return p.key
}

func WithVerify(alg jwa.SignatureAlgorithm, key interface{}) Option {
	return option.New(optkeyVerify, &verifyParams{
		alg: alg,
		key: key,
	})
}

// WithToken specifies the token instance that is used when parsing
// JWT tokens.
func WithToken(t Token) Option {
	return option.New(optkeyToken, t)
}

// WithOpenIDClaims is passed to the various JWT parsing functions, and
// specifies that it should use an instance of `openid.Token` as the
// destination to store the parsed results.
//
// This is exactly equivalent to specifying `jwt.WithToken(openid.New())`
func WithOpenIDClaims() Option {
	return WithToken(openid.New())
}
