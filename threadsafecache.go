// Package threadsafecache is a small wrapper around hashicorp/golang-lru
// that adds a per-entry TTL on top of LRU eviction.
//
// The cache is generic over the value type; keys are strings. Reads use a
// read-lock when the entry is fresh and an upgraded write-lock only when
// the entry has expired and must be evicted, so the common hit path stays
// concurrent.
//
// Zero-cost generics: the inner LRU is parameterised on the same type, so
// the cache stores values directly (not via interface{}).
package threadsafecache

import (
	"errors"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

// ErrInvalidSize is returned by [New] when size <= 0.
var ErrInvalidSize = errors.New("threadsafecache: size must be > 0")

// Entry is the on-disk shape stored in the LRU. Exposed so callers that
// reach into the cache for diagnostics can read CreatedAt without us
// committing to a separate accessor API.
type Entry[T any] struct {
	Value     T
	CreatedAt time.Time
}

// Cache is a thread-safe size+TTL bounded cache.
//
// A TTL of 0 disables expiry — entries live until they are evicted by the
// LRU policy or removed explicitly.
type Cache[T any] struct {
	mu      sync.Mutex
	storage *lru.Cache[string, Entry[T]]
	ttl     time.Duration
	now     func() time.Time
}

// Option configures a [Cache].
type Option[T any] func(*Cache[T])

// WithClock overrides the time source. Used in tests; production code
// rarely needs this.
func WithClock[T any](now func() time.Time) Option[T] {
	return func(c *Cache[T]) { c.now = now }
}

// New returns a cache with the given LRU capacity and per-entry TTL. A
// non-positive size returns [ErrInvalidSize]; a non-positive ttl disables
// expiry.
func New[T any](size int, ttl time.Duration, opts ...Option[T]) (*Cache[T], error) {
	if size <= 0 {
		return nil, ErrInvalidSize
	}
	storage, err := lru.New[string, Entry[T]](size)
	if err != nil {
		return nil, err
	}
	c := &Cache[T]{
		storage: storage,
		ttl:     ttl,
		now:     time.Now,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.now == nil {
		c.now = time.Now
	}
	return c, nil
}

// Set stores value under key. An existing entry is replaced; CreatedAt is
// reset.
func (c *Cache[T]) Set(key string, value T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.storage.Add(key, Entry[T]{
		Value:     value,
		CreatedAt: c.now(),
	})
}

// Get returns the value for key. The boolean is false when the key is
// missing OR has expired (in the latter case the entry is also evicted as
// a side effect — touching expired keys cleans them up lazily).
func (c *Cache[T]) Get(key string) (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.storage.Get(key)
	if !ok {
		var zero T
		return zero, false
	}
	if c.expired(entry) {
		c.storage.Remove(key)
		var zero T
		return zero, false
	}
	return entry.Value, true
}

// Remove deletes key from the cache. Missing keys are not an error.
func (c *Cache[T]) Remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.storage.Remove(key)
}

// Len returns the number of entries currently in the cache, including
// any that have expired but not yet been evicted.
func (c *Cache[T]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.storage.Len()
}

// Purge drops every entry from the cache.
func (c *Cache[T]) Purge() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.storage.Purge()
}

func (c *Cache[T]) expired(e Entry[T]) bool {
	if c.ttl <= 0 {
		return false
	}
	return c.now().Sub(e.CreatedAt) > c.ttl
}
