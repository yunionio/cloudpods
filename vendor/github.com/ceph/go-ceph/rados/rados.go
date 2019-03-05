package rados

// #cgo LDFLAGS: -lrados
// #include <errno.h>
// #include <stdlib.h>
// #include <rados/librados.h>
import "C"

import (
	"fmt"
	"unsafe"
)

type RadosError int

func (e RadosError) Error() string {
	return fmt.Sprintf("rados: %s", C.GoString(C.strerror(C.int(-e))))
}

var RadosAllNamespaces = C.LIBRADOS_ALL_NSPACES

var RadosErrorNotFound = RadosError(-C.ENOENT)
var RadosErrorPermissionDenied = RadosError(-C.EPERM)

func GetRadosError(err int) error {
	if err == 0 {
		return nil
	}
	return RadosError(err)
}

// Version returns the major, minor, and patch components of the version of
// the RADOS library linked against.
func Version() (int, int, int) {
	var c_major, c_minor, c_patch C.int
	C.rados_version(&c_major, &c_minor, &c_patch)
	return int(c_major), int(c_minor), int(c_patch)
}

func makeConn() *Conn {
	return &Conn{connected: false}
}

func newConn(user *C.char) (*Conn, error) {
	conn := makeConn()
	ret := C.rados_create(&conn.cluster, user)

	if ret == 0 {
		return conn, nil
	} else {
		return nil, RadosError(int(ret))
	}
}

// NewConn creates a new connection object. It returns the connection and an
// error, if any.
func NewConn() (*Conn, error) {
	return newConn(nil)
}

// NewConnWithUser creates a new connection object with a custom username.
// It returns the connection and an error, if any.
func NewConnWithUser(user string) (*Conn, error) {
	c_user := C.CString(user)
	defer C.free(unsafe.Pointer(c_user))
	return newConn(c_user)
}

// NewConnWithClusterAndUser creates a new connection object for a specific cluster and username.
// It returns the connection and an error, if any.
func NewConnWithClusterAndUser(clusterName string, userName string) (*Conn, error) {
	c_cluster_name := C.CString(clusterName)
	defer C.free(unsafe.Pointer(c_cluster_name))

	c_name := C.CString(userName)
	defer C.free(unsafe.Pointer(c_name))

	conn := makeConn()
	ret := C.rados_create2(&conn.cluster, c_cluster_name, c_name, 0)
	if ret == 0 {
		return conn, nil
	} else {
		return nil, RadosError(int(ret))
	}
}
