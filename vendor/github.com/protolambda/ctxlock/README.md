# ctxlock

Minimal Go read-write-lock with context-cancellation support.

Usage is like `symc.RWMutex`:
- the `Lock` must not be copied
- `Lock()`/`Unlock()` for write locking
- `RLock()`/`RUnlock()` for read locking
- `LockCtx(ctx) error` for write locking with cancellation
- `RLockCtx(ctx) error` for read locking with cancellation
- panics on invalid usage (unlocks when not locked, or unlock type not matching lock type)

## License

MIT, see [`LICENSE`](./LICENSE) file.
