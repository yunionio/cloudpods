package dcesecurity

import (
	"encoding/binary"
	"os"

	"github.com/golang-plus/uuid/internal"
	"github.com/golang-plus/uuid/internal/timebased"

	"github.com/golang-plus/errors"
)

// NewUUID Generate returns a new DCE security uuid.
func NewUUID(domain Domain) ([]byte, error) {
	uuid, err := timebased.NewUUID()
	if err != nil {
		return nil, err
	}

	switch domain {
	case User:
		uid := os.Getuid()
		binary.BigEndian.PutUint32(uuid[0:], uint32(uid)) // network byte order
	case Group:
		gid := os.Getgid()
		binary.BigEndian.PutUint32(uuid[0:], uint32(gid)) // network byte order
	default:
		return nil, errors.New("domain is invalid")
	}

	// set version(v2)
	internal.SetVersion(uuid, internal.VersionDCESecurity)
	// set layout(RFC4122)
	internal.SetLayout(uuid, internal.LayoutRFC4122)

	return uuid, nil
}
