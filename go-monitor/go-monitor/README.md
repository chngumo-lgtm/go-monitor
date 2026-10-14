# go-monitor

A small service-health monitor with a REST API, written in Go using only the
standard library. Register services, then trigger health checks on demand.

## Endpoints
| Method | Path | Description |
|---|---|---|
| GET | `/healthz` | Liveness probe |
| GET | `/services` | List services |
| POST | `/services` | Register `{"name","url"}` |
| GET | `/services/{name}` | Get one service |
| DELETE | `/services/{name}` | Remove a service |
| POST | `/services/{name}/check` | Probe the URL and record up/down |

## Run
```bash
go run ./cmd/server          # listens on :8080 (override with ADDR)
curl -X POST localhost:8080/services -d '{"name":"web","url":"https://example.com"}'
curl -X POST localhost:8080/services/web/check
```

## Test
```bash
go test ./... -cover
```
Requires Go 1.22+ (method-based route patterns).
