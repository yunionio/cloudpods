package rados

// #cgo LDFLAGS: -lrados
// #include <errno.h>
// #include <stdlib.h>
// #include <rados/librados.h>
//
// char* nextChunk(char **idx) {
// 	char *copy;
// 	copy = strdup(*idx);
// 	*idx += strlen(*idx) + 1;
// 	return copy;
// }
//
// #if __APPLE__
// #define ceph_time_t __darwin_time_t
// #define ceph_suseconds_t __darwin_suseconds_t
// #elif __GLIBC__
// #define ceph_time_t __time_t
// #define ceph_suseconds_t __suseconds_t
// #else
// #define ceph_time_t time_t
// #define ceph_suseconds_t suseconds_t
// #endif
import "C"

import (
	"syscall"
	"time"
	"unsafe"
)

// PoolStat represents Ceph pool statistics.
type PoolStat struct {
	// space used in bytes
	Num_bytes uint64
	// space used in KB
	Num_kb uint64
	// number of objects in the pool
	Num_objects uint64
	// number of clones of objects
	Num_object_clones uint64
	// num_objects * num_replicas
	Num_object_copies              uint64
	Num_objects_missing_on_primary uint64
	// number of objects found on no OSDs
	Num_objects_unfound uint64
	// number of objects replicated fewer times than they should be
	// (but found on at least one OSD)
	Num_objects_degraded uint64
	Num_rd               uint64
	Num_rd_kb            uint64
	Num_wr               uint64
	Num_wr_kb            uint64
}

// ObjectStat represents an object stat information
type ObjectStat struct {
	// current length in bytes
	Size uint64
	// last modification time
	ModTime time.Time
}

// LockInfo represents information on a current Ceph lock
type LockInfo struct {
	NumLockers int
	Exclusive  bool
	Tag        string
	Clients    []string
	Cookies    []string
	Addrs      []string
}

// IOContext represents a context for performing I/O within a pool.
type IOContext struct {
	ioctx C.rados_ioctx_t
}

// Pointer returns a uintptr representation of the IOContext.
func (ioctx *IOContext) Pointer() uintptr {
	return uintptr(ioctx.ioctx)
}

// SetNamespace sets the namespace for objects within this IO context (pool).
// Setting namespace to a empty or zero length string sets the pool to the default namespace.
func (ioctx *IOContext) SetNamespace(namespace string) {
	var c_ns *C.char
	if len(namespace) > 0 {
		c_ns = C.CString(namespace)
		defer C.free(unsafe.Pointer(c_ns))
	}
	C.rados_ioctx_set_namespace(ioctx.ioctx, c_ns)
}

// Write writes len(data) bytes to the object with key oid starting at byte
// offset offset. It returns an error, if any.
func (ioctx *IOContext) Write(oid string, data []byte, offset uint64) error {
	c_oid := C.CString(oid)
	defer C.free(unsafe.Pointer(c_oid))

	dataPointer := unsafe.Pointer(nil)
	if len(data) > 0 {
		dataPointer = unsafe.Pointer(&data[0])
	}

	ret := C.rados_write(ioctx.ioctx, c_oid,
		(*C.char)(dataPointer),
		(C.size_t)(len(data)),
		(C.uint64_t)(offset))

	return GetRadosError(int(ret))
}

// WriteFull writes len(data) bytes to the object with key oid.
// The object is filled with the provided data. If the object exists,
// it is atomically truncated and then written. It returns an error, if any.
func (ioctx *IOContext) WriteFull(oid string, data []byte) error {
	c_oid := C.CString(oid)
	defer C.free(unsafe.Pointer(c_oid))

	ret := C.rados_write_full(ioctx.ioctx, c_oid,
		(*C.char)(unsafe.Pointer(&data[0])),
		(C.size_t)(len(data)))
	return GetRadosError(int(ret))
}

// Append appends len(data) bytes to the object with key oid.
// The object is appended with the provided data. If the object exists,
// it is atomically appended to. It returns an error, if any.
func (ioctx *IOContext) Append(oid string, data []byte) error {
	c_oid := C.CString(oid)
	defer C.free(unsafe.Pointer(c_oid))

	ret := C.rados_append(ioctx.ioctx, c_oid,
		(*C.char)(unsafe.Pointer(&data[0])),
		(C.size_t)(len(data)))
	return GetRadosError(int(ret))
}

// Read reads up to len(data) bytes from the object with key oid starting at byte
// offset offset. It returns the number of bytes read and an error, if any.
func (ioctx *IOContext) Read(oid string, data []byte, offset uint64) (int, error) {
	c_oid := C.CString(oid)
	defer C.free(unsafe.Pointer(c_oid))

	var buf *C.char
	if len(data) > 0 {
		buf = (*C.char)(unsafe.Pointer(&data[0]))
	}

	ret := C.rados_read(
		ioctx.ioctx,
		c_oid,
		buf,
		(C.size_t)(len(data)),
		(C.uint64_t)(offset))

	if ret >= 0 {
		return int(ret), nil
	} else {
		return 0, GetRadosError(int(ret))
	}
}

// Delete deletes the object with key oid. It returns an error, if any.
func (ioctx *IOContext) Delete(oid string) error {
	c_oid := C.CString(oid)
	defer C.free(unsafe.Pointer(c_oid))

	return GetRadosError(int(C.rados_remove(ioctx.ioctx, c_oid)))
}

// Truncate resizes the object with key oid to size size. If the operation
// enlarges the object, the new area is logically filled with zeroes. If the
// operation shrinks the object, the excess data is removed. It returns an
// error, if any.
func (ioctx *IOContext) Truncate(oid string, size uint64) error {
	c_oid := C.CString(oid)
	defer C.free(unsafe.Pointer(c_oid))

	return GetRadosError(int(C.rados_trunc(ioctx.ioctx, c_oid, (C.uint64_t)(size))))
}

// Destroy informs librados that the I/O context is no longer in use.
// Resources associated with the context may not be freed immediately, and the
// context should not be used again after calling this method.
func (ioctx *IOContext) Destroy() {
	C.rados_ioctx_destroy(ioctx.ioctx)
}

// Stat returns a set of statistics about the pool associated with this I/O
// context.
func (ioctx *IOContext) GetPoolStats() (stat PoolStat, err error) {
	c_stat := C.struct_rados_pool_stat_t{}
	ret := C.rados_ioctx_pool_stat(ioctx.ioctx, &c_stat)
	if ret < 0 {
		return PoolStat{}, GetRadosError(int(ret))
	} else {
		return PoolStat{
			Num_bytes:                      uint64(c_stat.num_bytes),
			Num_kb:                         uint64(c_stat.num_kb),
			Num_objects:                    uint64(c_stat.num_objects),
			Num_object_clones:              uint64(c_stat.num_object_clones),
			Num_object_copies:              uint64(c_stat.num_object_copies),
			Num_objects_missing_on_primary: uint64(c_stat.num_objects_missing_on_primary),
			Num_objects_unfound:            uint64(c_stat.num_objects_unfound),
			Num_objects_degraded:           uint64(c_stat.num_objects_degraded),
			Num_rd:                         uint64(c_stat.num_rd),
			Num_rd_kb:                      uint64(c_stat.num_rd_kb),
			Num_wr:                         uint64(c_stat.num_wr),
			Num_wr_kb:                      uint64(c_stat.num_wr_kb),
		}, nil
	}
}

// GetPoolName returns the name of the pool associated with the I/O context.
func (ioctx *IOContext) GetPoolName() (name string, err error) {
	buf := make([]byte, 128)
	for {
		ret := C.rados_ioctx_get_pool_name(ioctx.ioctx,
			(*C.char)(unsafe.Pointer(&buf[0])), C.unsigned(len(buf)))
		if ret == -C.ERANGE {
			buf = make([]byte, len(buf)*2)
			continue
		} else if ret < 0 {
			return "", GetRadosError(int(ret))
		}
		name = C.GoStringN((*C.char)(unsafe.Pointer(&buf[0])), ret)
		return name, nil
	}
}

// ObjectListFunc is the type of the function called for each object visited
// by ListObjects.
type ObjectListFunc func(oid string)

// ListObjects lists all of the objects in the pool associated with the I/O
// context, and called the provided listFn function for each object, passing
// to the function the name of the object. Call SetNamespace with
// RadosAllNamespaces before calling this function to return objects from all
// namespaces
func (ioctx *IOContext) ListObjects(listFn ObjectListFunc) error {
	var ctx C.rados_list_ctx_t
	ret := C.rados_nobjects_list_open(ioctx.ioctx, &ctx)
	if ret < 0 {
		return GetRadosError(int(ret))
	}
	defer func() { C.rados_nobjects_list_close(ctx) }()

	for {
		var c_entry *C.char
		ret := C.rados_nobjects_list_next(ctx, &c_entry, nil, nil)
		if ret == -C.ENOENT {
			return nil
		} else if ret < 0 {
			return GetRadosError(int(ret))
		}
		listFn(C.GoString(c_entry))
	}
}

// Stat returns the size of the object and its last modification time
func (ioctx *IOContext) Stat(object string) (stat ObjectStat, err error) {
	var c_psize C.uint64_t
	var c_pmtime C.time_t
	c_object := C.CString(object)
	defer C.free(unsafe.Pointer(c_object))

	ret := C.rados_stat(
		ioctx.ioctx,
		c_object,
		&c_psize,
		&c_pmtime)

	if ret < 0 {
		return ObjectStat{}, GetRadosError(int(ret))
	} else {
		return ObjectStat{
			Size:    uint64(c_psize),
			ModTime: time.Unix(int64(c_pmtime), 0),
		}, nil
	}
}

// GetXattr gets an xattr with key `name`, it returns the length of
// the key read or an error if not successful
func (ioctx *IOContext) GetXattr(object string, name string, data []byte) (int, error) {
	c_object := C.CString(object)
	c_name := C.CString(name)
	defer C.free(unsafe.Pointer(c_object))
	defer C.free(unsafe.Pointer(c_name))

	ret := C.rados_getxattr(
		ioctx.ioctx,
		c_object,
		c_name,
		(*C.char)(unsafe.Pointer(&data[0])),
		(C.size_t)(len(data)))

	if ret >= 0 {
		return int(ret), nil
	} else {
		return 0, GetRadosError(int(ret))
	}
}

// Sets an xattr for an object with key `name` with value as `data`
func (ioctx *IOContext) SetXattr(object string, name string, data []byte) error {
	c_object := C.CString(object)
	c_name := C.CString(name)
	defer C.free(unsafe.Pointer(c_object))
	defer C.free(unsafe.Pointer(c_name))

	ret := C.rados_setxattr(
		ioctx.ioctx,
		c_object,
		c_name,
		(*C.char)(unsafe.Pointer(&data[0])),
		(C.size_t)(len(data)))

	return GetRadosError(int(ret))
}

// function that lists all the xattrs for an object, since xattrs are
// a k-v pair, this function returns a map of k-v pairs on
// success, error code on failure
func (ioctx *IOContext) ListXattrs(oid string) (map[string][]byte, error) {
	c_oid := C.CString(oid)
	defer C.free(unsafe.Pointer(c_oid))

	var it C.rados_xattrs_iter_t

	ret := C.rados_getxattrs(ioctx.ioctx, c_oid, &it)
	if ret < 0 {
		return nil, GetRadosError(int(ret))
	}
	defer func() { C.rados_getxattrs_end(it) }()
	m := make(map[string][]byte)
	for {
		var c_name, c_val *C.char
		var c_len C.size_t
		defer C.free(unsafe.Pointer(c_name))
		defer C.free(unsafe.Pointer(c_val))

		ret := C.rados_getxattrs_next(it, &c_name, &c_val, &c_len)
		if ret < 0 {
			return nil, GetRadosError(int(ret))
		}
		// rados api returns a null name,val & 0-length upon
		// end of iteration
		if c_name == nil {
			return m, nil // stop iteration
		}
		m[C.GoString(c_name)] = C.GoBytes(unsafe.Pointer(c_val), (C.int)(c_len))
	}
}

// Remove an xattr with key `name` from object `oid`
func (ioctx *IOContext) RmXattr(oid string, name string) error {
	c_oid := C.CString(oid)
	c_name := C.CString(name)
	defer C.free(unsafe.Pointer(c_oid))
	defer C.free(unsafe.Pointer(c_name))

	ret := C.rados_rmxattr(
		ioctx.ioctx,
		c_oid,
		c_name)

	return GetRadosError(int(ret))
}

// Append the map `pairs` to the omap `oid`
func (ioctx *IOContext) SetOmap(oid string, pairs map[string][]byte) error {
	c_oid := C.CString(oid)
	defer C.free(unsafe.Pointer(c_oid))

	var s C.size_t
	var c *C.char
	ptrSize := unsafe.Sizeof(c)

	c_keys := C.malloc(C.size_t(len(pairs)) * C.size_t(ptrSize))
	c_values := C.malloc(C.size_t(len(pairs)) * C.size_t(ptrSize))
	c_lengths := C.malloc(C.size_t(len(pairs)) * C.size_t(unsafe.Sizeof(s)))

	defer C.free(unsafe.Pointer(c_keys))
	defer C.free(unsafe.Pointer(c_values))
	defer C.free(unsafe.Pointer(c_lengths))

	i := 0
	for key, value := range pairs {
		// key
		c_key_ptr := (**C.char)(unsafe.Pointer(uintptr(c_keys) + uintptr(i)*ptrSize))
		*c_key_ptr = C.CString(key)
		defer C.free(unsafe.Pointer(*c_key_ptr))

		// value and its length
		c_value_ptr := (**C.char)(unsafe.Pointer(uintptr(c_values) + uintptr(i)*ptrSize))

		var c_length C.size_t
		if len(value) > 0 {
			*c_value_ptr = (*C.char)(unsafe.Pointer(&value[0]))
			c_length = C.size_t(len(value))
		} else {
			*c_value_ptr = nil
			c_length = C.size_t(0)
		}

		c_length_ptr := (*C.size_t)(unsafe.Pointer(uintptr(c_lengths) + uintptr(i)*ptrSize))
		*c_length_ptr = c_length

		i++
	}

	op := C.rados_create_write_op()
	C.rados_write_op_omap_set(
		op,
		(**C.char)(c_keys),
		(**C.char)(c_values),
		(*C.size_t)(c_lengths),
		C.size_t(len(pairs)))

	ret := C.rados_write_op_operate(op, ioctx.ioctx, c_oid, nil, 0)
	C.rados_release_write_op(op)

	return GetRadosError(int(ret))
}

// OmapListFunc is the type of the function called for each omap key
// visited by ListOmapValues
type OmapListFunc func(key string, value []byte)

// Iterate on a set of keys and their values from an omap
// `startAfter`: iterate only on the keys after this specified one
// `filterPrefix`: iterate only on the keys beginning with this prefix
// `maxReturn`: iterate no more than `maxReturn` key/value pairs
// `listFn`: the function called at each iteration
func (ioctx *IOContext) ListOmapValues(oid string, startAfter string, filterPrefix string, maxReturn int64, listFn OmapListFunc) error {
	c_oid := C.CString(oid)
	c_start_after := C.CString(startAfter)
	c_filter_prefix := C.CString(filterPrefix)
	c_max_return := C.uint64_t(maxReturn)

	defer C.free(unsafe.Pointer(c_oid))
	defer C.free(unsafe.Pointer(c_start_after))
	defer C.free(unsafe.Pointer(c_filter_prefix))

	op := C.rados_create_read_op()

	var c_iter C.rados_omap_iter_t
	var c_prval C.int
	C.rados_read_op_omap_get_vals2(
		op,
		c_start_after,
		c_filter_prefix,
		c_max_return,
		&c_iter,
		nil,
		&c_prval,
	)

	ret := C.rados_read_op_operate(op, ioctx.ioctx, c_oid, 0)

	if int(ret) != 0 {
		return GetRadosError(int(ret))
	} else if int(c_prval) != 0 {
		return RadosError(int(c_prval))
	}

	for {
		var c_key *C.char
		var c_val *C.char
		var c_len C.size_t

		ret = C.rados_omap_get_next(c_iter, &c_key, &c_val, &c_len)

		if int(ret) != 0 {
			return GetRadosError(int(ret))
		}

		if c_key == nil {
			break
		}

		listFn(C.GoString(c_key), C.GoBytes(unsafe.Pointer(c_val), C.int(c_len)))
	}

	C.rados_omap_get_end(c_iter)
	C.rados_release_read_op(op)

	return nil
}

// Fetch a set of keys and their values from an omap and returns then as a map
// `startAfter`: retrieve only the keys after this specified one
// `filterPrefix`: retrieve only the keys beginning with this prefix
// `maxReturn`: retrieve no more than `maxReturn` key/value pairs
func (ioctx *IOContext) GetOmapValues(oid string, startAfter string, filterPrefix string, maxReturn int64) (map[string][]byte, error) {
	omap := map[string][]byte{}

	err := ioctx.ListOmapValues(
		oid, startAfter, filterPrefix, maxReturn,
		func(key string, value []byte) {
			omap[key] = value
		},
	)

	return omap, err
}

// Fetch all the keys and their values from an omap and returns then as a map
// `startAfter`: retrieve only the keys after this specified one
// `filterPrefix`: retrieve only the keys beginning with this prefix
// `iteratorSize`: internal number of keys to fetch during a read operation
func (ioctx *IOContext) GetAllOmapValues(oid string, startAfter string, filterPrefix string, iteratorSize int64) (map[string][]byte, error) {
	omap := map[string][]byte{}
	omapSize := 0

	for {
		err := ioctx.ListOmapValues(
			oid, startAfter, filterPrefix, iteratorSize,
			func(key string, value []byte) {
				omap[key] = value
				startAfter = key
			},
		)

		if err != nil {
			return omap, err
		}

		// End of omap
		if len(omap) == omapSize {
			break
		}

		omapSize = len(omap)
	}

	return omap, nil
}

// Remove the specified `keys` from the omap `oid`
func (ioctx *IOContext) RmOmapKeys(oid string, keys []string) error {
	c_oid := C.CString(oid)
	defer C.free(unsafe.Pointer(c_oid))

	var c *C.char
	ptrSize := unsafe.Sizeof(c)

	c_keys := C.malloc(C.size_t(len(keys)) * C.size_t(ptrSize))
	defer C.free(unsafe.Pointer(c_keys))

	i := 0
	for _, key := range keys {
		c_key_ptr := (**C.char)(unsafe.Pointer(uintptr(c_keys) + uintptr(i)*ptrSize))
		*c_key_ptr = C.CString(key)
		defer C.free(unsafe.Pointer(*c_key_ptr))
		i++
	}

	op := C.rados_create_write_op()
	C.rados_write_op_omap_rm_keys(
		op,
		(**C.char)(c_keys),
		C.size_t(len(keys)))

	ret := C.rados_write_op_operate(op, ioctx.ioctx, c_oid, nil, 0)
	C.rados_release_write_op(op)

	return GetRadosError(int(ret))
}

// Clear the omap `oid`
func (ioctx *IOContext) CleanOmap(oid string) error {
	c_oid := C.CString(oid)
	defer C.free(unsafe.Pointer(c_oid))

	op := C.rados_create_write_op()
	C.rados_write_op_omap_clear(op)

	ret := C.rados_write_op_operate(op, ioctx.ioctx, c_oid, nil, 0)
	C.rados_release_write_op(op)

	return GetRadosError(int(ret))
}

type Iter struct {
	ctx       C.rados_list_ctx_t
	err       error
	entry     string
	namespace string
}

type IterToken uint32

// Return a Iterator object that can be used to list the object names in the current pool
func (ioctx *IOContext) Iter() (*Iter, error) {
	iter := Iter{}
	if cerr := C.rados_nobjects_list_open(ioctx.ioctx, &iter.ctx); cerr < 0 {
		return nil, GetRadosError(int(cerr))
	}
	return &iter, nil
}

// Returns a token marking the current position of the iterator. To be used in combination with Iter.Seek()
func (iter *Iter) Token() IterToken {
	return IterToken(C.rados_nobjects_list_get_pg_hash_position(iter.ctx))
}

func (iter *Iter) Seek(token IterToken) {
	C.rados_nobjects_list_seek(iter.ctx, C.uint32_t(token))
}

// Next retrieves the next object name in the pool/namespace iterator.
// Upon a successful invocation (return value of true), the Value method should
// be used to obtain the name of the retrieved object name. When the iterator is
// exhausted, Next returns false. The Err method should used to verify whether the
// end of the iterator was reached, or the iterator received an error.
//
// Example:
//	iter := pool.Iter()
//	defer iter.Close()
//	for iter.Next() {
//		fmt.Printf("%v\n", iter.Value())
//	}
//	return iter.Err()
//
func (iter *Iter) Next() bool {
	var c_entry *C.char
	var c_namespace *C.char
	if cerr := C.rados_nobjects_list_next(iter.ctx, &c_entry, nil, &c_namespace); cerr < 0 {
		iter.err = GetRadosError(int(cerr))
		return false
	}
	iter.entry = C.GoString(c_entry)
	iter.namespace = C.GoString(c_namespace)
	return true
}

// Returns the current value of the iterator (object name), after a successful call to Next.
func (iter *Iter) Value() string {
	if iter.err != nil {
		return ""
	}
	return iter.entry
}

// Returns the namespace associated with the current value of the iterator (object name), after a successful call to Next.
func (iter *Iter) Namespace() string {
	if iter.err != nil {
		return ""
	}
	return iter.namespace
}

// Checks whether the iterator has encountered an error.
func (iter *Iter) Err() error {
	if iter.err == RadosErrorNotFound {
		return nil
	}
	return iter.err
}

// Closes the iterator cursor on the server. Be aware that iterators are not closed automatically
// at the end of iteration.
func (iter *Iter) Close() {
	C.rados_nobjects_list_close(iter.ctx)
}

// Take an exclusive lock on an object.
func (ioctx *IOContext) LockExclusive(oid, name, cookie, desc string, duration time.Duration, flags *byte) (int, error) {
	c_oid := C.CString(oid)
	c_name := C.CString(name)
	c_cookie := C.CString(cookie)
	c_desc := C.CString(desc)

	var c_duration C.struct_timeval
	if duration != 0 {
		tv := syscall.NsecToTimeval(duration.Nanoseconds())
		c_duration = C.struct_timeval{tv_sec: C.ceph_time_t(tv.Sec), tv_usec: C.ceph_suseconds_t(tv.Usec)}
	}

	var c_flags C.uint8_t
	if flags != nil {
		c_flags = C.uint8_t(*flags)
	}

	defer C.free(unsafe.Pointer(c_oid))
	defer C.free(unsafe.Pointer(c_name))
	defer C.free(unsafe.Pointer(c_cookie))
	defer C.free(unsafe.Pointer(c_desc))

	ret := C.rados_lock_exclusive(
		ioctx.ioctx,
		c_oid,
		c_name,
		c_cookie,
		c_desc,
		&c_duration,
		c_flags)

	// 0 on success, negative error code on failure
	// -EBUSY if the lock is already held by another (client, cookie) pair
	// -EEXIST if the lock is already held by the same (client, cookie) pair

	switch ret {
	case 0:
		return int(ret), nil
	case -C.EBUSY:
		return int(ret), nil
	case -C.EEXIST:
		return int(ret), nil
	default:
		return int(ret), RadosError(int(ret))
	}
}

// Take a shared lock on an object.
func (ioctx *IOContext) LockShared(oid, name, cookie, tag, desc string, duration time.Duration, flags *byte) (int, error) {
	c_oid := C.CString(oid)
	c_name := C.CString(name)
	c_cookie := C.CString(cookie)
	c_tag := C.CString(tag)
	c_desc := C.CString(desc)

	var c_duration C.struct_timeval
	if duration != 0 {
		tv := syscall.NsecToTimeval(duration.Nanoseconds())
		c_duration = C.struct_timeval{tv_sec: C.ceph_time_t(tv.Sec), tv_usec: C.ceph_suseconds_t(tv.Usec)}
	}

	var c_flags C.uint8_t
	if flags != nil {
		c_flags = C.uint8_t(*flags)
	}

	defer C.free(unsafe.Pointer(c_oid))
	defer C.free(unsafe.Pointer(c_name))
	defer C.free(unsafe.Pointer(c_cookie))
	defer C.free(unsafe.Pointer(c_tag))
	defer C.free(unsafe.Pointer(c_desc))

	ret := C.rados_lock_shared(
		ioctx.ioctx,
		c_oid,
		c_name,
		c_cookie,
		c_tag,
		c_desc,
		&c_duration,
		c_flags)

	// 0 on success, negative error code on failure
	// -EBUSY if the lock is already held by another (client, cookie) pair
	// -EEXIST if the lock is already held by the same (client, cookie) pair

	switch ret {
	case 0:
		return int(ret), nil
	case -C.EBUSY:
		return int(ret), nil
	case -C.EEXIST:
		return int(ret), nil
	default:
		return int(ret), RadosError(int(ret))
	}
}

// Release a shared or exclusive lock on an object.
func (ioctx *IOContext) Unlock(oid, name, cookie string) (int, error) {
	c_oid := C.CString(oid)
	c_name := C.CString(name)
	c_cookie := C.CString(cookie)

	defer C.free(unsafe.Pointer(c_oid))
	defer C.free(unsafe.Pointer(c_name))
	defer C.free(unsafe.Pointer(c_cookie))

	// 0 on success, negative error code on failure
	// -ENOENT if the lock is not held by the specified (client, cookie) pair

	ret := C.rados_unlock(
		ioctx.ioctx,
		c_oid,
		c_name,
		c_cookie)

	switch ret {
	case 0:
		return int(ret), nil
	case -C.ENOENT:
		return int(ret), nil
	default:
		return int(ret), RadosError(int(ret))
	}
}

// List clients that have locked the named object lock and information about the lock.
// The number of bytes required in each buffer is put in the corresponding size out parameter.
// If any of the provided buffers are too short, -ERANGE is returned after these sizes are filled in.
func (ioctx *IOContext) ListLockers(oid, name string) (*LockInfo, error) {
	c_oid := C.CString(oid)
	c_name := C.CString(name)

	c_tag := (*C.char)(C.malloc(C.size_t(1024)))
	c_clients := (*C.char)(C.malloc(C.size_t(1024)))
	c_cookies := (*C.char)(C.malloc(C.size_t(1024)))
	c_addrs := (*C.char)(C.malloc(C.size_t(1024)))

	var c_exclusive C.int
	c_tag_len := C.size_t(1024)
	c_clients_len := C.size_t(1024)
	c_cookies_len := C.size_t(1024)
	c_addrs_len := C.size_t(1024)

	defer C.free(unsafe.Pointer(c_oid))
	defer C.free(unsafe.Pointer(c_name))
	defer C.free(unsafe.Pointer(c_tag))
	defer C.free(unsafe.Pointer(c_clients))
	defer C.free(unsafe.Pointer(c_cookies))
	defer C.free(unsafe.Pointer(c_addrs))

	ret := C.rados_list_lockers(
		ioctx.ioctx,
		c_oid,
		c_name,
		&c_exclusive,
		c_tag,
		&c_tag_len,
		c_clients,
		&c_clients_len,
		c_cookies,
		&c_cookies_len,
		c_addrs,
		&c_addrs_len)

	splitCString := func(items *C.char, itemsLen C.size_t) []string {
		currLen := 0
		clients := []string{}
		for currLen < int(itemsLen) {
			client := C.GoString(C.nextChunk(&items))
			clients = append(clients, client)
			currLen += len(client) + 1
		}
		return clients
	}

	if ret < 0 {
		return nil, RadosError(int(ret))
	} else {
		return &LockInfo{int(ret), c_exclusive == 1, C.GoString(c_tag), splitCString(c_clients, c_clients_len), splitCString(c_cookies, c_cookies_len), splitCString(c_addrs, c_addrs_len)}, nil
	}
}

// Releases a shared or exclusive lock on an object, which was taken by the specified client.
func (ioctx *IOContext) BreakLock(oid, name, client, cookie string) (int, error) {
	c_oid := C.CString(oid)
	c_name := C.CString(name)
	c_client := C.CString(client)
	c_cookie := C.CString(cookie)

	defer C.free(unsafe.Pointer(c_oid))
	defer C.free(unsafe.Pointer(c_name))
	defer C.free(unsafe.Pointer(c_client))
	defer C.free(unsafe.Pointer(c_cookie))

	// 0 on success, negative error code on failure
	// -ENOENT if the lock is not held by the specified (client, cookie) pair
	// -EINVAL if the client cannot be parsed

	ret := C.rados_break_lock(
		ioctx.ioctx,
		c_oid,
		c_name,
		c_client,
		c_cookie)

	switch ret {
	case 0:
		return int(ret), nil
	case -C.ENOENT:
		return int(ret), nil
	case -C.EINVAL: // -EINVAL
		return int(ret), nil
	default:
		return int(ret), RadosError(int(ret))
	}
}
