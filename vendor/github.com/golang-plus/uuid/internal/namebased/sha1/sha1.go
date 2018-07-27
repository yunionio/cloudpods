package sha1

import (
	"crypto/sha1"

	"github.com/golang-plus/uuid/internal"

	"github.com/golang-plus/errors"
)

// NewUUID returns a new name-based uses SHA-1 hashing uuid.
func NewUUID(namespace, name string) ([]byte, error) {
	hash := sha1.New()
	_, err := hash.Write([]byte(namespace))
	if err != nil {
		return nil, errors.Wrapf(err, "could not compute hash value for namespace %q", namespace)
	}
	_, err = hash.Write([]byte(name))
	if err != nil {
		return nil, errors.Wrapf(err, "could not compute hash value for name %q", name)
	}

	sum := hash.Sum(nil)

	uuid := make([]byte, 16)
	copy(uuid, sum)

	// set version(v5)
	internal.SetVersion(uuid, internal.VersionNameBasedSHA1)
	// set layout(RFC4122)
	internal.SetLayout(uuid, internal.LayoutRFC4122)

	return uuid, nil
}
