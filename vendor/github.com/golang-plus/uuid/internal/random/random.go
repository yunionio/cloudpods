package random

import (
	"crypto/rand"

	"github.com/golang-plus/uuid/internal"

	"github.com/golang-plus/errors"
)

// NewUUID returns a new randomly uuid.
func NewUUID() ([]byte, error) {
	uuid := make([]byte, 16)
	n, err := rand.Read(uuid[:])
	if err != nil {
		return nil, errors.Wrap(err, "could not generate random bytes")
	}
	if n != len(uuid) {
		return nil, errors.New("could not generate random bytes with 16 length")
	}

	// set version(v4)
	internal.SetVersion(uuid, internal.VersionRandom)
	// set layout(RFC4122)
	internal.SetLayout(uuid, internal.LayoutRFC4122)

	return uuid, nil
}
