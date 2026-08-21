package health

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckAll_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := New([]string{srv.URL}, 10*time.Second)
	results := c.CheckAll()
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Err != nil {
		t.Fatalf("unexpected error: %v", r.Err)
	}
	if r.StatusCode != 200 {
		t.Errorf("expected 200, got %d", r.StatusCode)
	}
	if r.LatencyMs < 0 {
		t.Errorf("expected non-negative latency, got %d", r.LatencyMs)
	}
}

func TestCheckAll_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New([]string{srv.URL}, 10*time.Second)
	results := c.CheckAll()
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Err != nil {
		t.Fatalf("unexpected error: %v", r.Err)
	}
	if r.StatusCode != 500 {
		t.Errorf("expected 500, got %d", r.StatusCode)
	}
}

func TestCheckAll_MultipleEndpoints(t *testing.T) {
	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv1.Close()
	defer srv2.Close()

	c := New([]string{srv1.URL, srv2.URL}, 10*time.Second)
	results := c.CheckAll()
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].StatusCode != 200 {
		t.Errorf("expected 200, got %d", results[0].StatusCode)
	}
	if results[1].StatusCode != 503 {
		t.Errorf("expected 503, got %d", results[1].StatusCode)
	}
}

func TestCheckAll_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New([]string{srv.URL}, 10*time.Second)
	c.Timeout = 100 * time.Millisecond
	c.Client.Timeout = 100 * time.Millisecond

	results := c.CheckAll()
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestOnSample_Called(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	var samples []HealthSample
	c := New([]string{srv.URL}, 10*time.Second)
	c.OnSample = func(s HealthSample) { samples = append(samples, s) }
	c.CheckAll()

	if len(samples) != 1 {
		t.Fatalf("expected 1 sample, got %d", len(samples))
	}
	if samples[0].Endpoint != srv.URL {
		t.Errorf("unexpected endpoint: %s", samples[0].Endpoint)
	}
}

func TestCheckAll_CacheWithinTTL(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New([]string{srv.URL}, 10*time.Second)
	c.CacheTTL = time.Minute

	c.CheckAll()
	r2 := c.CheckAll()
	if hits != 1 {
		t.Fatalf("expected 1 HTTP hit within TTL, got %d", hits)
	}
	if len(r2) != 1 || r2[0].StatusCode != 200 {
		t.Fatalf("cached result wrong: %+v", r2)
	}
}

func TestCheckAll_CacheExpires(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New([]string{srv.URL}, 10*time.Second)
	c.CacheTTL = 30 * time.Millisecond

	c.CheckAll()
	time.Sleep(50 * time.Millisecond)
	c.CheckAll()
	if hits != 2 {
		t.Fatalf("expected 2 HTTP hits after TTL expiry, got %d", hits)
	}
}

func TestCheckAll_CacheDisabled(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New([]string{srv.URL}, 10*time.Second)
	c.CacheTTL = 0 // disabled

	c.CheckAll()
	c.CheckAll()
	if hits != 2 {
		t.Fatalf("expected 2 HTTP hits with cache disabled, got %d", hits)
	}
}

func TestCheckAll_CacheDoesNotRefireOnSample(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	var samples []HealthSample
	c := New([]string{srv.URL}, 10*time.Second)
	c.CacheTTL = time.Minute
	c.OnSample = func(s HealthSample) { samples = append(samples, s) }

	c.CheckAll()
	c.CheckAll()
	if hits != 1 {
		t.Fatalf("expected 1 HTTP hit, got %d", hits)
	}
	if len(samples) != 1 {
		t.Fatalf("expected OnSample to fire only on real checks, got %d samples", len(samples))
	}
}
