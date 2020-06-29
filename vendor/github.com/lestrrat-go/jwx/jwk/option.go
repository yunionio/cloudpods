package jwk

import (
	"crypto"
	"net/http"

	"github.com/lestrrat-go/jwx/internal/option"
)

type Option = option.Interface

const (
	optkeyHTTPClient     = `http-client`
	optkeyThumbprintHash = `thumbprint-hash`
)

func WithHTTPClient(cl *http.Client) Option {
	return option.New(optkeyHTTPClient, cl)
}

func WithThumbprintHash(h crypto.Hash) Option {
	return option.New(optkeyThumbprintHash, h)
}
