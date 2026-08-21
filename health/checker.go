package health

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// HealthSample is a single health-check observation.
type HealthSample struct {
	Endpoint   string
	StatusCode int
	LatencyMs  int64
	Err        error
	CheckedAt  time.Time
}

// Result is a single health-check observation.
type Result struct {
	Endpoint   string
	StatusCode int
	LatencyMs  int64
	Err        error
	CheckedAt  time.Time
}

// Checker periodically pings endpoints. Sample persistence is delegated to
// the OnSample callback so this module stays free of storage concerns.
type Checker struct {
	Endpoints []string
	Interval  time.Duration
	Timeout   time.Duration
	// CacheTTL controls how long CheckAll results are reused. Within the TTL,
	// CheckAll returns the previous results without hitting the network and
	// OnSample is NOT fired again (a sample was already recorded when the
	// check actually ran). 0 disables caching.
	CacheTTL time.Duration
	Client   *http.Client
	// OnSample is invoked for every completed check (may be nil).
	OnSample func(HealthSample)

	mu          sync.Mutex
	lastResults []Result
	lastChecked time.Time
}

// New creates a default Checker with a 30s result cache.
func New(endpoints []string, interval time.Duration) *Checker {
	return &Checker{
		Endpoints: endpoints,
		Interval:  interval,
		Timeout:   10 * time.Second,
		CacheTTL:  30 * time.Second,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CheckAll runs one health check against every endpoint, invokes OnSample for
// each, and returns the results. Results from the last real check are reused
// while the CacheTTL has not expired; the returned slice is a copy.
func (c *Checker) CheckAll() []Result {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.CacheTTL > 0 && !c.lastChecked.IsZero() && time.Since(c.lastChecked) < c.CacheTTL {
		return append([]Result(nil), c.lastResults...)
	}

	results := make([]Result, 0, len(c.Endpoints))
	for _, ep := range c.Endpoints {
		r := c.checkOne(ep)
		if c.OnSample != nil {
			c.OnSample(HealthSample{
				Endpoint:   ep,
				StatusCode: r.StatusCode,
				LatencyMs:  r.LatencyMs,
				Err:        r.Err,
				CheckedAt:  r.CheckedAt,
			})
		}
		results = append(results, r)
	}
	c.lastResults = append(c.lastResults[:0], results...)
	c.lastChecked = time.Now()
	return results
}

func (c *Checker) checkOne(endpoint string) Result {
	start := time.Now()
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return Result{Endpoint: endpoint, Err: fmt.Errorf("create request: %w", err), CheckedAt: start}
	}
	req.Header.Set("User-Agent", "ghhealth/0.1.0")

	resp, err := c.Client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return Result{Endpoint: endpoint, Err: fmt.Errorf("request failed: %w", err), LatencyMs: latency, CheckedAt: start}
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) // drain for keep-alive

	return Result{
		Endpoint:   endpoint,
		StatusCode: resp.StatusCode,
		LatencyMs:  latency,
		CheckedAt:  start,
	}
}
