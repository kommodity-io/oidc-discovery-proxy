package handler

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestResponseCacheSetGet(t *testing.T) {
	t.Parallel()

	cache := newResponseCache(time.Minute)

	_, found := cache.get("/foo")
	if found {
		t.Fatalf("expected cache miss before any set")
	}

	cache.set("/foo", []byte("bar"))

	data, found := cache.get("/foo")
	if !found {
		t.Fatalf("expected cache hit after set")
	}

	if string(data) != "bar" {
		t.Fatalf("got data=%q, want data=%q", data, "bar")
	}
}

func TestResponseCacheExpiry(t *testing.T) {
	t.Parallel()

	cache := newResponseCache(10 * time.Millisecond)

	cache.set("/foo", []byte("bar"))

	if _, found := cache.get("/foo"); !found {
		t.Fatalf("expected cache hit immediately after set")
	}

	time.Sleep(20 * time.Millisecond)

	if _, found := cache.get("/foo"); found {
		t.Fatalf("expected cache miss after TTL expiry")
	}
}

func TestResponseCacheZeroTTLDisablesCaching(t *testing.T) {
	t.Parallel()

	cache := newResponseCache(0)

	cache.set("/foo", []byte("bar"))

	if _, found := cache.get("/foo"); found {
		t.Fatalf("expected zero TTL to disable caching")
	}
}

func TestResponseCacheDistinctKeys(t *testing.T) {
	t.Parallel()

	cache := newResponseCache(time.Minute)

	cache.set("/foo", []byte("foo-data"))
	cache.set("/bar", []byte("bar-data"))

	fooData, fooFound := cache.get("/foo")
	barData, barFound := cache.get("/bar")

	if !fooFound || !barFound {
		t.Fatalf("expected both keys to be cached independently")
	}

	if string(fooData) != "foo-data" || string(barData) != "bar-data" {
		t.Fatalf("got fooData=%q barData=%q, cache entries bled into each other", fooData, barData)
	}
}

// TestResponseCacheConcurrentAccess hammers get/set from many goroutines across a handful of
// keys. Run with -race: the test itself only asserts data integrity (a get never observes a
// partially written entry or a value from the wrong key), but the real point is that the race
// detector must report nothing.
func TestResponseCacheConcurrentAccess(t *testing.T) {
	t.Parallel()

	const (
		numKeys       = 4
		numGoroutines = 50
		numOpsPerG    = 200
	)

	cache := newResponseCache(time.Minute)

	keys := make([]string, numKeys)
	values := make([][]byte, numKeys)

	for keyIndex := range numKeys {
		keys[keyIndex] = fmt.Sprintf("/key-%d", keyIndex)
		values[keyIndex] = fmt.Appendf(nil, "value-for-key-%d", keyIndex)
	}

	var waitGroup sync.WaitGroup

	for workerID := range numGoroutines {
		waitGroup.Add(1)

		go func(seed int) {
			defer waitGroup.Done()

			for op := range numOpsPerG {
				idx := (seed + op) % numKeys

				if op%2 == 0 {
					cache.set(keys[idx], values[idx])

					continue
				}

				data, found := cache.get(keys[idx])
				if !found {
					continue
				}

				if string(data) != string(values[idx]) {
					t.Errorf("get(%q) = %q, want %q", keys[idx], data, values[idx])
				}
			}
		}(workerID)
	}

	waitGroup.Wait()
}
