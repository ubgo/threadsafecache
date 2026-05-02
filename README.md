# threadsafecache

> Generic LRU cache + per-entry TTL. Thread-safe. One dependency
> (`hashicorp/golang-lru/v2`), one type, six methods.

```go
import "github.com/ubgo/threadsafecache"

c, _ := threadsafecache.New[string](
    1024,                     // LRU size
    5 * time.Minute,          // TTL (0 = disabled)
)

c.Set("user:42", payload)

if v, ok := c.Get("user:42"); ok {
    use(v)
}
```

## What you get

- **Generics.** Values are stored without boxing through `interface{}`.
- **LRU bound.** When the cache fills, the least-recently-used entry is
  evicted. (`hashicorp/golang-lru/v2` does the heavy lifting.)
- **TTL.** Entries older than the TTL miss on `Get` and are evicted as a
  side effect of being touched. Set TTL to `0` to disable expiry and use
  the cache as pure LRU.
- **Lock everything.** All operations take the same `sync.Mutex` —
  hashicorp's LRU mutates internal state on `Get` (recency tracking), so
  splitting reads and writes is unsafe in general. The result is simple
  and race-clean.

## API

| Method | Purpose |
|--------|---------|
| `New[T](size, ttl, opts...)` | Construct a cache. `size <= 0` returns `ErrInvalidSize`. |
| `Set(key, value)` | Insert or replace. CreatedAt is reset on replace. |
| `Get(key) (T, bool)` | Returns the value if present and fresh; evicts on expiry. |
| `Remove(key)` | Delete a key. Missing keys are not an error. |
| `Len() int` | Current entry count, including expired-but-not-yet-evicted. |
| `Purge()` | Drop every entry. |

## Options

```go
threadsafecache.New[int](
    1024, ttl,
    threadsafecache.WithClock[int](myFakeClock),  // tests only
)
```

## License

Apache-2.0 — see [`LICENSE`](LICENSE).
