package threadsafecache

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestNew_InvalidSize(t *testing.T) {
	if _, err := New[int](0, time.Second); err == nil {
		t.Fatal("size 0 should error")
	}
	if _, err := New[int](-1, time.Second); err == nil {
		t.Fatal("size -1 should error")
	}
}

func TestSetGet(t *testing.T) {
	c, err := New[string](8, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	c.Set("k", "v")
	v, ok := c.Get("k")
	if !ok || v != "v" {
		t.Fatalf("Get(k) = %q, %v", v, ok)
	}
	if _, ok := c.Get("missing"); ok {
		t.Fatal("Get(missing) should miss")
	}
}

func TestExpiry(t *testing.T) {
	now := time.Unix(1000, 0)
	clock := &struct {
		mu sync.Mutex
		t  time.Time
	}{t: now}
	tick := func() time.Time {
		clock.mu.Lock()
		defer clock.mu.Unlock()
		return clock.t
	}

	c, err := New[int](8, 10*time.Second, WithClock[int](tick))
	if err != nil {
		t.Fatal(err)
	}
	c.Set("k", 1)

	clock.mu.Lock()
	clock.t = clock.t.Add(5 * time.Second)
	clock.mu.Unlock()
	if v, ok := c.Get("k"); !ok || v != 1 {
		t.Fatalf("within TTL: Get = %d, %v", v, ok)
	}

	clock.mu.Lock()
	clock.t = clock.t.Add(20 * time.Second)
	clock.mu.Unlock()
	if _, ok := c.Get("k"); ok {
		t.Fatal("after TTL: Get should miss")
	}
	if c.Len() != 0 {
		t.Fatalf("expired entry not evicted: Len = %d", c.Len())
	}
}

func TestZeroTTL_NoExpiry(t *testing.T) {
	c, err := New[int](4, 0)
	if err != nil {
		t.Fatal(err)
	}
	c.Set("k", 99)
	time.Sleep(10 * time.Millisecond)
	if v, ok := c.Get("k"); !ok || v != 99 {
		t.Fatalf("Get with ttl=0: %d, %v", v, ok)
	}
}

func TestRemove(t *testing.T) {
	c, _ := New[string](4, time.Minute)
	c.Set("k", "v")
	c.Remove("k")
	if _, ok := c.Get("k"); ok {
		t.Fatal("Remove did not delete")
	}
	c.Remove("missing")
}

func TestLRUEviction(t *testing.T) {
	c, _ := New[int](2, time.Minute)
	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3)
	if _, ok := c.Get("a"); ok {
		t.Fatal("a should have been LRU-evicted")
	}
	if v, ok := c.Get("b"); !ok || v != 2 {
		t.Fatalf("b: %d %v", v, ok)
	}
	if v, ok := c.Get("c"); !ok || v != 3 {
		t.Fatalf("c: %d %v", v, ok)
	}
}

func TestPurge(t *testing.T) {
	c, _ := New[int](4, time.Minute)
	c.Set("a", 1)
	c.Set("b", 2)
	c.Purge()
	if c.Len() != 0 {
		t.Fatalf("Purge: Len = %d", c.Len())
	}
}

func TestConcurrentSetGet(t *testing.T) {
	c, _ := New[int](256, time.Minute)
	const workers = 16
	const iters = 500

	var wg sync.WaitGroup
	wg.Add(workers * 2)
	for w := 0; w < workers; w++ {
		go func(w int) {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				c.Set("k"+strconv.Itoa((w*iters+i)%128), i)
			}
		}(w)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				_, _ = c.Get("k" + strconv.Itoa(i%128))
			}
		}(w)
	}
	wg.Wait()
}
