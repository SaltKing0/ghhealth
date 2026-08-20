# ghhealth

Shared health engine for the kami family — used by [kagutsuchi](https://github.com/SaltKing0/kagutsuchi),
[fūjin](https://github.com/SaltKing0/fujin), [raijin](https://github.com/SaltKing0/raijin).

## Features

- **statuspage** — client for GitHub's [atlassian statuspage](https://www.githubstatus.com/api/v2):
  `GetStatus()`, `GetComponents()`, `GetIncidents(page)`
- **health** — HTTP health checker with `CheckAll()` + optional `OnSample` callback

## Usage

```go
import "github.com/SaltKing0/ghhealth/statuspage"
import "github.com/SaltKing0/ghhealth/health"

sp := statuspage.NewClient("")
status, _ := sp.GetStatus()       // overall indicator
comps, _ := sp.GetComponents()    // all components
incs, _ := sp.GetIncidents(1)    // latest incidents

h := health.New([]string{"https://github.com", "https://api.github.com"}, 60*time.Second)
h.OnSample = func(s health.HealthSample) { log.Printf("%s → %d", s.Endpoint, s.StatusCode) }
results := h.CheckAll()
```

## License

MIT
