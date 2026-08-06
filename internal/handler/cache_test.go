package handler

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestResponseCacheSetGet(t *testing.T) {
	t.Parallel()

	cache := newResponseCache(time.Minute)

	_, _, found := cache.get("/foo")
	if found {
		t.Fatalf("expected cache miss before any set")
	}

	cache.set("/foo", []byte("bar"), http.StatusOK)

	data, statusCode, found := cache.get("/foo")
	if !found {
		t.Fatalf("expected cache hit after set")
	}

	if string(data) != "bar" || statusCode != http.StatusOK {
		t.Fatalf("got data=%q statusCode=%d, want data=%q statusCode=%d", data, statusCode, "bar", http.StatusOK)
	}
}

func TestResponseCacheExpiry(t *testing.T) {
	t.Parallel()

	cache := newResponseCache(10 * time.Millisecond)

	cache.set("/foo", []byte("bar"), http.StatusOK)

	if _, _, found := cache.get("/foo"); !found {
		t.Fatalf("expected cache hit immediately after set")
	}

	time.Sleep(20 * time.Millisecond)

	if _, _, found := cache.get("/foo"); found {
		t.Fatalf("expected cache miss after TTL expiry")
	}
}

func TestResponseCacheZeroTTLDisablesCaching(t *testing.T) {
	t.Parallel()

	cache := newResponseCache(0)

	cache.set("/foo", []byte("bar"), http.StatusOK)

	if _, _, found := cache.get("/foo"); found {
		t.Fatalf("expected zero TTL to disable caching")
	}
}

func TestResponseCacheDistinctKeys(t *testing.T) {
	t.Parallel()

	cache := newResponseCache(time.Minute)

	cache.set("/foo", []byte("foo-data"), http.StatusOK)
	cache.set("/bar", []byte("bar-data"), http.StatusOK)

	fooData, _, fooFound := cache.get("/foo")
	barData, _, barFound := cache.get("/bar")

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

	for i := range numKeys {
		keys[i] = fmt.Sprintf("/key-%d", i)
		values[i] = fmt.Appendf(nil, "value-for-key-%d", i)
	}

	var waitGroup sync.WaitGroup

	for g := range numGoroutines {
		waitGroup.Add(1)

		go func(seed int) {
			defer waitGroup.Done()

			for op := range numOpsPerG {
				idx := (seed + op) % numKeys

				if op%2 == 0 {
					cache.set(keys[idx], values[idx], http.StatusOK)

					continue
				}

				data, statusCode, found := cache.get(keys[idx])
				if !found {
					continue
				}

				if statusCode != http.StatusOK || string(data) != string(values[idx]) {
					t.Errorf("get(%q) = (%q, %d), want value for that key", keys[idx], data, statusCode)
				}
			}
		}(g)
	}

	waitGroup.Wait()
}
