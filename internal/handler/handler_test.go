package handler

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestGetCacheTTL(t *testing.T) {
	tests := map[string]struct {
		envValue string
		want     time.Duration
	}{
		"unset uses default":    {envValue: "", want: defaultCacheTTL},
		"valid seconds":         {envValue: "30", want: 30 * time.Second},
		"zero disables caching": {envValue: "0", want: 0},
		"invalid falls back":    {envValue: "not-a-number", want: defaultCacheTTL},
		"negative falls back":   {envValue: "-5", want: defaultCacheTTL},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv(envCacheTTLSeconds, testCase.envValue)

			got := getCacheTTL(testLogger())
			if got != testCase.want {
				t.Fatalf("getCacheTTL() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestApplyClientRateLimits(t *testing.T) {
	t.Parallel()

	config := &rest.Config{}

	applyClientRateLimits(config)

	if config.QPS != kubernetesClientQPS {
		t.Errorf("QPS = %v, want %v", config.QPS, kubernetesClientQPS)
	}

	if config.Burst != kubernetesClientBurst {
		t.Errorf("Burst = %v, want %v", config.Burst, kubernetesClientBurst)
	}
}

func newTestHandler(t *testing.T, ttl time.Duration, upstream http.Handler) *OIDCDiscoveryProxyHandler {
	t.Helper()

	server := httptest.NewServer(upstream)
	t.Cleanup(server.Close)

	client, err := kubernetes.NewForConfig(&rest.Config{Host: server.URL})
	if err != nil {
		t.Fatalf("failed to create test Kubernetes client: %v", err)
	}

	return &OIDCDiscoveryProxyHandler{
		client: client,
		logger: testLogger(),
		cache:  newResponseCache(ttl),
	}
}

func TestHandleCachesSuccessfulResponses(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32

	upstream := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"ok":true}`))
	})

	h := newTestHandler(t, time.Minute, upstream)

	for i := range 3 {
		data, statusCode, err := h.handle(context.Background(), "/foo")
		if err != nil {
			t.Fatalf("unexpected error on call %d: %v", i, err)
		}

		if statusCode != http.StatusOK || string(data) != `{"ok":true}` {
			t.Fatalf("call %d: got statusCode=%d data=%q", i, statusCode, data)
		}
	}

	if got := hits.Load(); got != 1 {
		t.Fatalf("upstream hits = %d, want 1 (subsequent calls should be served from cache)", got)
	}
}

func TestHandleDoesNotCacheErrorResponses(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32

	upstream := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(`boom`))
	})

	h := newTestHandler(t, time.Minute, upstream)

	for i := range 2 {
		_, _, err := h.handle(context.Background(), "/foo")
		if err == nil {
			t.Fatalf("call %d: expected error from upstream failure", i)
		}
	}

	if got := hits.Load(); got != 2 {
		t.Fatalf("upstream hits = %d, want 2 (error responses must not be cached)", got)
	}
}

func TestHandleExpiresCacheAfterTTL(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32

	upstream := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"ok":true}`))
	})

	h := newTestHandler(t, 10*time.Millisecond, upstream)

	_, _, err := h.handle(context.Background(), "/foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	_, _, err = h.handle(context.Background(), "/foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := hits.Load(); got != 2 {
		t.Fatalf("upstream hits = %d, want 2 (cache entry should have expired)", got)
	}
}

func TestHandleCachesPathsIndependently(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32

	upstream := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		hits.Add(1)
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(request.URL.Path))
	})

	h := newTestHandler(t, time.Minute, upstream)

	data1, _, err := h.handle(context.Background(), OpenIDConfigPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data2, _, err := h.handle(context.Background(), JWKSPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(data1) != OpenIDConfigPath || string(data2) != JWKSPath {
		t.Fatalf("got data1=%q data2=%q, want distinct per-path responses", data1, data2)
	}

	if got := hits.Load(); got != 2 {
		t.Fatalf("upstream hits = %d, want 2 (one per distinct path)", got)
	}
}

// TestHandleConcurrentRequests fires many concurrent requests at a handler sharing a single
// cache, mixing paths that hit and miss. Run with -race to confirm handle() and the underlying
// cache are safe under concurrent read/write access.
func TestHandleConcurrentRequests(t *testing.T) {
	t.Parallel()

	upstream := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(request.URL.Path))
	})

	h := newTestHandler(t, time.Minute, upstream)

	paths := []string{OpenIDConfigPath, JWKSPath, "/other-path"}

	const numGoroutines = 50

	var waitGroup sync.WaitGroup

	for g := range numGoroutines {
		waitGroup.Add(1)

		go func(seed int) {
			defer waitGroup.Done()

			path := paths[seed%len(paths)]

			data, statusCode, err := h.handle(context.Background(), path)
			if err != nil {
				t.Errorf("handle(%q) returned error: %v", path, err)

				return
			}

			if statusCode != http.StatusOK || string(data) != path {
				t.Errorf("handle(%q) = (%q, %d), want (%q, %d)", path, data, statusCode, path, http.StatusOK)
			}
		}(g)
	}

	waitGroup.Wait()
}
