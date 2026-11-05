# Khoorooth Router 🐓

**Khoorooth Router Lite** — a low-footprint OpenAI-compatible reverse proxy/gateway prototype, written in Go.

> This is a new lightweight gateway project, not a full feature-equivalent port of 9Router. It forwards requests to an upstream OpenAI-compatible API; it does not implement provider aggregation, dashboard, account management, or billing.

## Features
- `POST /v1/chat/completions` and `GET /v1/models` proxying
- Streaming/SSE passthrough using Go's standard reverse proxy
- Optional client API key authentication
- Upstream API key injection
- Request body limit, concurrent request cap, per-IP token-bucket rate limit
- Health endpoint: `GET /healthz`
- Minimal scratch Docker image and GitHub Actions build workflow

## Quick start
Requires Go 1.22+.

```bash
export UPSTREAM_URL=https://api.openai.com
export UPSTREAM_API_KEY='your-provider-key'
export KHOOROOTH_API_KEY='your-client-key'
go run ./cmd/khoorooth
```

```bash
curl http://localhost:8080/v1/models \
  -H "Authorization: Bearer your-client-key"
```

## Docker
Copy `.env.example` to `.env`, fill in the secrets, then:

```bash
docker compose up -d --build
```

The compose file binds the service to localhost (127.0.0.1:8080) intentionally. Put a TLS reverse proxy in front before public exposure.

## Environment variables
| Variable | Default | Meaning |
|---|---|---|
| `LISTEN_ADDR` | `:8080` | Listen address |
| `UPSTREAM_URL` | `https://api.openai.com` | Upstream API base URL |
| `UPSTREAM_API_KEY` | empty | Provider secret |
| `KHOOROOTH_API_KEY` | empty | Client key; empty disables client auth |
| `MAX_BODY_BYTES` | `1048576` | Request body cap (1 MiB) |
| `MAX_CONCURRENCY` | `4` | Active request limit |
| `RATE_LIMIT_RPS` | `3` | Per-IP token refill rate |
| `RATE_LIMIT_BURST` | `6` | Per-IP burst capacity |
| `UPSTREAM_TIMEOUT_SECONDS` | `120` | Upstream response-header timeout |

## RAM reality check
Go generally has lower baseline overhead than a full Node/Next.js stack, but **128 MB for the entire VPS (OS + Docker + service) with multiple public users is not a guaranteed safe target**. Streaming connections and concurrent requests still consume memory. Start with low concurrency, test RSS under real traffic, and consider running the binary directly without Docker on a minimal OS. Docker's memory limit is an OOM guard, not a promise that the process will fit.

## Security checklist before public use
- Set a long random client key and upstream provider key.
- Use TLS via Caddy/nginx or another trusted reverse proxy.
- Do not expose provider credentials to callers.
- Add durable user accounts, per-user quotas, usage tracking, and abuse controls before multi-user production.
- In-memory IP rate limits are reset on restart and are not a billing system.
- Restrict who can access the host and keep secrets out of git.

## License
MIT
