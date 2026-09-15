package auth

import (
	"github.com/regclient/regclient/types"
)

var (
	// ErrEmptyChallenge indicates an issue with the received challenge in the WWW-Authenticate header
	ErrEmptyChallenge = types.ErrEmptyChallenge
	// ErrInvalidChallenge indicates an issue with the received challenge in the WWW-Authenticate header
	ErrInvalidChallenge = types.ErrInvalidChallenge
	// ErrNoNewChallenge indicates a challenge update did not result in any change
	ErrNoNewChallenge = types.ErrNoNewChallenge
	// ErrNotFound indicates no credentials found for basic auth
	ErrNotFound = types.ErrNotFound
	// ErrNotImplemented returned when method has not been implemented yet
	ErrNotImplemented = types.ErrNotImplemented
	// ErrParseFailure indicates the WWW-Authenticate header could not be parsed
	ErrParseFailure = types.ErrParsingFailed
	// ErrUnauthorized request was not authorized
	ErrUnauthorized = types.ErrHTTPUnauthorized
	// ErrUnsupported indicates the request was unsupported
	ErrUnsupported = types.ErrUnsupported
)
